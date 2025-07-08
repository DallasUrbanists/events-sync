package migration

import (
	"database/sql"
	"os"
	"testing"

	_ "github.com/lib/pq"
)

func TestRunMigrations(t *testing.T) {
	// Skip if not in CI and no database available
	if os.Getenv("CI") == "" {
		dbURL := os.Getenv("TEST_DATABASE_URL")
		if dbURL == "" {
			t.Skip("Skipping migration test: no TEST_DATABASE_URL set")
		}
	}

	// Use test database URL or default
	dbURL := os.Getenv("TEST_DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("No DATABASE_URL given")
	}

	// Connect to test database
	db, err := sql.Open("postgres", dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}
	defer db.Close()

	// Test running migrations
	err = RunMigrations(db, "migrations")
	if err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Test getting migration version
	version, err := GetMigrationVersion(db, "migrations")
	if err != nil {
		t.Fatalf("Failed to get migration version: %v", err)
	}

	// Should have at least version 1 (our initial migration)
	if version < 1 {
		t.Errorf("Expected migration version >= 1, got %d", version)
	}
}
