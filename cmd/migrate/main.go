package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/Danikarat/tally/internal/config"
	"github.com/Danikarat/tally/internal/database"
)

func main() {
	up := flag.Bool("up", false, "apply all pending migrations")
	down := flag.Bool("down", false, "rollback all migrations")
	version := flag.Bool("version", false, "print current migration version")
	flag.Parse()

	cfg, err := config.Load()
	if err != nil {
		slog.Error("failed to load configuration", "error", err)
		os.Exit(1)
	}

	dsn := cfg.DB.DSN()

	switch {
	case *up:
		if err := database.MigrateUp(dsn); err != nil {
			slog.Error("migration up failed", "error", err)
			os.Exit(1)
		}
		fmt.Println("Migrations applied successfully.")
	case *down:
		if err := database.MigrateDown(dsn); err != nil {
			slog.Error("migration down failed", "error", err)
			os.Exit(1)
		}
		fmt.Println("Migrations rolled back successfully.")
	case *version:
		ver, dirty, err := database.MigrateVersion(dsn)
		if err != nil {
			slog.Error("failed to get migration version", "error", err)
			os.Exit(1)
		}
		fmt.Printf("Current migration version: %d (dirty: %v)\n", ver, dirty)
	default:
		flag.Usage()
	}
}
