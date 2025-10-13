package models_test

import (
	"gemini-demo/internal/models"
	"testing"
	"time"


	"github.com/stretchr/testify/assert"
)

func TestNewDBStore(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := models.NewDBStore(db)
	assert.NotNil(t, store)
}

func TestGetPageCount(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := models.NewDBStore(db)

	// Get initial page count from seed
	count, err := store.GetPageCount()
	assert.NoError(t, err)
	assert.Greater(t, count, int64(0)) // Assuming seed data has pages
}

func TestCreateAndGetRecentLoginLogs(t *testing.T) {
	db, cleanup := setupTestDB(t)
	defer cleanup()

	store := models.NewDBStore(db)

	// Create a couple of login logs
	log1 := &models.LoginLog{Username: "user1", Success: true, IPAddress: "127.0.0.1"}
	err := store.CreateLoginLog(log1)
	assert.NoError(t, err)

	time.Sleep(10 * time.Millisecond) // Ensure CreatedAt is different

	log2 := &models.LoginLog{Username: "user2", Success: false, IPAddress: "127.0.0.2"}
	err = store.CreateLoginLog(log2)
	assert.NoError(t, err)

	// Get recent login logs
	logs, err := store.GetRecentLoginLogs(2)
	assert.NoError(t, err)
	assert.Len(t, logs, 2)

	// Check if the logs are in the correct order (most recent first)
	assert.Equal(t, "user2", logs[0].Username)
	assert.Equal(t, "user1", logs[1].Username)

	// Test limit
	logs, err = store.GetRecentLoginLogs(1)
	assert.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "user2", logs[0].Username)
}