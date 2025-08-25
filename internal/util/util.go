package util

import (
	"os"
	"path/filepath"
	"sync"
)

var ( 
	projectRootCache string
	once sync.Once
)

// ProjectRoot returns the project's root directory.
// If testExecutablePath is provided, it's used instead of os.Executable() for testing.
func ProjectRoot(testExecutablePath string) string {
	once.Do(func() {
		if testRoot := os.Getenv("GEMINI_TEST_ROOT"); testRoot != "" {
			projectRootCache = testRoot
			return
		}

		var ex string
		var err error

		if testExecutablePath != "" {
			ex = testExecutablePath
		} else {
			ex, err = os.Executable()
			if err != nil {
				panic(err) // Or handle error more gracefully
			}
		}

			// Search upwards from the executable's directory for a known project root marker (e.g., go.mod)
		currentDir := filepath.Dir(ex)
		// Look for go.mod file, if not found, search parent directory
		for {
			if _, err := os.Stat(filepath.Join(currentDir, "go.mod")); err == nil {
				projectRootCache = currentDir
				return
			}
			parent := filepath.Dir(currentDir)
			if parent == currentDir {
				break
			}
			currentDir = parent
		}

		// Fallback if go.mod is not found
		dir := filepath.Dir(ex)
		if filepath.Base(dir) == "bin" {
			projectRootCache = filepath.Dir(dir) // Go up one more level
			return
		}
		projectRootCache = dir
	})
	return projectRootCache
}

// ResetProjectRootCacheForTesting resets the cached project root for testing purposes.
// This function should only be called in test code.
func ResetProjectRootCacheForTesting() {
	once = sync.Once{}
	projectRootCache = ""
}