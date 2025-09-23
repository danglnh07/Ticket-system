package main

import (
	"log/slog"
	"os"

	"github.com/danglnh07/ticket-system/ticket-system/db"
	"github.com/danglnh07/ticket-system/ticket-system/util"
)

func main() {
	// Initialize logger
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	// Load config
	config := util.LoadConfig(".env")

	// Connect to database and run database migration
	queries := db.NewQueries()
	if err := queries.ConnectDB(config.DBConn); err != nil {
		logger.Error("Error connecting to database", "error", err)
		os.Exit(1)
	}
	if err := queries.AutoMigration(); err != nil {
		logger.Error("Error running auto migration", "error", err)
		os.Exit(1)
	}
}
