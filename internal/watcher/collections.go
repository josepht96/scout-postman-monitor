package watcher

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
)

// CollectionWatcher watches a directory for Postman collection files
type CollectionWatcher struct {
	directory string
}

// NewCollectionWatcher creates a new collection watcher
func NewCollectionWatcher(directory string) *CollectionWatcher {
	return &CollectionWatcher{
		directory: directory,
	}
}

// CollectionFile represents a discovered collection file
type CollectionFile struct {
	Name     string
	Path     string
	FullPath string
}

// EnvironmentFile represents a discovered Postman environment file
type EnvironmentFile struct {
	Name     string // Environment name from JSON
	FileName string // Actual filename
	Path     string
	FullPath string
}

// CollectionGroup represents a group of collections with an optional environment
type CollectionGroup struct {
	Directory    string
	Environment  *EnvironmentFile
	Collections  []CollectionFile
}

// ScanGroups scans subdirectories for collections and environment files, grouping them
func (w *CollectionWatcher) ScanGroups() ([]CollectionGroup, error) {
	// Check if directory exists
	if _, err := os.Stat(w.directory); os.IsNotExist(err) {
		return nil, fmt.Errorf("directory does not exist: %s", w.directory)
	}

	// Get all subdirectories
	entries, err := os.ReadDir(w.directory)
	if err != nil {
		return nil, fmt.Errorf("failed to read directory: %w", err)
	}

	var groups []CollectionGroup

	for _, entry := range entries {
		if !entry.IsDir() {
			continue // Skip files in root directory
		}

		// Validate directory name does not contain spaces
		if strings.Contains(entry.Name(), " ") {
			log.Printf("Error: Collection directory name contains spaces: '%s'. Directory names must not contain spaces. Skipping this directory.", entry.Name())
			continue
		}

		subdir := filepath.Join(w.directory, entry.Name())

		// Scan this subdirectory
		subdirGroups, err := w.scanSubdirectory(subdir, entry.Name())
		if err != nil {
			// Log error but continue with other directories
			fmt.Printf("Warning: failed to scan subdirectory %s: %v\n", subdir, err)
			continue
		}

		groups = append(groups, subdirGroups...)
	}

	return groups, nil
}

// scanSubdirectory scans a single subdirectory and creates groups
func (w *CollectionWatcher) scanSubdirectory(subdirPath, subdirName string) ([]CollectionGroup, error) {
	// Find all .json files in this subdirectory
	entries, err := os.ReadDir(subdirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read subdirectory: %w", err)
	}

	var environmentFiles []EnvironmentFile
	var collectionFiles []CollectionFile

	for _, entry := range entries {
		if entry.IsDir() {
			continue // Don't recurse into subdirectories
		}

		filename := entry.Name()
		if !strings.HasSuffix(strings.ToLower(filename), ".json") {
			continue
		}

		filePath := filepath.Join(subdirPath, filename)
		absPath, err := filepath.Abs(filePath)
		if err != nil {
			continue
		}

		relPath, err := filepath.Rel(w.directory, filePath)
		if err != nil {
			relPath = filename
		}

		// Check if this is an environment file
		if strings.Contains(strings.ToLower(filename), ".postman_environment.json") {
			envFile, err := w.parseEnvironmentFile(absPath, filename, relPath)
			if err != nil {
				fmt.Printf("Warning: failed to parse environment file %s: %v\n", filename, err)
				continue
			}
			environmentFiles = append(environmentFiles, *envFile)
		} else {
			// It's a collection file
			collectionFiles = append(collectionFiles, CollectionFile{
				Name:     filename,
				Path:     relPath,
				FullPath: absPath,
			})
		}
	}

	// Create groups based on environment files
	var groups []CollectionGroup

	if len(environmentFiles) > 0 {
		// Create a group for each environment file
		for _, envFile := range environmentFiles {
			group := CollectionGroup{
				Directory:   subdirName,
				Environment: &envFile,
				Collections: collectionFiles,
			}
			groups = append(groups, group)
		}
	} else {
		// No environment file - create an ungrouped group
		if len(collectionFiles) > 0 {
			group := CollectionGroup{
				Directory:   subdirName,
				Environment: nil,
				Collections: collectionFiles,
			}
			groups = append(groups, group)
		}
	}

	return groups, nil
}

// parseEnvironmentFile parses a Postman environment file to extract the name
func (w *CollectionWatcher) parseEnvironmentFile(fullPath, filename, relPath string) (*EnvironmentFile, error) {
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	var envData struct {
		Name string `json:"name"`
	}

	if err := json.Unmarshal(data, &envData); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %w", err)
	}

	if envData.Name == "" {
		envData.Name = strings.TrimSuffix(filename, ".postman_environment.json")
	}

	return &EnvironmentFile{
		Name:     envData.Name,
		FileName: filename,
		Path:     relPath,
		FullPath: fullPath,
	}, nil
}

// Scan is deprecated in favor of ScanGroups but kept for backward compatibility
func (w *CollectionWatcher) Scan() ([]CollectionFile, error) {
	groups, err := w.ScanGroups()
	if err != nil {
		return nil, err
	}

	// Flatten groups into a single list of collections
	var collections []CollectionFile
	for _, group := range groups {
		collections = append(collections, group.Collections...)
	}

	return collections, nil
}

// GetDirectory returns the watched directory path
func (w *CollectionWatcher) GetDirectory() string {
	return w.directory
}
