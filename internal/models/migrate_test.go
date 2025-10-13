package models_test

import (
	"gemini-demo/internal/models"
	"io/ioutil"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

var seedFileMutex = &sync.Mutex{}

func TestAutoMigrateAndSeed_FileError(t *testing.T) {
	seedFileMutex.Lock()
	defer seedFileMutex.Unlock()

	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Temporarily rename the seed file to trigger a read error
	seedFilePath := filepath.Join("..", "..", "data", "seed_data.json")
	backupPath := seedFilePath + ".bak"
	err := os.Rename(seedFilePath, backupPath)
	assert.NoError(t, err)

	// Defer renaming it back
	defer os.Rename(backupPath, seedFilePath)

	// Now, run the function, expecting an error
	err = models.AutoMigrateAndSeed(db)
	assert.Error(t, err) // Just check that an error is returned
}

func TestAutoMigrateAndSeed_JsonError(t *testing.T) {
	seedFileMutex.Lock()
	defer seedFileMutex.Unlock()
	
	db, cleanup := setupTestDB(t)
	defer cleanup()

	// Create a malformed seed file
	seedFilePath := filepath.Join("..", "..", "data", "seed_data.json")
	originalContent, err := ioutil.ReadFile(seedFilePath)
	assert.NoError(t, err)

	malformedContent := []byte("{")
	err = ioutil.WriteFile(seedFilePath, malformedContent, 0644)
	assert.NoError(t, err)

	// Defer restoring the original content
	defer ioutil.WriteFile(seedFilePath, originalContent, 0644)

	// Now, run the function, expecting an error
	err = models.AutoMigrateAndSeed(db)
	assert.Error(t, err)
}