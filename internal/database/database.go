package database

import (
	"database/sql"
	"fmt"
	"time" // New import

	"github.com/spf13/viper"
	sqliteGorm "gorm.io/driver/sqlite" // Alias to avoid conflict
	"gorm.io/gorm"
)

// InitDB initializes the database connection using the global Viper instance.
func InitDB() (*gorm.DB, *sql.DB, error) {
	return InitDBWithViper(viper.GetViper())
}

// InitDBWithViper initializes the database connection using a provided Viper instance.
func InitDBWithViper(v *viper.Viper) (*gorm.DB, *sql.DB, error) {
	dbType := v.GetString("database.type")
	dbDSN := v.GetString("database.dsn")

	if dbType == "" {
		return nil, nil, fmt.Errorf("database type is not specified in config")
	}
	if dbDSN == "" {
		return nil, nil, fmt.Errorf("database DSN is not specified in config")
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
		sqlDB.SetMaxIdleConns(10)  // Maximum number of connections in the idle connection pool.
		sqlDB.SetMaxOpenConns(100) // Maximum number of open connections to the database.
		sqlDB.SetConnMaxLifetime(time.Hour) // Maximum amount of time a connection may be reused.

		return gormDB, sqlDB, nil
	} else {
		return nil, nil, fmt.Errorf("unsupported database type: %s", dbType)
	}
}
