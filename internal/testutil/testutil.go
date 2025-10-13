package testutil

import (
	"fmt"
	"gemini-demo/internal/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
)

// CopyFn is a mockable version of copy.Copy.
var CopyFn = copy.Copy

// SetupTestEnv creates a temporary directory to act as the project root,
// copies necessary assets (data, static, templates), sets the
// GEMINI_TEST_ROOT environment variable to point to this directory,
// and returns the temporary directory path and a cleanup function.
func SetupTestEnv(t *testing.T) (string, func(), error) {
	t.Helper()

	// Get the real project root.
	// We clear the test root env var first to make sure we find the real root.
	originalEnv := os.Getenv("GEMINI_TEST_ROOT")
	os.Unsetenv("GEMINI_TEST_ROOT")
	util.ResetProjectRootCacheForTesting()
	realProjectRoot := util.ProjectRoot("")
	if realProjectRoot == "" {
		return "", nil, fmt.Errorf("Could not find project root")
	}
	// Restore env var if it was present, in case of error
	if originalEnv != "" {
		os.Setenv("GEMINI_TEST_ROOT", originalEnv)
	}


	// Create a temporary directory to act as the project root for tests.
	tmpDir, err := ioutil.TempDir("", "test-project-root-")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create temp dir: %w", err)
	}

	// Copy necessary directories to the temporary root.
	for _, dir := range []string{"data", "static", "templates"} {
		srcPath := filepath.Join(realProjectRoot, dir)
		destPath := filepath.Join(tmpDir, dir)
		if _, err := os.Stat(srcPath); os.IsNotExist(err) {
			os.RemoveAll(tmpDir)
			return "", nil, fmt.Errorf("source directory for copy does not exist: %s", srcPath)
		}
		if err := CopyFn(srcPath, destPath); err != nil {
			// If the source directory doesn't exist, we can skip it.
			// This can happen in some CI environments.
			if !os.IsNotExist(err) {
				os.RemoveAll(tmpDir)
				return "", nil, fmt.Errorf("failed to copy directory %s: %w", dir, err)
			}
		}
	}

	// Set the environment variable to point to the temporary directory.
	os.Setenv("GEMINI_TEST_ROOT", tmpDir)
	util.ResetProjectRootCacheForTesting()

	// Return a cleanup function.
	return tmpDir, func() {
		os.RemoveAll(tmpDir)
		os.Setenv("GEMINI_TEST_ROOT", originalEnv)
		util.ResetProjectRootCacheForTesting()
	}, nil
}