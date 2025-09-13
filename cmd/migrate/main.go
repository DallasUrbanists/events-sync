package main

import (
	"database/sql"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/dallasurbanists/events-sync/internal/migration"
	_ "github.com/lib/pq"
)

func main() {
	var (
		dbURL         = flag.String("database", "", "Database connection URL")
		migrationsDir = flag.String("migrations", "migrations", "Path to migrations directory")
		action        = flag.String("action", "up", "Migration action: up, down, version")
		steps         = flag.Int("steps", 1, "Number of migration steps (for down action)")
	)
	flag.Parse()

	if *dbURL == "" {
		*dbURL = os.Getenv("DATABASE_URL")
		if *dbURL == "" {
			log.Fatalf("No DATABASE_URL given")
		}
	}

	db, err := sql.Open("postgres", *dbURL)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := db.Ping(); err != nil {
		log.Fatalf("Failed to ping database: %v", err)
	}

	switch *action {
	case "up":
		fmt.Println("Running migrations...")
		if err := migration.RunMigrations(db, *migrationsDir); err != nil {
			log.Fatalf("Failed to run migrations: %v", err)
		}
		fmt.Println("Migrations completed successfully")

	case "down":
		if *steps <= 0 {
			log.Fatalf("Steps must be a positive number, got: %d", *steps)
		}
		fmt.Printf("Rolling back %d migration(s)...\n", *steps)
		if err := migration.RunMigrationsDown(db, *migrationsDir, *steps); err != nil {
			log.Fatalf("Failed to rollback migrations: %v", err)
		}
		fmt.Println("Migration rollback completed successfully")

	case "version":
		version, err := migration.GetMigrationVersion(db, *migrationsDir)
		if err != nil {
			log.Fatalf("Failed to get migration version: %v", err)
		}
		fmt.Printf("Current migration version: %d\n", version)

	default:
		log.Fatalf("Unknown action: %s. Use 'up', 'down', or 'version'", *action)
	}
}
