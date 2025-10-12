package util_test

import (
	"gemini-demo/internal/i18n"
	"gemini-demo/internal/testutil"
	"gemini-demo/internal/util"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestProjectRoot(t *testing.T) {
	// Save original environment and working directory
	originalTestRoot := os.Getenv("GEMINI_TEST_ROOT")
	originalWd, _ := os.Getwd()
	defer func() {
		os.Setenv("GEMINI_TEST_ROOT", originalTestRoot)
		os.Chdir(originalWd)
		util.ResetProjectRootCacheForTesting()
	}()

	// Unset GEMINI_TEST_ROOT for most tests
	os.Unsetenv("GEMINI_TEST_ROOT")
	util.ResetProjectRootCacheForTesting()

	t.Run("GEMINI_TEST_ROOT is set", func(t *testing.T) {
		expectedRoot := "/fake/root"
		os.Setenv("GEMINI_TEST_ROOT", expectedRoot)
		defer os.Unsetenv("GEMINI_TEST_ROOT")
		util.ResetProjectRootCacheForTesting()

		assert.Equal(t, expectedRoot, util.ProjectRoot(""))
	})

	t.Run("finds go.mod", func(t *testing.T) {
		tmpDir, err := os.MkdirTemp("", "proj-root-test-")
		assert.NoError(t, err)
		defer os.RemoveAll(tmpDir)

		projectDir := filepath.Join(tmpDir, "my-project")
		subDir := filepath.Join(projectDir, "cmd", "app")
		os.MkdirAll(subDir, 0755)

		_, err = os.Create(filepath.Join(projectDir, "go.mod"))
		assert.NoError(t, err)

		os.Chdir(subDir)

		actualRoot := util.ProjectRoot("")
		assert.Equal(t, filepath.Clean(projectDir), filepath.Clean(actualRoot))
	})

	t.Run("no go.mod found - fallback to start dir", func(t *testing.T) {
		// Use a non-existent path to ensure no go.mod is found
		mockPath := filepath.Join("non", "existent", "path")
		expectedRoot := mockPath
		actualRoot := util.ProjectRoot(filepath.Join(mockPath, "fake_executable"))
		assert.Equal(t, expectedRoot, actualRoot)
	})

	t.Run("executable in bin directory", func(t *testing.T) {
		// Use a non-existent path to ensure no go.mod is found
		mockExecutablePath := filepath.Join("C:", "mock", "project", "bin", "test_executable.exe")
		expectedRoot := filepath.Join("C:", "mock", "project")
		actualRoot := util.ProjectRoot(mockExecutablePath)
		assert.Equal(t, expectedRoot, actualRoot)
	})
}

func TestParseTemplates(t *testing.T) {
	tmpDir, cleanup, err := testutil.SetupTestEnv(t)
	assert.NoError(t, err)
	if err != nil {
		t.FailNow()
	}
	defer cleanup()

	// Create a dummy translator
	i18nPath := filepath.Join(tmpDir, "data", "i18n")
	translator := i18n.NewTranslator(i18nPath, "en")

	t.Run("success", func(t *testing.T) {
		templates, err := util.ParseTemplates(translator, tmpDir)
		assert.NoError(t, err)
		assert.NotEmpty(t, templates)
		// Check if a specific template is parsed
		_, ok := templates["index.html"]
		assert.True(t, ok)
	})

	t.Run("no templates found", func(t *testing.T) {
		subTmpDir, err := os.MkdirTemp(tmpDir, "sub")
		assert.NoError(t, err)
		defer os.RemoveAll(subTmpDir)

		// Create an empty templates directory
		err = os.Mkdir(filepath.Join(subTmpDir, "templates"), 0755)
		assert.NoError(t, err)

		_, err = util.ParseTemplates(translator, subTmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no templates found")
	})

	t.Run("template with syntax error", func(t *testing.T) {
		// Create a template with a syntax error
		badTemplatePath := filepath.Join(tmpDir, "templates", "bad.html")
		badTemplateContent := "{{.InvalidSyntax"
		os.WriteFile(badTemplatePath, []byte(badTemplateContent), 0644)
		defer os.Remove(badTemplatePath)

		_, err := util.ParseTemplates(translator, tmpDir)
		assert.Error(t, err)
	})

	t.Run("directory with .html suffix", func(t *testing.T) {
		// Create a directory with a .html suffix to cause an error
		badPath := filepath.Join(tmpDir, "templates", "a-directory.html")
		os.Mkdir(badPath, 0755)
		defer os.RemoveAll(badPath)

		_, err := util.ParseTemplates(translator, tmpDir)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "template entry is a directory")
	})
}
