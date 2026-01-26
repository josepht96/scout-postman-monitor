package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/josepht96/scout/internal/api"
	"github.com/josepht96/scout/internal/executor"
	"github.com/josepht96/scout/internal/metrics"
	"github.com/josepht96/scout/internal/scheduler"
	"github.com/josepht96/scout/internal/storage"
	"github.com/josepht96/scout/internal/watcher"
)

func main() {
	log.Println("Starting Scout - Postman Test Monitor")

	// Load configuration from environment
	config := loadConfig()

	// Initialize database
	log.Printf("Connecting to database: %s", maskConnectionString(config.DatabaseURL))
	store, err := storage.NewStorage(config.DatabaseURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer store.Close()

	// Run migrations
	log.Println("Running database migrations...")
	if err := store.RunMigrations(""); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// Get absolute path to newman executor
	executableDir, err := os.Executable()
	if err != nil {
		log.Fatalf("Failed to get executable directory: %v", err)
	}
	baseDir := filepath.Dir(executableDir)

	// In development, use relative paths
	newmanScript := config.NewmanScriptPath
	if newmanScript == "" {
		newmanScript = filepath.Join(baseDir, "newman", "executor.js")
		// Try relative path for development
		if _, err := os.Stat(newmanScript); os.IsNotExist(err) {
			newmanScript = "newman/executor.js"
		}
	}

	// Initialize components
	log.Printf("Newman script path: %s", newmanScript)
	exec := executor.NewNewmanExecutor(newmanScript)

	// Check if Node.js is available
	if !exec.IsAvailable() {
		log.Fatal("Node.js is not available. Please install Node.js to run Scout.")
	}

	version, _ := exec.GetVersion()
	log.Printf("Node.js version: %s", version)

	log.Printf("Watching collections directory: %s", config.CollectionsDir)
	watch := watcher.NewCollectionWatcher(config.CollectionsDir)

	// Initialize Prometheus metrics
	metricsExporter := metrics.NewPrometheusExporter()

	// Initialize scheduler
	sched := scheduler.NewScheduler(scheduler.Config{
		Storage:        store,
		Executor:       exec,
		Watcher:        watch,
		Interval:       config.Interval,
		MetricsUpdater: metricsExporter,
	})

	// Start scheduler
	sched.Start()

	// Initialize HTTP server
	server := api.NewServer(api.Config{
		Storage:   store,
		Scheduler: sched,
		Watcher:   watch,
		Port:      config.Port,
	})

	// Start HTTP server in a goroutine
	go func() {
		if err := server.Start(); err != nil {
			log.Fatalf("Failed to start HTTP server: %v", err)
		}
	}()

	log.Printf("Scout is running on http://localhost:%d", config.Port)
	log.Println("Press Ctrl+C to stop")

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down Scout...")

	// Create shutdown context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Stop scheduler
	sched.Stop()

	// Wait for graceful shutdown
	<-ctx.Done()

	log.Println("Scout stopped")
}

// Config holds application configuration
type Config struct {
	DatabaseURL       string
	CollectionsDir    string
	NewmanScriptPath  string
	Interval          time.Duration
	Port              int
}

// loadConfig loads configuration from environment variables
func loadConfig() Config {
	config := Config{
		DatabaseURL:      getEnv("DATABASE_URL", "postgres://postgres:postgres@localhost:5432/scout?sslmode=disable"),
		CollectionsDir:   getEnv("COLLECTIONS_DIR", "collections"),
		NewmanScriptPath: getEnv("NEWMAN_SCRIPT_PATH", ""),
		Interval:         getDurationEnv("INTERVAL", 60*time.Second),
		Port:             getIntEnv("PORT", 8080),
	}

	// Ensure collections directory exists
	if err := os.MkdirAll(config.CollectionsDir, 0755); err != nil {
		log.Fatalf("Failed to create collections directory: %v", err)
	}

	return config
}

// getEnv gets an environment variable with a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv gets an integer environment variable with a default value
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intValue, err := strconv.Atoi(value); err == nil {
			return intValue
		}
	}
	return defaultValue
}

// getDurationEnv gets a duration environment variable with a default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}

// maskConnectionString masks sensitive parts of connection string for logging
func maskConnectionString(connStr string) string {
	// Simple masking - just show it's configured
	if connStr != "" {
		return "[CONFIGURED]"
	}
	return "[NOT SET]"
}
