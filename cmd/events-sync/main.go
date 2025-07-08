package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"sort"

	"github.com/dallasurbanists/events-sync/internal/config"
	"github.com/dallasurbanists/events-sync/internal/database"
	"github.com/dallasurbanists/events-sync/pkg/calendar"
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

		cal := calendar.NewCalendar(orgName)
		cal.SetDatabase(db)

		err := cal.FetchAndParse(url)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", orgName, err)
			continue
		}

		allEvents = append(allEvents, cal.GetEvents()...)
		fmt.Printf("Found %d events for %s\n", cal.GetEventCount(), orgName)
	}

	// Automatically mark past events as reviewed
	fmt.Println("\n=== Marking past events as reviewed ===")
	if err := db.MarkPastEventsAsReviewed(); err != nil {
		log.Printf("Warning: Could not mark past events as reviewed: %v", err)
	}

	cal := calendar.NewCalendar("all")
	cal.SetDatabase(db)
	if err := cal.LoadFromDB(); err != nil {
		log.Printf("Warning: Could not load events from database: %v", err)
	} else {
		allEvents = cal.GetEvents()
	}

	sort.Slice(allEvents, func(i, j int) bool {
		return allEvents[i].StartTime.Before(allEvents[j].StartTime)
	})

	output, err := json.MarshalIndent(allEvents, "", "  ")
	if err != nil {
		log.Fatalf("Error marshaling events: %v", err)
	}

	fmt.Printf("\nTotal events found: %d\n", len(allEvents))
	fmt.Println(string(output))

	if err := showReviewStatusSummary(db); err != nil {
		log.Printf("Warning: Could not show review status summary: %v", err)
	}
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
