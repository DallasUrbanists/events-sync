package main

import (
	"fmt"
	"log"
	"os"
	"sort"
	"time"

	"github.com/dallasurbanists/events-sync/internal/config"
	"github.com/dallasurbanists/events-sync/internal/database"
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

	var allEvents []event.Event
	for orgName, url := range cfg.Organizations {
		fmt.Printf("Processing organization: %s\n", orgName)

		events, err := event.FetchAndParseEvents(url, orgName)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", orgName, err)
			continue
		}

		// Store events in database
		for _, e := range events {
			if err := db.UpsertEvent(e); err != nil {
				fmt.Printf("Error storing event %s: %v\n", e.UID, err)
			}
		}

		allEvents = append(allEvents, events...)
		fmt.Printf("Found %d events for %s\n", len(events), orgName)
	}

	// Automatically mark past events as reviewed
	fmt.Println("\n=== Marking past events as reviewed ===")
	if err := db.MarkPastEventsAsReviewed(); err != nil {
		log.Printf("Warning: Could not mark past events as reviewed: %v", err)
	}

	// Load all events from database
	dbEvents, err := db.GetEvents()
	if err != nil {
		log.Printf("Warning: Could not load events from database: %v", err)
	} else {
		// Convert database events to package events
		allEvents = make([]event.Event, len(dbEvents))
		for i, dbEvent := range dbEvents {
			allEvents[i] = databaseEventToEvent(dbEvent)
		}
	}

	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].StartTime.Before(allEvents[j].StartTime)
	})

	// output, err := json.MarshalIndent(allEvents, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling events: %v", err)
	}

	fmt.Printf("\nTotal events found: %d\n", len(allEvents))
	// fmt.Println(string(output))

	if err := showReviewStatusSummary(db); err != nil {
		log.Printf("Warning: Could not show review status summary: %v", err)
	}
}

// databaseEventToEvent converts a database Event to a package Event
func databaseEventToEvent(dbEvent database.Event) event.Event {
	e := event.Event{
		Organization: dbEvent.Organization,
		UID:         dbEvent.UID,
		Summary:     dbEvent.Summary,
		StartTime:   dbEvent.StartTime,
		EndTime:     dbEvent.EndTime,
		Created:     time.Time{},
		Modified:    time.Time{},
	}

	if dbEvent.Description != nil {
		e.Description = *dbEvent.Description
	}
	if dbEvent.Location != nil {
		e.Location = *dbEvent.Location
	}
	if dbEvent.CreatedTime != nil {
		e.Created = *dbEvent.CreatedTime
	}
	if dbEvent.ModifiedTime != nil {
		e.Modified = *dbEvent.ModifiedTime
	}
	if dbEvent.Status != nil {
		e.Status = *dbEvent.Status
	}
	if dbEvent.Transparency != nil {
		e.Transparency = *dbEvent.Transparency
	}

	return e
}

func showReviewStatusSummary(db *database.DB) error {
	fmt.Println("\n=== Review Status Summary ===")

	statuses := []string{"pending", "reviewed", "rejected"}
	for _, status := range statuses {
		events, err := db.GetEventsByReviewStatus(status)
		if err != nil {
			return fmt.Errorf("failed to get events with status %s: %v", status, err)
		}
		fmt.Printf("%s: %d events\n", status, len(events))
	}

	return nil
}
