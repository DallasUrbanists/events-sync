package database

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/dallasurbanists/events-sync/pkg/event"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type DB struct {
	*sqlx.DB
}

// Event represents an event in the database
type Event struct {
	ID           int       `db:"id"`
	UID          string    `db:"uid"`
	Organization string    `db:"organization"`
	Summary      string    `db:"summary"`
	Description  *string   `db:"description"`
	Location     *string   `db:"location"`
	StartTime    time.Time `db:"start_time"`
	EndTime      time.Time `db:"end_time"`
	CreatedTime  *time.Time `db:"created_time"`
	ModifiedTime *time.Time `db:"modified_time"`
	Status       *string   `db:"status"`
	Transparency *string   `db:"transparency"`
	Sequence     int       `db:"sequence"`
	RecurrenceID *string   `db:"recurrence_id"`
	RRule        *string   `db:"rrule"`
	RDate        *string   `db:"rdate"`
	ExDate       *string   `db:"exdate"`
	Rejected     bool      `db:"rejected"`
	CreatedAt    time.Time `db:"created_at"`
	UpdatedAt    time.Time `db:"updated_at"`
}

// Connect establishes a database connection
func Connect(connStr string) (*DB, error) {
	db, err := sqlx.Connect("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	// Test the connection
	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to ping database: %v", err)
	}

	return &DB{db}, nil
}

// UpsertEvent upserts an event, handling sequence-based logic and rejected status
func (db *DB) UpsertEvent(e event.Event) error {
	// Check if event exists with same UID and recurrence-ID
	var existing Event
	var err error

	if e.RecurrenceID != "" {
		// Check for existing event with same UID and recurrence-ID
		err = db.Get(&existing, "SELECT * FROM events WHERE uid = $1 AND recurrence_id = $2", e.UID, e.RecurrenceID)
	} else {
		// Check for existing event with same UID and empty recurrence-ID (represents NULL)
		err = db.Get(&existing, "SELECT * FROM events WHERE uid = $1 AND recurrence_id = ''", e.UID)
	}

	if err == sql.ErrNoRows {
		// New event - insert with not rejected status
		return db.insertEvent(e)
	} else if err != nil {
		return fmt.Errorf("failed to check existing event: %v", err)
	}

	// Event exists - check sequence number
	if e.Sequence < existing.Sequence {
		// New event has lower sequence, ignore it
		return nil
	} else if e.Sequence == existing.Sequence {
		// Same sequence, check if it needs rejected status reset
		rejected := existing.Rejected
		if hasSignificantChanges(existing, e) {
			rejected = false // Reset to not rejected when there are significant changes
		}
		// Update the event
		return db.updateEvent(e, rejected)
	} else {
		// Higher sequence, always update
		return db.updateEvent(e, false)
	}
}

// insertEvent inserts a new event
func (db *DB) insertEvent(e event.Event) error {
	query := `
		INSERT INTO events (
			uid, organization, summary, description, location,
			start_time, end_time, created_time, modified_time,
			status, transparency, sequence, recurrence_id, rrule, rdate, exdate, rejected
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17
		)
	`

	_, err := db.Exec(query,
		e.UID, e.Organization, e.Summary, e.Description, e.Location,
		e.StartTime, e.EndTime, e.Created, e.Modified,
		e.Status, e.Transparency, e.Sequence, e.RecurrenceID, e.RRule, e.RDate, e.ExDate, false,
	)

	if err != nil {
		return fmt.Errorf("failed to insert event: %v", err)
	}

	return nil
}

// updateEvent updates an existing event
func (db *DB) updateEvent(e event.Event, rejected bool) error {
	var query string
	var args []interface{}

	if e.RecurrenceID != "" {
		query = `
			UPDATE events SET
				organization = $1,
				summary = $2,
				description = $3,
				location = $4,
				start_time = $5,
				end_time = $6,
				created_time = $7,
				modified_time = $8,
				status = $9,
				transparency = $10,
				sequence = $11,
				recurrence_id = $12,
				rrule = $13,
				rdate = $14,
				exdate = $15,
				rejected = $16
			WHERE uid = $17 AND recurrence_id = $18
		`
		args = []interface{}{
			e.Organization, e.Summary, e.Description, e.Location,
			e.StartTime, e.EndTime, e.Created, e.Modified,
			e.Status, e.Transparency, e.Sequence, e.RecurrenceID, e.RRule, e.RDate, e.ExDate, rejected, e.UID, e.RecurrenceID,
		}
	} else {
		query = `
			UPDATE events SET
				organization = $1,
				summary = $2,
				description = $3,
				location = $4,
				start_time = $5,
				end_time = $6,
				created_time = $7,
				modified_time = $8,
				status = $9,
				transparency = $10,
				sequence = $11,
				recurrence_id = $12,
				rrule = $13,
				rdate = $14,
				exdate = $15,
				rejected = $16
			WHERE uid = $17 AND recurrence_id = ''
		`
		args = []interface{}{
			e.Organization, e.Summary, e.Description, e.Location,
			e.StartTime, e.EndTime, e.Created, e.Modified,
			e.Status, e.Transparency, e.Sequence, e.RecurrenceID, e.RRule, e.RDate, e.ExDate, rejected, e.UID,
		}
	}

	_, err := db.Exec(query, args...)

	if err != nil {
		return fmt.Errorf("failed to update event: %v", err)
	}

	return nil
}

// hasSignificantChanges checks if an event has significant changes that require review
func hasSignificantChanges(existing Event, new event.Event) bool {
	// Check if title (summary) changed
	if existing.Summary != new.Summary {
		return true
	}

	// Check if the Sequence has been updated
	if existing.Sequence < new.Sequence {
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
	existingLoc := ""
	if existing.Location != nil {
		existingLoc = *existing.Location
	}
	if existingLoc != new.Location {
		return true
	}

	return false
}

// GetEvents retrieves all events from the database
func (db *DB) GetEvents() ([]Event, error) {
	var events []Event
	err := db.Select(&events, "SELECT * FROM events ORDER BY start_time")
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %v", err)
	}
	return events, nil
}

func (db *DB) GetUpcomingEvents() ([]Event, error) {
	var events []Event
	err := db.Select(&events, "SELECT * FROM events WHERE start_time > NOW() ORDER BY start_time")
	if err != nil {
		return nil, fmt.Errorf("failed to get events: %v", err)
	}
	return events, nil
}

// GetEventsByRejectedStatus retrieves events by rejected status
func (db *DB) GetEventsByRejectedStatus(rejected bool) ([]Event, error) {
	var events []Event
	err := db.Select(&events, "SELECT * FROM events WHERE rejected = $1 ORDER BY start_time", rejected)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by rejected status: %v", err)
	}
	return events, nil
}

// UpdateRejectedStatus updates the rejected status of an event
func (db *DB) UpdateRejectedStatus(uid string, rejected bool) error {
	_, err := db.Exec("UPDATE events SET rejected = $1 WHERE uid = $2", rejected, uid)
	if err != nil {
		return fmt.Errorf("failed to update rejected status: %v", err)
	}
	return nil
}

// MarkPastEventsAsReviewed automatically marks all events in the past as not rejected
func (db *DB) MarkPastEventsAsReviewed() error {
	now := time.Now()
	result, err := db.Exec("UPDATE events SET rejected = false WHERE end_time < $1 AND rejected = true", now)
	if err != nil {
		return fmt.Errorf("failed to mark past events as not rejected: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("Marked %d past events as not rejected\n", rowsAffected)
	}

	return nil
}

// DeleteEventsNotInSource deletes events for a specific organization that are not in the provided source events
func (db *DB) DeleteEventsNotInSource(organization string, sourceEvents []event.Event) error {
	// Create a map of source event UIDs for efficient lookup
	sourceEventMap := make(map[string]bool)
	for _, e := range sourceEvents {
		key := e.UID
		if e.RecurrenceID != "" {
			key = e.UID + ":" + e.RecurrenceID
		}
		sourceEventMap[key] = true
	}

	// Get all events for this organization from the database
	var dbEvents []Event
	err := db.Select(&dbEvents, "SELECT * FROM events WHERE organization = $1", organization)
	if err != nil {
		return fmt.Errorf("failed to get events for organization %s: %v", organization, err)
	}

	// Find events to delete (those in DB but not in source)
	var eventsToDelete []Event
	for _, dbEvent := range dbEvents {
		key := dbEvent.UID
		if dbEvent.RecurrenceID != nil && *dbEvent.RecurrenceID != "" {
			key = dbEvent.UID + ":" + *dbEvent.RecurrenceID
		}

		if !sourceEventMap[key] {
			eventsToDelete = append(eventsToDelete, dbEvent)
		}
	}

	// Delete events that are no longer in the source
	if len(eventsToDelete) > 0 {
		fmt.Printf("Deleting %d events for organization %s that are no longer in source calendar:\n", len(eventsToDelete), organization)

		// Log summaries of events being deleted
		for i, dbEvent := range eventsToDelete {
			summary := dbEvent.Summary
			if len(summary) > 50 {
				summary = summary[:47] + "..."
			}
			fmt.Printf("  %d. %s (UID: %s)\n", i+1, summary, dbEvent.UID)
		}

		// Delete events by UID and recurrence_id
		for _, dbEvent := range eventsToDelete {
			key := dbEvent.UID
			if dbEvent.RecurrenceID != nil && *dbEvent.RecurrenceID != "" {
				key = dbEvent.UID + ":" + *dbEvent.RecurrenceID
			}

			if dbEvent.RecurrenceID != nil && *dbEvent.RecurrenceID != "" {
				// Event with recurrence_id
				_, err := db.Exec("DELETE FROM events WHERE uid = $1 AND recurrence_id = $2 AND organization = $3", dbEvent.UID, *dbEvent.RecurrenceID, organization)
				if err != nil {
					fmt.Printf("Error deleting event %s: %v\n", key, err)
				}
			} else {
				// Event without recurrence_id
				_, err := db.Exec("DELETE FROM events WHERE uid = $1 AND (recurrence_id = '' OR recurrence_id IS NULL) AND organization = $2", dbEvent.UID, organization)
				if err != nil {
					fmt.Printf("Error deleting event %s: %v\n", key, err)
				}
			}
		}
	}

	return nil
}

// AuthenticatedDiscordUser represents an authenticated Discord user
type AuthenticatedDiscordUser struct {
	ID         int       `db:"id"`
	DiscordID  string    `db:"discord_id"`
	Username   string    `db:"username"`
	CreatedAt  time.Time `db:"created_at"`
	UpdatedAt  time.Time `db:"updated_at"`
}

// IsDiscordUserAuthenticated checks if a Discord user ID is in the authenticated users table
func (db *DB) IsDiscordUserAuthenticated(discordID string) (bool, error) {
	var count int
	err := db.Get(&count, "SELECT COUNT(*) FROM authenticated_discord_users WHERE discord_id = $1", discordID)
	if err != nil {
		return false, fmt.Errorf("failed to check if Discord user is authenticated: %v", err)
	}
	return count > 0, nil
}

// GetDiscordUserByID retrieves a Discord user by their Discord ID
func (db *DB) GetDiscordUserByID(discordID string) (*AuthenticatedDiscordUser, error) {
	var user AuthenticatedDiscordUser
	err := db.Get(&user, "SELECT * FROM authenticated_discord_users WHERE discord_id = $1", discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get Discord user: %v", err)
	}
	return &user, nil
}
