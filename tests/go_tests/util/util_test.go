package util_test

import (
	"gemini-demo/internal/util"
	"os"
	"path/filepath"
	"testing"
)

func TestProjectRootAndTemplatePattern(t *testing.T) {
	projectRoot := util.ProjectRoot("") // Use empty string to use os.Executable()
	t.Logf("Project Root: '%s'", projectRoot)

	templatePattern := filepath.ToSlash(filepath.Join(projectRoot, "templates")) + "/**/*.html"
	t.Logf("Template Pattern: '%s'", templatePattern)

	// You can add assertions here if you have expected values
}

func TestProjectRoot_FallbackLogic(t *testing.T) {
	// Save original GEMINI_TEST_ROOT and defer restore
	originalTestRoot := os.Getenv("GEMINI_TEST_ROOT")
	defer func() {
		os.Setenv("GEMINI_TEST_ROOT", originalTestRoot)
		util.ResetProjectRootCacheForTesting() // Reset for subsequent tests
	}()

	// Unset GEMINI_TEST_ROOT to force fallback logic
	os.Unsetenv("GEMINI_TEST_ROOT")

	t.Run("executable in bin directory", func(t *testing.T) {
		util.ResetProjectRootCacheForTesting() // Reset cache for this sub-test
		mockExecutablePath := filepath.Join("C:", "mock", "project", "bin", "test_executable.exe")
		expectedRoot := filepath.Join("C:", "mock", "project")

		actualRoot := util.ProjectRoot(mockExecutablePath)

		if actualRoot != expectedRoot {
			t.Errorf("Expected ProjectRoot to be %s, but got %s", expectedRoot, actualRoot)
		}
	})

	t.Run("executable in project root", func(t *testing.T) {
		util.ResetProjectRootCacheForTesting() // Reset cache for this sub-test
		mockExecutablePath := filepath.Join("C:", "mock", "project", "test_executable.exe")
		expectedRoot := filepath.Join("C:", "mock", "project")

		actualRoot := util.ProjectRoot(mockExecutablePath)

		if actualRoot != expectedRoot {
			t.Errorf("Expected ProjectRoot to be %s, but got %s", expectedRoot, actualRoot)
		}
	})
}