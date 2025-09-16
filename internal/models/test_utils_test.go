package models_test

import (
	"fmt"
	"gemini-demo/internal/database"
	"gemini-demo/internal/models"
	"os"
	"path/filepath"
	"testing"
	"time"

	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) (*gorm.DB, func()) {
	testDBName := filepath.Join(os.TempDir(), fmt.Sprintf("test_%d.db", time.Now().UnixNano()))
	db, sqlDB, err := database.InitDB("sqlite", testDBName)
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}

	// AutoMigrate and Seed for this test
	err = models.AutoMigrateAndSeed(db)
	if err != nil {
		t.Fatalf("failed to auto migrate and seed: %v", err)
	}

	cleanup := func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
		os.Remove(testDBName)
	}

	return db, cleanup
}
