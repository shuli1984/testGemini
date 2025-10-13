package models_test

import (
	"gemini-demo/internal/models"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDBErrors(t *testing.T) {
	db, cleanup := setupTestDB(t)

	store := models.NewDBStore(db)

	// Close the database connection to induce errors
	sqlDB, _ := db.DB()
	sqlDB.Close()

	t.Run("GetPageCount error", func(t *testing.T) {
		_, err := store.GetPageCount()
		assert.Error(t, err)
	})

	t.Run("GetRecentLoginLogs error", func(t *testing.T) {
		_, err := store.GetRecentLoginLogs(1)
		assert.Error(t, err)
	})

	t.Run("CreateLoginLog error", func(t *testing.T) {
		err := store.CreateLoginLog(&models.LoginLog{})
		assert.Error(t, err)
	})

	t.Run("GetPageData error", func(t *testing.T) {
		_, err := store.GetPageData("home", "en", "en")
		assert.Error(t, err)
	})

	t.Run("GetAllPages error", func(t *testing.T) {
		_, err := store.GetAllPages("en", "en")
		assert.Error(t, err)
	})

	t.Run("GetCoreSolutions error", func(t *testing.T) {
		_, err := store.GetCoreSolutions("en", "en")
		assert.Error(t, err)
	})

	t.Run("GetRecentPages error", func(t *testing.T) {
		_, err := store.GetRecentPages(1, "en", "en")
		assert.Error(t, err)
	})

	t.Run("GetPagesByIDs error", func(t *testing.T) {
		_, err := store.GetPagesByIDs([]uint{1}, "en", "en")
		assert.Error(t, err)
	})

	t.Run("GetSiteConfig error", func(t *testing.T) {
		_, err := store.GetSiteConfig("en", "en")
		assert.Error(t, err)
	})

	t.Run("GetSettingValue error", func(t *testing.T) {
		_, err := store.GetSettingValue("key", "en")
		assert.Error(t, err)
	})

	// We call cleanup at the end, although the DB is already closed.
	cleanup()
}
