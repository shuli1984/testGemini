package testutil_test

import (
	"fmt"
	"gemini-demo/internal/testutil"
	"gemini-demo/internal/util"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
	"github.com/stretchr/testify/assert"
)

func TestSetupTestEnv(t *testing.T) {
	// Store original GEMINI_TEST_ROOT to restore it later
	originalEnv := os.Getenv("GEMINI_TEST_ROOT")
	defer os.Setenv("GEMINI_TEST_ROOT", originalEnv)

	tmpDir, cleanup, err := testutil.SetupTestEnv(t)
	assert.NoError(t, err)
	if err != nil {
		t.FailNow() // Stop test if setup fails
	}
	defer cleanup()

	// 1. Check if the temp directory exists and is a directory
	info, err := os.Stat(tmpDir)
	assert.NoError(t, err, "Temporary directory should exist")
	assert.True(t, info.IsDir(), "Temporary path should be a directory")

	// 2. Check if the GEMINI_TEST_ROOT env var is set correctly
	assert.Equal(t, tmpDir, os.Getenv("GEMINI_TEST_ROOT"), "GEMINI_TEST_ROOT should be set to the temp dir")

	// 3. Check if the subdirectories were copied
	for _, dir := range []string{"data", "static", "templates"} {
		_, err := os.Stat(filepath.Join(tmpDir, dir))
		assert.NoError(t, err, "Directory '%s' should have been copied to the temp dir", dir)
	}

	// 4. Check if the cleanup function works
	cleanup()
	_, err = os.Stat(tmpDir)
	assert.True(t, os.IsNotExist(err), "Temporary directory should be removed after cleanup")

	// 5. Check if the env var is restored (or was empty)
	// Note: The cleanup function from SetupTestEnv handles this restoration.
	// We call the deferred os.Setenv to be sure in case of test panic.
	// The testutil's cleanup will run first.
	assert.Equal(t, originalEnv, os.Getenv("GEMINI_TEST_ROOT"), "GEMINI_TEST_ROOT should be restored after cleanup")
}

func TestSetupTestEnv_MissingDir(t *testing.T) {
	// Create a temporary directory to act as a fake project root
	fakeRoot, err := ioutil.TempDir("", "fakeroot")
	assert.NoError(t, err)
	defer os.RemoveAll(fakeRoot)

	// Create a go.mod file to make it a valid project root
	err = ioutil.WriteFile(filepath.Join(fakeRoot, "go.mod"), []byte("module fake"), 0644)
	assert.NoError(t, err)

	// Create some of the required directories, but not all
	err = os.Mkdir(filepath.Join(fakeRoot, "static"), 0755)
	assert.NoError(t, err)
	err = os.Mkdir(filepath.Join(fakeRoot, "templates"), 0755)
	assert.NoError(t, err)
	// "data" directory is intentionally omitted

	// Temporarily set the working directory to the fake root, so ProjectRoot finds it
	originalWd, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(fakeRoot)
	assert.NoError(t, err)
	defer os.Chdir(originalWd)
	util.ResetProjectRootCacheForTesting()

	// Now, run SetupTestEnv. It should fail because the 'data' dir is missing.
	_, cleanup, err := testutil.SetupTestEnv(t)
	assert.Error(t, err, "SetupTestEnv should return an error if a source directory is missing")
	assert.Contains(t, err.Error(), "source directory for copy does not exist")
	assert.Nil(t, cleanup, "Cleanup function should be nil on error")
}

func TestSetupTestEnv_CopyError(t *testing.T) {
	// Mock the copy function to return an error
	originalCopyFn := testutil.CopyFn
	testutil.CopyFn = func(src, dest string, opt ...copy.Options) error {
		return fmt.Errorf("mock copy error")
	}
	defer func() { testutil.CopyFn = originalCopyFn }()

	// We still need a valid project structure for the test to reach the copy part
	fakeRoot, err := ioutil.TempDir("", "fakeroot-copy-error")
	assert.NoError(t, err)
	defer os.RemoveAll(fakeRoot)

	err = ioutil.WriteFile(filepath.Join(fakeRoot, "go.mod"), []byte("module fake"), 0644)
	assert.NoError(t, err)
	err = os.Mkdir(filepath.Join(fakeRoot, "data"), 0755)
	assert.NoError(t, err)
	err = os.Mkdir(filepath.Join(fakeRoot, "static"), 0755)
	assert.NoError(t, err)
	err = os.Mkdir(filepath.Join(fakeRoot, "templates"), 0755)
	assert.NoError(t, err)

	originalWd, err := os.Getwd()
	assert.NoError(t, err)
	err = os.Chdir(fakeRoot)
	assert.NoError(t, err)
	defer os.Chdir(originalWd)
	util.ResetProjectRootCacheForTesting()

	// Now, run SetupTestEnv. It should fail because the copyFn is mocked.
	_, cleanup, err := testutil.SetupTestEnv(t)
	assert.Error(t, err, "SetupTestEnv should return an error if copy fails")
	assert.Contains(t, err.Error(), "mock copy error")
	assert.Nil(t, cleanup, "Cleanup function should be nil on error")
}