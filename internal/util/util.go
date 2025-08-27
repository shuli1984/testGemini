package util

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
)

var (
	projectRootCache string
	once             sync.Once
)

// ProjectRoot returns the project's root directory.
// If testExecutablePath is provided, it's used instead of os.Executable() for testing.
func ProjectRoot(testExecutablePath string) string {
	once.Do(func() {
		if testRoot := os.Getenv("GEMINI_TEST_ROOT"); testRoot != "" {
			projectRootCache = testRoot
			return
		}

		var startDir string
		var err error

		if testExecutablePath != "" {
			startDir = filepath.Dir(testExecutablePath) // For testing, use the directory of the provided executable path
		} else {
			// Use current working directory as the starting point
			startDir, err = os.Getwd()
			if err != nil {
				panic(err) // Or handle error more gracefully
			}
		}

		// Search upwards from the starting directory for a known project root marker (e.g., go.mod)
		currentDir := startDir
		// Look for go.mod file, if not found, search parent directory
		for {
			if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
				projectRootCache = currentDir
				return
			}
			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				break // Reached root directory, go.mod not found
			}
			currentDir = parent
		}

		// Fallback if go.mod is not found, with special handling for 'bin' directory
		if strings.HasSuffix(filepath.ToSlash(startDir), "/bin") {
			projectRootCache = filepath.Dir(startDir)
		} else {
			projectRootCache = startDir // Fallback to the starting directory
		}
	})
	return projectRootCache
}

// ResetProjectRootCacheForTesting resets the cached project root for testing purposes.
// This function should only be called in test code.
func ResetProjectRootCacheForTesting() {
	once = sync.Once{}
	projectRootCache = ""
}