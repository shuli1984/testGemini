package database_test

import (
	"gemini-demo/internal/database" // Added import for the database package
	"gemini-demo/tests/testutil"
	"os"
	"testing"
	"strings" // Added for strings.Contains

	"github.com/spf13/viper"
)

func TestMain(m *testing.M) {
	testutil.SetupViper()
	os.Exit(m.Run())
}

func TestInitDB(t *testing.T) {
	// Remove the database file if it exists
	os.Remove("./gemini.db")

	db, sqlDB, err := database.InitDB() // Updated call to database.InitDB()
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	defer func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	// Check if the database connection is successful
	if db == nil {
		t.Fatal("GORM DB instance is nil after InitDB")
	}

	// Try to ping the database to ensure connection is active
	if err := sqlDB.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}

	// Clean up the database file
	os.Remove("./gemini.db")
}

func TestInitDB_UnsupportedDBType(t *testing.T) {
	// Save current viper settings and restore them after the test
	originalDBType := viper.GetString("database.type")
	originalDBDSN := viper.GetString("database.dsn")
	defer func() {
		viper.Set("database.type", originalDBType)
		viper.Set("database.dsn", originalDBDSN)
	}()

	viper.Set("database.type", "unsupported")
	viper.Set("database.dsn", "file::memory:?cache=shared") // DSN doesn't matter for unsupported type

	_, _, err := database.InitDB() // Updated call to database.InitDB()
	if err == nil {
		t.Fatal("expected an error for unsupported database type, got nil")
	}

	expectedErrorMsg := "unsupported database type: unsupported"
	if err.Error() != expectedErrorMsg {
			t.Fatalf("expected error message \"%s\", got \"%s\"", expectedErrorMsg, err.Error())
		}
	}

	func TestInitDB_NoDBType(t *testing.T) {
		originalDBType := viper.GetString("database.type")
		defer func() {
			viper.Set("database.type", originalDBType)
		}()

		viper.Set("database.type", "") // Set empty database type

		_, _, err := database.InitDB()
		if err == nil {
			t.Fatal("expected an error for no database type, got nil")
		}

		expectedErrorMsg := "database type is not specified in config"
		if err.Error() != expectedErrorMsg {
			t.Fatalf("expected error message %q, got %q", expectedErrorMsg, err.Error())
		}
	}

	func TestInitDB_NoDBDSN(t *testing.T) {
		originalDBDSN := viper.GetString("database.dsn")
		defer func() {
			viper.Set("database.dsn", originalDBDSN)
		}()

		viper.Set("database.dsn", "") // Set empty database DSN

		_, _, err := database.InitDB()
		if err == nil {
			t.Fatal("expected an error for no database DSN, got nil")
		}

		expectedErrorMsg := "database DSN is not specified in config"
		if err.Error() != expectedErrorMsg {
			t.Fatalf("expected error message %q, got %q", expectedErrorMsg, err.Error())
		}
	}

func TestInitDB_InvalidDSN(t *testing.T) {
	originalDBDSN := viper.GetString("database.dsn")
	defer func() {
		viper.Set("database.dsn", originalDBDSN)
	}()

	// Set an invalid DSN for SQLite
	viper.Set("database.dsn", "/non/existent/path/to/db.db")

	_, _, err := database.InitDB()
	if err == nil {
		t.Fatal("expected an error for invalid DSN, got nil")
	}

	expectedErrorPart := "failed to connect database with GORM"
	if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Fatalf("expected error message to contain %q, got %q", expectedErrorPart, err.Error())
	}
}