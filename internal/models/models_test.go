package models_test

import (
	"gemini-demo/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB creates a new in-memory DB for testing.
func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	t.Helper()

	// Initialize an in-memory SQLite database
	db, err := gorm.Open(sqlite.Open("file::memory:"), &gorm.Config{})
	assert.NoError(t, err)

	sqlDB, err := db.DB()
	assert.NoError(t, err)

	// Auto-migrate and seed models
	err = models.AutoMigrateAndSeed(db)
	assert.NoError(t, err)

	cleanup := func() {
		sqlDB.Close()
	}

	return db, cleanup
}
