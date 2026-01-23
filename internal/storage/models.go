package storage

import "time"

// Collection represents a Postman collection being monitored
type Collection struct {
	ID              int       `json:"id"`
	Name            string    `json:"name"`
	FilePath        string    `json:"file_path"`
	CompositeKey    string    `json:"composite_key"`
	DirectoryName   string    `json:"directory_name"`
	EnvironmentName string    `json:"environment_name"`
	CollectionName  string    `json:"collection_name"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// TestExecution represents a single execution run of a collection
type TestExecution struct {
	ID             int       `json:"id"`
	CollectionID   int       `json:"collection_id"`
	CollectionName string    `json:"collection_name"`
	StartedAt      time.Time `json:"started_at"`
	CompletedAt    time.Time `json:"completed_at"`
	DurationMs     int       `json:"duration_ms"`
	TotalTests     int       `json:"total_tests"`
	PassedTests    int       `json:"passed_tests"`
	FailedTests    int       `json:"failed_tests"`
	Error          *string   `json:"error,omitempty"`
	CreatedAt      time.Time `json:"created_at"`
}

// TestResult represents an individual test result within an execution
type TestResult struct {
	ID              int       `json:"id"`
	ExecutionID     int       `json:"execution_id"`
	TestName        string    `json:"test_name"`
	ExecutionName   *string   `json:"execution_name,omitempty"`
	URL             *string   `json:"url,omitempty"`
	Method          *string   `json:"method,omitempty"`
	Status          string    `json:"status"`
	StatusCode      *int      `json:"status_code,omitempty"`
	ResponseTimeMs  *int      `json:"response_time_ms,omitempty"`
	Passed          bool      `json:"passed"`
	Error           *string   `json:"error,omitempty"`
	CreatedAt       time.Time `json:"created_at"`
}

// ExecutionWithResults combines execution data with its test results
type ExecutionWithResults struct {
	Execution TestExecution `json:"execution"`
	Results   []TestResult  `json:"results"`
}

// EnvironmentInfo represents environment metadata for API responses
type EnvironmentInfo struct {
	Name     string `json:"name"`
	FileName string `json:"file_name"`
	Path     string `json:"path"`
}

// EnvironmentGroup represents a group of collections with optional environment
type EnvironmentGroup struct {
	Environment *EnvironmentInfo   `json:"environment,omitempty"`
	Directory   string             `json:"directory"`
	Collections []CollectionResult `json:"collections"`
}

// LatestResults represents the latest test results for API responses
type LatestResults struct {
	EnvironmentGroups []EnvironmentGroup `json:"environment_groups"`
	UpdatedAt         time.Time          `json:"updated_at"`
}

// CollectionResult represents results for a single collection
type CollectionResult struct {
	Collection          Collection      `json:"collection"`
	Execution           *TestExecution  `json:"execution,omitempty"`
	LastSuccessExecution *TestExecution `json:"last_success_execution,omitempty"`
	Results             []TestResult    `json:"results"`
}
