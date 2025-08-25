package main

import (
	"fmt"
	"gemini-demo/internal/database"
	"gemini-demo/internal/models"
	"gemini-demo/internal/server"
	"log"

	"github.com/spf13/viper"
)

func main() {
	// Load configuration
	viper.SetConfigName("config")
	viper.AddConfigPath(".")
	viper.SetConfigType("yml")
	if err := viper.ReadInConfig(); err != nil {
		log.Fatalf("Error reading config file, %s", err)
	}

	// Initialize database
	db, sqlDB, err := database.InitDB()
	if err != nil {
		log.Fatalf("failed to initialize database: %v", err)
	}
	defer func() {
		if sqlDB != nil {
			sqlDB.Close()
		}
	}()

	err = models.AutoMigrateAndSeed(db)
	if err != nil {
		log.Fatalf("failed to auto migrate and seed models: %v", err)
	}

	// Create and start server
	srv := server.New(db)
	addr := viper.GetString("server.address")
	srv.Addr = addr

	fmt.Printf("Server is listening on %s\n", addr)
	log.Fatal(srv.ListenAndServe())
}
