package database_test

import (
	"gemini-demo/internal/database" // Added import for the database package
	"os"
	"strings" // Added for strings.Contains
	"testing"
)

func TestInitDB(t *testing.T) {
	// Remove the database file if it exists
	os.Remove("./gemini.db")

	db, sqlDB, err := database.InitDB("sqlite", "./gemini.db")
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
	_, _, err := database.InitDB("unsupported", "file::memory:?cache=shared") // DSN doesn't matter for unsupported type
	if err == nil {
		t.Fatal("expected an error for unsupported database type, got nil")
	}

	expectedErrorMsg := "unsupported database type: unsupported"
	if err.Error() != expectedErrorMsg {
		t.Fatalf("expected error message \"%s\", got \"%s\"", expectedErrorMsg, err.Error())
	}
}

func TestInitDB_NoDBType(t *testing.T) {
	_, _, err := database.InitDB("", "file::memory:?cache=shared")
	if err == nil {
		t.Fatal("expected an error for no database type, got nil")
	}

	expectedErrorMsg := "database type is not specified"
	if err.Error() != expectedErrorMsg {
		t.Fatalf("expected error message %q, got %q", expectedErrorMsg, err.Error())
	}
}

func TestInitDB_NoDBDSN(t *testing.T) {
	_, _, err := database.InitDB("sqlite", "")
	if err == nil {
		t.Fatal("expected an error for no database DSN, got nil")
	}

	expectedErrorMsg := "database DSN is not specified"
	if err.Error() != expectedErrorMsg {
		t.Fatalf("expected error message %q, got %q", expectedErrorMsg, err.Error())
	}
}

func TestInitDB_InvalidDSN(t *testing.T) {
	_, _, err := database.InitDB("sqlite", "/non/existent/path/to/db.db")
	if err == nil {
		t.Fatal("expected an error for invalid DSN, got nil")
	}

	expectedErrorPart := "failed to connect database with GORM"
	if !strings.Contains(err.Error(), expectedErrorPart) {
		t.Fatalf("expected error message to contain %q, got %q", expectedErrorPart, err.Error())
	}
}
