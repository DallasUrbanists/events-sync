package main

import (
	"fmt"
	"log"
	"os"

	"github.com/dallasurbanists/events-sync/internal/config"
	"github.com/dallasurbanists/events-sync/internal/database"
	"github.com/dallasurbanists/events-sync/internal/importer"
	"github.com/dallasurbanists/events-sync/pkg/event"
)

func main() {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		log.Fatalf("No DATABASE_URL given")
	}

	db, err := database.Connect(dbURL)
	if err != nil {
		log.Fatalf("Error connecting to database: %v", err)
	}
	defer db.Close()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	i := importer.RegisterImporters()
	var allEvents []event.Event
	for orgName, org := range cfg.Organizations {
		fmt.Printf("Processing organization: %s\n", orgName)

		var events []event.Event
		var err error
		events, err = i[org.Importer](org.URL, orgName, org.Options)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", orgName, err)
			continue
		}

		for _, e := range events {
			if err := db.UpsertEvent(e); err != nil {
				fmt.Printf("Error storing event %s: %v\n", e.UID, err)
			}
		}

		if err := db.DeleteEventsNotInSource(orgName, events); err != nil {
			fmt.Printf("Error deleting events not in source for %s: %v\n", orgName, err)
		}

		allEvents = append(allEvents, events...)
		fmt.Printf("Found %d events for %s\n", len(events), orgName)
	}

	fmt.Printf("\nTotal events found: %d\n", len(allEvents))
	if err := showReviewStatusSummary(db); err != nil {
		log.Printf("Warning: Could not show review status summary: %v", err)
	}
}


func showReviewStatusSummary(db *database.DB) error {
	fmt.Println("\n=== Rejected Status Summary ===")

	// Get rejected events
	rejectedEvents, err := db.GetEventsByRejectedStatus(true)
	if err != nil {
		return fmt.Errorf("failed to get rejected events: %v", err)
	}
	fmt.Printf("rejected: %d events\n", len(rejectedEvents))

	// Get non-rejected events
	nonRejectedEvents, err := db.GetEventsByRejectedStatus(false)
	if err != nil {
		return fmt.Errorf("failed to get non-rejected events: %v", err)
	}
	fmt.Printf("approved: %d events\n", len(nonRejectedEvents))

	return nil
}
