package main

import (
	"errors"
	"fmt"
	"log"
	"os"
	"time"

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
	defer db.DB.Close()

	cfg, err := config.LoadConfig()
	if err != nil {
		log.Fatalf("Error loading config: %v", err)
	}

	i := importer.RegisterImporters()
	for orgName, org := range cfg.Organizations {
		fmt.Printf("Processing organization: %s\n", orgName)

		events, err := i[org.Importer](org.URL, orgName, org.Options)
		if err != nil {
			fmt.Printf("Error processing %s: %v\n", orgName, err)
			continue
		}

		err = syncEvents(orgName, events, db.Events)
		if err != nil {
			log.Fatalf("failed to sync %v: %v", orgName, err)
		}

		fmt.Printf("Found %d events for %s\n", len(events), orgName)
	}

	err = reportStats(db.Events)
	if err != nil {
		log.Fatalf("failed to report stats: %v", err)
	}
}

func syncEvents(organization string, events []*event.Event, repo event.Repository) error {
	for _, newEvent := range events {
		gi := event.GetEventInput{UID: newEvent.UID}
		if newEvent.RecurrenceID != nil && *newEvent.RecurrenceID != "" {
			gi.RecurrenceID = newEvent.RecurrenceID
		}

		// check if event already exists in DB
		var noEventsError event.NoEventsError
		existingEvent, err := repo.GetEvent(&gi)
		if errors.As(err, &noEventsError) {
			// new event -- insert and move on
			insertErr := repo.InsertEvent(newEvent)
			if insertErr != nil {
				return insertErr
			}

			continue
		}
		if err != nil {
			return fmt.Errorf("failed to check existing event: %v", err)
		}

		// event exists
		si := event.SyncEventInput{
			Summary: &newEvent.Summary,
			Description: newEvent.Description,
			Location: newEvent.Location,
			StartTime: &newEvent.StartTime,
			EndTime: &newEvent.EndTime,
			Status: newEvent.Status,
			Transparency: newEvent.Transparency,
			Sequence: &newEvent.Sequence,
			RRule: newEvent.RRule,
			RDate: newEvent.RDate,
			ExDate: newEvent.ExDate,
		}

		if hasSignificantChanges(existingEvent, newEvent) {
			f := false
			si.Rejected = &f
		}

		err = repo.SyncEvent(&gi, &si)
		if err != nil {
			return err
		}
	}

	pi := event.PruneOrganizationEventsInput{
		Organization: organization,
		ExistingEvents: []event.GetEventInput{},
	}

	for _, e := range(events) {
		i := event.GetEventInput{UID: e.UID}
		if e.RecurrenceID != nil && *e.RecurrenceID != "" {
			i.RecurrenceID = e.RecurrenceID
		}
		pi.ExistingEvents = append(pi.ExistingEvents, i)
	}

	if err := repo.PruneOrganizationEvents(&pi); err != nil {
		fmt.Printf("Error deleting events not in source for %s: %v\n", organization, err)
	}

	return nil
}

func hasSignificantChanges(existing *event.Event, new *event.Event) bool {
  if existing.Summary != new.Summary ||
		existing.Sequence < new.Sequence {
    return true
  }

  // Check if time changed (within 1 minute tolerance)
  if !existing.StartTime.Equal(new.StartTime) {
    diff := existing.StartTime.Sub(new.StartTime)
    if diff < -time.Minute || diff > time.Minute {
      return true
    }
  }

  // Check if location changed
  existingLocation := ""
	if existing.Location != nil {
    existingLocation = *existing.Location
  }

	newLocation := ""
	if new.Location != nil {
		newLocation = *new.Location
	}

	if existingLocation != newLocation {
    return true
  }

  return false
}

func reportStats(repo event.Repository) error {
	events, err := repo.GetEvents(nil)
	if err != nil {
		return fmt.Errorf("Warning: Could not load events from database: %v", err)
	}

	fmt.Printf("\nTotal events found: %d\n", len(events))

	rejectedEvents := []*event.Event{}
	nonRejectedEvents := []*event.Event{}
	for _, e := range events {
		if e.Rejected {
			rejectedEvents = append(rejectedEvents, e)
		} else {
			nonRejectedEvents = append(nonRejectedEvents, e)
		}
	}


	fmt.Println("\n=== Rejected Status Summary ===")
	fmt.Printf("rejected: %d events\n", len(rejectedEvents))
	fmt.Printf("approved: %d events\n", len(nonRejectedEvents))

	return nil
}
