package scheduler

import (
	"context"
	"log"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/josepht96/scout/internal/executor"
	"github.com/josepht96/scout/internal/storage"
	"github.com/josepht96/scout/internal/watcher"
)

// GenerateCompositeKey creates a unique composite key from directory, environment, and collection names
// Format: {directory}_{environment}_{collection} (all lowercase)
// If no environment: {directory}_env_{collection}
func GenerateCompositeKey(directoryName string, environmentName *string, collectionFileName string) (compositeKey, directory, environment, collection string) {
	// Extract collection name from filename (strip .postman_collection.json)
	collectionName := strings.TrimSuffix(collectionFileName, ".postman_collection.json")

	// Use environment name or "env" as placeholder
	envName := "env"
	if environmentName != nil && *environmentName != "" {
		envName = *environmentName
	}

	// Convert all to lowercase and join with underscores
	dir := strings.ToLower(directoryName)
	env := strings.ToLower(envName)
	col := strings.ToLower(collectionName)

	key := dir + "_" + env + "_" + col

	return key, dir, env, col
}

// Scheduler manages periodic execution of Postman collections
type Scheduler struct {
	storage        *storage.Storage
	executor       *executor.NewmanExecutor
	watcher        *watcher.CollectionWatcher
	interval       time.Duration
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	metricsUpdater MetricsUpdater
	mu             sync.RWMutex
	lastRunTime    time.Time
	totalRuns      int
	failedRuns     int
}

// MetricsUpdater is an interface for updating metrics
type MetricsUpdater interface {
	UpdateMetrics(*storage.LatestResults)
}

// Config contains scheduler configuration
type Config struct {
	Storage        *storage.Storage
	Executor       *executor.NewmanExecutor
	Watcher        *watcher.CollectionWatcher
	Interval       time.Duration
	MetricsUpdater MetricsUpdater
}

// NewScheduler creates a new scheduler
func NewScheduler(config Config) *Scheduler {
	ctx, cancel := context.WithCancel(context.Background())
	return &Scheduler{
		storage:        config.Storage,
		executor:       config.Executor,
		watcher:        config.Watcher,
		interval:       config.Interval,
		ctx:            ctx,
		cancel:         cancel,
		metricsUpdater: config.MetricsUpdater,
	}
}

// Start starts the scheduler
func (s *Scheduler) Start() {
	log.Printf("Starting scheduler with interval: %v", s.interval)

	// Run once immediately
	s.runOnce()

	// Start ticker for periodic execution
	s.wg.Add(1)
	go func() {
		defer s.wg.Done()

		ticker := time.NewTicker(s.interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				s.runOnce()
			case <-s.ctx.Done():
				log.Println("Scheduler stopped")
				return
			}
		}
	}()
}

// Stop stops the scheduler
func (s *Scheduler) Stop() {
	log.Println("Stopping scheduler...")
	s.cancel()
	s.wg.Wait()
	log.Println("Scheduler stopped")
}

// runOnce executes all collections once
func (s *Scheduler) runOnce() {
	s.mu.Lock()
	s.lastRunTime = time.Now()
	s.totalRuns++
	s.mu.Unlock()

	log.Println("Starting test execution cycle")

	// Scan for collection groups
	groups, err := s.watcher.ScanGroups()
	if err != nil {
		log.Printf("Error scanning for collection groups: %v", err)
		s.incrementFailedRuns()
		return
	}

	if len(groups) == 0 {
		log.Printf("No collection groups found in %s", s.watcher.GetDirectory())
		return
	}

	totalCollections := 0
	for _, group := range groups {
		totalCollections += len(group.Collections)
	}

	log.Printf("Found %d group(s) with %d total collection(s)", len(groups), totalCollections)

	// Execute collections from each group
	var wg sync.WaitGroup
	for _, group := range groups {
		for _, col := range group.Collections {
			wg.Add(1)

			// Determine environment path for this collection
			var envPath *string
			var envName *string
			if group.Environment != nil {
				envPath = &group.Environment.FullPath
				// Extract environment name from filename (strip .postman_environment.json)
				name := strings.TrimSuffix(group.Environment.FileName, ".postman_environment.json")
				envName = &name
			}

			// Get directory name
			dirName := group.Directory

			go func(c watcher.CollectionFile, env *string, dir string, eName *string) {
				defer wg.Done()
				if err := s.executeCollection(c, env, dir, eName); err != nil {
					log.Printf("Error executing collection %s: %v", c.Name, err)
				}
			}(col, envPath, dirName, envName)
		}
	}

	// Wait for all executions to complete
	wg.Wait()

	// Update metrics
	if s.metricsUpdater != nil {
		results, err := s.storage.GetLatestResults()
		if err != nil {
			log.Printf("Error getting latest results for metrics: %v", err)
		} else {
			s.metricsUpdater.UpdateMetrics(results)
		}
	}

	log.Println("Test execution cycle completed")
}

// executeCollection executes a single collection with optional environment
func (s *Scheduler) executeCollection(col watcher.CollectionFile, environmentPath *string, directoryName string, environmentName *string) error {
	if environmentPath != nil {
		log.Printf("Executing collection: %s with environment", col.Name)
	} else {
		log.Printf("Executing collection: %s", col.Name)
	}

	startTime := time.Now()

	// Generate composite key and extract normalized components BEFORE execution
	// This ensures the executor receives the same normalized values used in the composite key
	compositeKey, dir, env, collName := GenerateCompositeKey(directoryName, environmentName, filepath.Base(col.FullPath))

	// Execute with Newman using normalized directory and environment names
	normalizedEnvName := &env
	if env == "env" {
		// If env is the placeholder "env", pass nil to executor
		normalizedEnvName = nil
	}
	result, err := s.executor.Execute(col.FullPath, environmentPath, dir, normalizedEnvName)
	if err != nil {
		log.Printf("Newman execution error for %s: %v", col.Name, err)
		// Continue to store the partial result if available
		if result == nil {
			s.incrementFailedRuns()
			return err
		}
	}

	// Debug logging
	log.Printf("[DEBUG] Composite key generation: dir=%s, env=%s, collection=%s -> key=%s", dir, env, collName, compositeKey)

	// Ensure collection exists in database with composite key
	dbCollection, err := s.storage.UpsertCollection(result.CollectionName, col.FullPath, compositeKey, dir, env, collName)
	if err != nil {
		log.Printf("Error upserting collection %s: %v", col.Name, err)
		s.incrementFailedRuns()
		return err
	}

	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, result.Timestamp)
	if err != nil {
		timestamp = startTime
	}

	// Create execution record
	execution := &storage.TestExecution{
		CollectionID:   dbCollection.ID,
		CollectionName: result.CollectionName,
		StartedAt:      timestamp,
		CompletedAt:    timestamp.Add(time.Duration(result.TotalDurationMs) * time.Millisecond),
		DurationMs:     result.TotalDurationMs,
		TotalTests:     result.Summary.Total,
		PassedTests:    result.Summary.Passed,
		FailedTests:    result.Summary.Failed,
		Error:          result.Error,
	}

	if err := s.storage.CreateTestExecution(execution); err != nil {
		log.Printf("Error creating test execution for %s: %v", col.Name, err)
		s.incrementFailedRuns()
		return err
	}

	// Store test results
	for _, test := range result.Tests {
		testResult := &storage.TestResult{
			ExecutionID:   execution.ID,
			TestName:      test.Name,
			ExecutionName: &test.ExecutionName,
			Status:        "unknown",
			Passed:        test.Passed,
			Error:         test.Error,
		}

		// Try to find matching execution info
		for _, exec := range result.Executions {
			if exec.Name == test.ExecutionName {
				testResult.URL = &exec.URL
				testResult.Method = &exec.Method
				testResult.Status = exec.Status
				testResult.StatusCode = exec.StatusCode
				testResult.ResponseTimeMs = exec.ResponseTime
				break
			}
		}

		if err := s.storage.CreateTestResult(testResult); err != nil {
			log.Printf("Error creating test result for %s: %v", test.Name, err)
		}
	}

	duration := time.Since(startTime)
	status := "SUCCESS"
	if result.Summary.Failed > 0 && result.Summary.Passed > 0 {
		status = "PARTIAL"
	} else if result.Summary.Failed > 0 {
		status = "FAILED"
	}

	log.Printf("Collection %s completed in %v - Status: %s (Passed: %d, Failed: %d)",
		col.Name, duration, status, result.Summary.Passed, result.Summary.Failed)

	return nil
}

// incrementFailedRuns increments the failed runs counter
func (s *Scheduler) incrementFailedRuns() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.failedRuns++
}

// GetStats returns scheduler statistics
func (s *Scheduler) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"last_run_time": s.lastRunTime,
		"total_runs":    s.totalRuns,
		"failed_runs":   s.failedRuns,
		"interval":      s.interval.String(),
	}
}

// RunNow triggers an immediate execution cycle
func (s *Scheduler) RunNow() {
	go s.runOnce()
}
