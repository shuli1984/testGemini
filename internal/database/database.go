package database

import (
	"database/sql"
	"fmt"
	"time"

	sqliteGorm "gorm.io/driver/sqlite" // Alias to avoid conflict
	"gorm.io/gorm"
)

// InitDB initializes the database connection using the provided configuration.
func InitDB(dbType, dbDSN string) (*gorm.DB, *sql.DB, error) {
	if dbType == "" {
		return nil, nil, fmt.Errorf("database type is not specified")
	}
	if dbDSN == "" {
		return nil, nil, fmt.Errorf("database DSN is not specified")
	}

	if dbType == "sqlite" {
		gormDB, err := gorm.Open(sqliteGorm.Open(dbDSN), &gorm.Config{})
		if err != nil {
			return nil, nil, fmt.Errorf("failed to connect database with GORM: %w", err)
		}

		sqlDB, err := gormDB.DB()
		if err != nil {
			return nil, nil, fmt.Errorf("failed to get *sql.DB from GORM: %w", err)
		}

		// Set connection pool settings
		sqlDB.SetMaxIdleConns(10)                  // Maximum number of connections in the idle connection pool.
		sqlDB.SetMaxOpenConns(100)                 // Maximum number of open connections to the database.
		sqlDB.SetConnMaxLifetime(time.Hour) // Maximum amount of time a connection may be reused.

		return gormDB, sqlDB, nil
	} else {
		return nil, nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
