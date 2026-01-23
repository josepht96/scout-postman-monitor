package executor

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"time"
)

// NewmanExecutor executes Postman collections using Newman
type NewmanExecutor struct {
	nodeExecutable string
	scriptPath     string
}

// NewNewmanExecutor creates a new Newman executor
func NewNewmanExecutor(scriptPath string) *NewmanExecutor {
	return &NewmanExecutor{
		nodeExecutable: "node",
		scriptPath:     scriptPath,
	}
}

// ExecutionSummary contains high-level execution summary
type ExecutionSummary struct {
	Total  int `json:"total"`
	Passed int `json:"passed"`
	Failed int `json:"failed"`
}

// TestInfo contains individual test information
type TestInfo struct {
	Name          string  `json:"name"`
	Passed        bool    `json:"passed"`
	Error         *string `json:"error"`
	ExecutionName string  `json:"executionName"`
}

// ExecutionInfo contains HTTP request execution information
type ExecutionInfo struct {
	Name         string  `json:"name"`
	URL          string  `json:"url"`
	Method       string  `json:"method"`
	Status       string  `json:"status"`
	StatusCode   *int    `json:"statusCode"`
	ResponseTime *int    `json:"responseTime"`
	Error        *string `json:"error"`
}

// NewmanResult contains the result from Newman execution
type NewmanResult struct {
	CollectionName  string           `json:"collectionName"`
	CollectionPath  string           `json:"collectionPath"`
	Timestamp       string           `json:"timestamp"`
	Summary         ExecutionSummary `json:"summary"`
	Tests           []TestInfo       `json:"tests"`
	Executions      []ExecutionInfo  `json:"executions"`
	TotalDurationMs int              `json:"totalDurationMs"`
	Error           *string          `json:"error"`
}

// Execute runs a Postman collection using Newman with an optional environment file
func (e *NewmanExecutor) Execute(collectionPath string, environmentPath *string, directoryName string, environmentName *string) (*NewmanResult, error) {
	// Resolve absolute path to the script
	scriptPath, err := filepath.Abs(e.scriptPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve script path: %w", err)
	}

	// Resolve absolute path to collection
	absCollectionPath, err := filepath.Abs(collectionPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve collection path: %w", err)
	}

	// Prepare command arguments
	args := []string{scriptPath, absCollectionPath}

	// Add environment path if provided (or empty string if not)
	if environmentPath != nil && *environmentPath != "" {
		absEnvironmentPath, err := filepath.Abs(*environmentPath)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve environment path: %w", err)
		}
		args = append(args, absEnvironmentPath)
	} else {
		args = append(args, "")
	}

	// Add directory name
	args = append(args, directoryName)

	// Add environment name (or empty string if not provided)
	if environmentName != nil && *environmentName != "" {
		args = append(args, *environmentName)
	} else {
		args = append(args, "")
	}

	// Prepare command
	cmd := exec.Command(e.nodeExecutable, args...)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	// Execute command
	err = cmd.Run()

	// Newman may return non-zero exit code if tests fail, but still produce valid output
	// So we'll try to parse the output regardless of exit code

	// Parse the JSON output
	var result NewmanResult
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		// If we can't parse the output, return the error along with stderr
		return nil, fmt.Errorf("failed to parse newman output: %w\nStderr: %s\nStdout: %s",
			err, stderr.String(), stdout.String())
	}

	// If there was an execution error but we got valid JSON, the error will be in result.Error
	if result.Error != nil && err != nil {
		return &result, fmt.Errorf("newman execution failed: %s", *result.Error)
	}

	return &result, nil
}

// SetNodeExecutable allows customizing the node executable path
func (e *NewmanExecutor) SetNodeExecutable(path string) {
	e.nodeExecutable = path
}

// IsAvailable checks if Node.js is available
func (e *NewmanExecutor) IsAvailable() bool {
	cmd := exec.Command(e.nodeExecutable, "--version")
	return cmd.Run() == nil
}

// GetVersion returns the Node.js version
func (e *NewmanExecutor) GetVersion() (string, error) {
	cmd := exec.Command(e.nodeExecutable, "--version")
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return string(bytes.TrimSpace(output)), nil
}

// Helper function to convert NewmanResult to storage-compatible format
func (r *NewmanResult) ToStorageFormat() (map[string]interface{}, error) {
	// Parse timestamp
	timestamp, err := time.Parse(time.RFC3339, r.Timestamp)
	if err != nil {
		timestamp = time.Now()
	}

	return map[string]interface{}{
		"collection_name": r.CollectionName,
		"started_at":      timestamp,
		"completed_at":    timestamp.Add(time.Duration(r.TotalDurationMs) * time.Millisecond),
		"duration_ms":     r.TotalDurationMs,
		"total_tests":     r.Summary.Total,
		"passed_tests":    r.Summary.Passed,
		"failed_tests":    r.Summary.Failed,
		"error":           r.Error,
		"tests":           r.Tests,
		"executions":      r.Executions,
	}, nil
}
