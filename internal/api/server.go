package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"

	"github.com/josepht96/scout/internal/scheduler"
	"github.com/josepht96/scout/internal/storage"
	"github.com/josepht96/scout/internal/watcher"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server handles HTTP requests
type Server struct {
	storage   *storage.Storage
	scheduler *scheduler.Scheduler
	watcher   *watcher.CollectionWatcher
	port      int
}

// Config contains server configuration
type Config struct {
	Storage   *storage.Storage
	Scheduler *scheduler.Scheduler
	Watcher   *watcher.CollectionWatcher
	Port      int
}

// NewServer creates a new HTTP server
func NewServer(config Config) *Server {
	return &Server{
		storage:   config.Storage,
		scheduler: config.Scheduler,
		watcher:   config.Watcher,
		port:      config.Port,
	}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	mux := http.NewServeMux()

	// Static UI
	mux.HandleFunc("/", s.handleIndex)
	mux.HandleFunc("/favicon.svg", s.handleFavicon)

	// API endpoints
	mux.HandleFunc("/api/results", s.handleResults)
	mux.HandleFunc("/api/history", s.handleHistory)
	mux.HandleFunc("/api/collections", s.handleCollections)
	mux.HandleFunc("/api/run", s.handleRun)
	mux.HandleFunc("/api/stats", s.handleStats)

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	// Prometheus metrics
	mux.Handle("/metrics", promhttp.Handler())

	addr := fmt.Sprintf(":%d", s.port)
	log.Printf("Starting HTTP server on %s", addr)

	return http.ListenAndServe(addr, s.loggingMiddleware(mux))
}

// loggingMiddleware logs all HTTP requests
func (s *Server) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		log.Printf("%s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
	})
}

// handleIndex serves the static UI
func (s *Server) handleIndex(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	// Try to read from filesystem
	data, err := os.ReadFile("web/index.html")
	if err != nil {
		// If not found, serve a simple default page
		w.Header().Set("Content-Type", "text/html")
		w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Scout</title></head>
<body>
<h1>Scout - Postman Test Monitor</h1>
<p>UI not yet loaded. Access <a href="/api/results">/api/results</a> for JSON data.</p>
</body>
</html>`))
		return
	}

	w.Header().Set("Content-Type", "text/html")
	w.Write(data)
}

// handleFavicon serves the favicon
func (s *Server) handleFavicon(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("web/favicon.svg")
	if err != nil {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "image/svg+xml")
	w.Write(data)
}

// handleResults returns the latest test results grouped by environment
func (s *Server) handleResults(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get collection groups from watcher
	groups, err := s.watcher.ScanGroups()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error scanning groups: %v", err), http.StatusInternalServerError)
		return
	}

	// Get results from storage (as ungrouped)
	storageResults, err := s.storage.GetLatestResults()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching results: %v", err), http.StatusInternalServerError)
		return
	}

	// Build a map of composite key to collection result for easy lookup
	resultsByCompositeKey := make(map[string]storage.CollectionResult)
	for _, envGroup := range storageResults.EnvironmentGroups {
		for _, cr := range envGroup.Collections {
			resultsByCompositeKey[cr.Collection.CompositeKey] = cr
		}
	}

	// Build grouped results
	var environmentGroups []storage.EnvironmentGroup
	for _, group := range groups {
		envGroup := storage.EnvironmentGroup{
			Directory:   group.Directory,
			Collections: []storage.CollectionResult{},
		}

		// Add environment info if present
		if group.Environment != nil {
			envGroup.Environment = &storage.EnvironmentInfo{
				Name:     group.Environment.Name,
				FileName: group.Environment.FileName,
				Path:     group.Environment.Path,
			}
		}

		// Match collections to results
		for _, col := range group.Collections {
			// Generate composite key for this collection in this group
			var envName *string
			if group.Environment != nil {
				envName = &group.Environment.Name
			}
			compositeKey, dir, env, collName := scheduler.GenerateCompositeKey(group.Directory, envName, filepath.Base(col.FullPath))

			if result, found := resultsByCompositeKey[compositeKey]; found {
				envGroup.Collections = append(envGroup.Collections, result)
			} else {
				// Collection file exists but no execution yet
				// Create a placeholder with just the collection info
				cr := storage.CollectionResult{
					Collection: storage.Collection{
						Name:            col.Name,
						FilePath:        col.FullPath,
						CompositeKey:    compositeKey,
						DirectoryName:   dir,
						EnvironmentName: env,
						CollectionName:  collName,
					},
					Execution:            nil,
					LastSuccessExecution: nil,
					Results:              []storage.TestResult{},
				}
				envGroup.Collections = append(envGroup.Collections, cr)
			}
		}

		environmentGroups = append(environmentGroups, envGroup)
	}

	response := &storage.LatestResults{
		EnvironmentGroups: environmentGroups,
		UpdatedAt:         storageResults.UpdatedAt,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// handleHistory returns historical execution data for a collection
func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Get collection_id from query params
	collectionIDStr := r.URL.Query().Get("collection_id")
	if collectionIDStr == "" {
		http.Error(w, "collection_id parameter is required", http.StatusBadRequest)
		return
	}

	collectionID, err := strconv.Atoi(collectionIDStr)
	if err != nil {
		http.Error(w, "Invalid collection_id", http.StatusBadRequest)
		return
	}

	// Get limit (default 50, max 200)
	limit := 50
	if limitStr := r.URL.Query().Get("limit"); limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil {
			limit = l
			if limit > 200 {
				limit = 200
			}
		}
	}

	history, err := s.storage.GetExecutionHistory(collectionID, limit)
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching history: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

// handleCollections returns all collections
func (s *Server) handleCollections(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	collections, err := s.storage.GetAllCollections()
	if err != nil {
		http.Error(w, fmt.Sprintf("Error fetching collections: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(collections)
}

// handleRun triggers an immediate test run
func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	s.scheduler.RunNow()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status":  "ok",
		"message": "Test execution triggered",
	})
}

// handleStats returns scheduler statistics
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	stats := s.scheduler.GetStats()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// handleHealth returns health status
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
	})
}
