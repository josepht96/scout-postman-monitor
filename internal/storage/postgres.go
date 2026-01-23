package storage

import (
	"database/sql"
	"fmt"
	"time"

	_ "github.com/lib/pq"
)

// Storage provides database operations for Scout
type Storage struct {
	db *sql.DB
}

// NewStorage creates a new Storage instance
func NewStorage(connectionString string) (*Storage, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %w", err)
	}

	// Set connection pool settings
	db.SetMaxOpenConns(25)
	db.SetMaxIdleConns(5)
	db.SetConnMaxLifetime(5 * time.Minute)

	return &Storage{db: db}, nil
}

// Close closes the database connection
func (s *Storage) Close() error {
	return s.db.Close()
}

// UpsertCollection inserts or updates a collection
func (s *Storage) UpsertCollection(name, filePath, compositeKey, directoryName, environmentName, collectionName string) (*Collection, error) {
	query := `
		INSERT INTO collections (name, file_path, composite_key, directory_name, environment_name, collection_name, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		ON CONFLICT (composite_key)
		DO UPDATE SET name = EXCLUDED.name, updated_at = EXCLUDED.updated_at
		RETURNING id, name, file_path, composite_key, directory_name, environment_name, collection_name, created_at, updated_at
	`

	now := time.Now()
	var c Collection
	err := s.db.QueryRow(query, name, filePath, compositeKey, directoryName, environmentName, collectionName, now, now).Scan(
		&c.ID, &c.Name, &c.FilePath, &c.CompositeKey, &c.DirectoryName, &c.EnvironmentName, &c.CollectionName, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to upsert collection: %w", err)
	}

	return &c, nil
}

// GetCollectionByPath retrieves a collection by file path
func (s *Storage) GetCollectionByPath(filePath string) (*Collection, error) {
	query := `SELECT id, name, file_path, created_at, updated_at FROM collections WHERE file_path = $1`

	var c Collection
	err := s.db.QueryRow(query, filePath).Scan(
		&c.ID, &c.Name, &c.FilePath, &c.CreatedAt, &c.UpdatedAt,
	)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get collection: %w", err)
	}

	return &c, nil
}

// GetAllCollections retrieves all collections
func (s *Storage) GetAllCollections() ([]Collection, error) {
	query := `SELECT id, name, file_path, composite_key, directory_name, environment_name, collection_name, created_at, updated_at FROM collections ORDER BY directory_name, environment_name, collection_name`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query collections: %w", err)
	}
	defer rows.Close()

	var collections []Collection
	for rows.Next() {
		var c Collection
		if err := rows.Scan(&c.ID, &c.Name, &c.FilePath, &c.CompositeKey, &c.DirectoryName, &c.EnvironmentName, &c.CollectionName, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan collection: %w", err)
		}
		collections = append(collections, c)
	}

	return collections, rows.Err()
}

// CreateTestExecution creates a new test execution record
func (s *Storage) CreateTestExecution(exec *TestExecution) error {
	query := `
		INSERT INTO test_executions (
			collection_id, collection_name, started_at, completed_at,
			duration_ms, total_tests, passed_tests, failed_tests, error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, created_at
	`

	err := s.db.QueryRow(
		query,
		exec.CollectionID,
		exec.CollectionName,
		exec.StartedAt,
		exec.CompletedAt,
		exec.DurationMs,
		exec.TotalTests,
		exec.PassedTests,
		exec.FailedTests,
		exec.Error,
	).Scan(&exec.ID, &exec.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create test execution: %w", err)
	}

	return nil
}

// CreateTestResult creates a new test result record
func (s *Storage) CreateTestResult(result *TestResult) error {
	query := `
		INSERT INTO test_results (
			execution_id, test_name, execution_name, url, method,
			status, status_code, response_time_ms, passed, error
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id, created_at
	`

	err := s.db.QueryRow(
		query,
		result.ExecutionID,
		result.TestName,
		result.ExecutionName,
		result.URL,
		result.Method,
		result.Status,
		result.StatusCode,
		result.ResponseTimeMs,
		result.Passed,
		result.Error,
	).Scan(&result.ID, &result.CreatedAt)

	if err != nil {
		return fmt.Errorf("failed to create test result: %w", err)
	}

	return nil
}

// GetLatestExecutions retrieves the latest execution for each collection
func (s *Storage) GetLatestExecutions() ([]TestExecution, error) {
	query := `
		SELECT id, collection_id, collection_name, started_at, completed_at,
		       duration_ms, total_tests, passed_tests, failed_tests, error, created_at
		FROM latest_test_executions
		ORDER BY collection_name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, fmt.Errorf("failed to query latest executions: %w", err)
	}
	defer rows.Close()

	var executions []TestExecution
	for rows.Next() {
		var e TestExecution
		if err := rows.Scan(
			&e.ID, &e.CollectionID, &e.CollectionName, &e.StartedAt, &e.CompletedAt,
			&e.DurationMs, &e.TotalTests, &e.PassedTests, &e.FailedTests, &e.Error, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, e)
	}

	return executions, rows.Err()
}

// GetLastSuccessfulExecution retrieves the last successful execution for a collection
func (s *Storage) GetLastSuccessfulExecution(collectionID int) (*TestExecution, error) {
	query := `
		SELECT id, collection_id, collection_name, started_at, completed_at,
		       duration_ms, total_tests, passed_tests, failed_tests, error, created_at
		FROM test_executions
		WHERE collection_id = $1
		  AND failed_tests = 0
		  AND total_tests > 0
		ORDER BY started_at DESC
		LIMIT 1
	`

	var e TestExecution
	err := s.db.QueryRow(query, collectionID).Scan(
		&e.ID, &e.CollectionID, &e.CollectionName, &e.StartedAt, &e.CompletedAt,
		&e.DurationMs, &e.TotalTests, &e.PassedTests, &e.FailedTests, &e.Error, &e.CreatedAt,
	)

	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No successful execution found
		}
		return nil, fmt.Errorf("failed to query last successful execution: %w", err)
	}

	return &e, nil
}

// GetTestResultsByExecutionID retrieves all test results for a given execution
func (s *Storage) GetTestResultsByExecutionID(executionID int) ([]TestResult, error) {
	query := `
		SELECT id, execution_id, test_name, execution_name, url, method,
		       status, status_code, response_time_ms, passed, error, created_at
		FROM test_results
		WHERE execution_id = $1
		ORDER BY test_name
	`

	rows, err := s.db.Query(query, executionID)
	if err != nil {
		return nil, fmt.Errorf("failed to query test results: %w", err)
	}
	defer rows.Close()

	var results []TestResult
	for rows.Next() {
		var r TestResult
		if err := rows.Scan(
			&r.ID, &r.ExecutionID, &r.TestName, &r.ExecutionName, &r.URL, &r.Method,
			&r.Status, &r.StatusCode, &r.ResponseTimeMs, &r.Passed, &r.Error, &r.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan test result: %w", err)
		}
		results = append(results, r)
	}

	return results, rows.Err()
}

// GetLatestResults retrieves the latest execution and results for all collections
func (s *Storage) GetLatestResults() (*LatestResults, error) {
	collections, err := s.GetAllCollections()
	if err != nil {
		return nil, err
	}

	executions, err := s.GetLatestExecutions()
	if err != nil {
		return nil, err
	}

	// Create a map of collection ID to execution
	execMap := make(map[int]*TestExecution)
	for i := range executions {
		execMap[executions[i].CollectionID] = &executions[i]
	}

	// Build collection results grouped by collection+environment
	var collectionResults []CollectionResult
	for _, exec := range executions {
		// Find the matching collection
		var matchingCol *Collection
		for _, col := range collections {
			if col.ID == exec.CollectionID {
				matchingCol = &col
				break
			}
		}
		if matchingCol == nil {
			continue // Skip if collection not found
		}

		cr := CollectionResult{
			Collection: *matchingCol,
			Execution:  &exec,
			Results:    []TestResult{},
		}

		// Get last successful execution for this collection
		lastSuccess, err := s.GetLastSuccessfulExecution(exec.CollectionID)
		if err != nil {
			return nil, err
		}
		cr.LastSuccessExecution = lastSuccess

		// Get test results for this execution
		testResults, err := s.GetTestResultsByExecutionID(exec.ID)
		if err != nil {
			return nil, err
		}
		cr.Results = testResults

		collectionResults = append(collectionResults, cr)
	}

	// Group collection results by directory and environment
	type groupKey struct {
		directory string
		envName   string
	}

	groupMap := make(map[groupKey][]CollectionResult)

	for _, cr := range collectionResults {
		key := groupKey{
			directory: cr.Collection.DirectoryName,
			envName:   cr.Collection.EnvironmentName,
		}

		groupMap[key] = append(groupMap[key], cr)
	}

	// Build environment groups
	var envGroups []EnvironmentGroup
	for key, collections := range groupMap {
		group := EnvironmentGroup{
			Directory:   key.directory,
			Collections: collections,
		}

		// Set environment info if available (use "env" for no-environment placeholder)
		if key.envName != "" && key.envName != "env" {
			group.Environment = &EnvironmentInfo{
				Name:     key.envName,
				FileName: key.envName + ".postman_environment.json",
				Path:     "", // Path not stored anymore
			}
		}

		envGroups = append(envGroups, group)
	}

	results := &LatestResults{
		EnvironmentGroups: envGroups,
		UpdatedAt:         time.Now(),
	}

	return results, nil
}

// GetExecutionHistory retrieves execution history for a collection
func (s *Storage) GetExecutionHistory(collectionID int, limit int) ([]TestExecution, error) {
	query := `
		SELECT id, collection_id, collection_name, started_at, completed_at,
		       duration_ms, total_tests, passed_tests, failed_tests, error, created_at
		FROM test_executions
		WHERE collection_id = $1
		ORDER BY started_at DESC
		LIMIT $2
	`

	rows, err := s.db.Query(query, collectionID, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query execution history: %w", err)
	}
	defer rows.Close()

	var executions []TestExecution
	for rows.Next() {
		var e TestExecution
		if err := rows.Scan(
			&e.ID, &e.CollectionID, &e.CollectionName, &e.StartedAt, &e.CompletedAt,
			&e.DurationMs, &e.TotalTests, &e.PassedTests, &e.FailedTests, &e.Error, &e.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("failed to scan execution: %w", err)
		}
		executions = append(executions, e)
	}

	return executions, rows.Err()
}

// RunMigrations runs database migrations
func (s *Storage) RunMigrations(migrationsPath string) error {
	// Read and execute migration files
	upSQL := `
-- Collections table
CREATE TABLE IF NOT EXISTS collections (
    id SERIAL PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    file_path TEXT NOT NULL,
    composite_key VARCHAR(512) NOT NULL UNIQUE,
    directory_name VARCHAR(255) NOT NULL,
    environment_name VARCHAR(255) NOT NULL,
    collection_name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Add new columns to existing collections table
ALTER TABLE collections ADD COLUMN IF NOT EXISTS composite_key VARCHAR(512);
ALTER TABLE collections ADD COLUMN IF NOT EXISTS directory_name VARCHAR(255);
ALTER TABLE collections ADD COLUMN IF NOT EXISTS environment_name VARCHAR(255);
ALTER TABLE collections ADD COLUMN IF NOT EXISTS collection_name VARCHAR(255);

-- Add unique constraint on composite_key if it doesn't exist
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'collections_composite_key_key'
    ) THEN
        ALTER TABLE collections ADD CONSTRAINT collections_composite_key_key UNIQUE (composite_key);
    END IF;
END $$;

-- Drop unique constraint on file_path if it exists
DO $$
BEGIN
    IF EXISTS (
        SELECT 1 FROM pg_constraint WHERE conname = 'collections_file_path_key'
    ) THEN
        ALTER TABLE collections DROP CONSTRAINT collections_file_path_key;
    END IF;
END $$;

-- Test executions table
CREATE TABLE IF NOT EXISTS test_executions (
    id SERIAL PRIMARY KEY,
    collection_id INTEGER NOT NULL REFERENCES collections(id) ON DELETE CASCADE,
    collection_name VARCHAR(255) NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL,
    completed_at TIMESTAMP WITH TIME ZONE NOT NULL,
    duration_ms INTEGER NOT NULL,
    total_tests INTEGER NOT NULL DEFAULT 0,
    passed_tests INTEGER NOT NULL DEFAULT 0,
    failed_tests INTEGER NOT NULL DEFAULT 0,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_test_executions_collection_id ON test_executions(collection_id);
CREATE INDEX IF NOT EXISTS idx_test_executions_started_at ON test_executions(started_at DESC);

-- Test results table
CREATE TABLE IF NOT EXISTS test_results (
    id SERIAL PRIMARY KEY,
    execution_id INTEGER NOT NULL REFERENCES test_executions(id) ON DELETE CASCADE,
    test_name TEXT NOT NULL,
    execution_name VARCHAR(255),
    url TEXT,
    method VARCHAR(10),
    status VARCHAR(50) NOT NULL,
    status_code INTEGER,
    response_time_ms INTEGER,
    passed BOOLEAN NOT NULL,
    error TEXT,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX IF NOT EXISTS idx_test_results_execution_id ON test_results(execution_id);
CREATE INDEX IF NOT EXISTS idx_test_results_test_name ON test_results(test_name);

-- Latest results views
CREATE OR REPLACE VIEW latest_test_executions AS
SELECT DISTINCT ON (collection_id) *
FROM test_executions
ORDER BY collection_id, started_at DESC;

CREATE OR REPLACE VIEW latest_test_results AS
SELECT DISTINCT ON (tr.test_name, te.collection_id)
    tr.*,
    te.collection_id,
    te.collection_name,
    te.started_at as execution_started_at
FROM test_results tr
JOIN test_executions te ON tr.execution_id = te.id
ORDER BY tr.test_name, te.collection_id, te.started_at DESC;
	`

	_, err := s.db.Exec(upSQL)
	if err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}
