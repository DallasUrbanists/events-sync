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
	ReviewStatus string    `db:"review_status"`
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

// UpsertEvent upserts an event, handling review status logic
func (db *DB) UpsertEvent(e event.Event) error {
	// Check if event exists
	var existing Event
	err := db.Get(&existing, "SELECT * FROM events WHERE uid = $1", e.UID)

	if err == sql.ErrNoRows {
		// New event - insert with pending status
		return db.insertEvent(e)
	} else if err != nil {
		return fmt.Errorf("failed to check existing event: %v", err)
	}

	// Event exists - check if it needs review status reset
	reviewStatus := existing.ReviewStatus
	if hasSignificantChanges(existing, e) {
		reviewStatus = "pending"
	}

	// Update the event
	return db.updateEvent(e, reviewStatus)
}

// insertEvent inserts a new event
func (db *DB) insertEvent(e event.Event) error {
	query := `
		INSERT INTO events (
			uid, organization, summary, description, location,
			start_time, end_time, created_time, modified_time,
			status, transparency, review_status
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12
		)
	`

	_, err := db.Exec(query,
		e.UID, e.Organization, e.Summary, e.Description, e.Location,
		e.StartTime, e.EndTime, e.Created, e.Modified,
		e.Status, e.Transparency, "pending",
	)

	if err != nil {
		return fmt.Errorf("failed to insert event: %v", err)
	}

	return nil
}

// updateEvent updates an existing event
func (db *DB) updateEvent(e event.Event, reviewStatus string) error {
	query := `
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
			review_status = $11
		WHERE uid = $12
	`

	_, err := db.Exec(query,
		e.Organization, e.Summary, e.Description, e.Location,
		e.StartTime, e.EndTime, e.Created, e.Modified,
		e.Status, e.Transparency, reviewStatus, e.UID,
	)

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

// GetEventsByReviewStatus retrieves events by review status
func (db *DB) GetEventsByReviewStatus(status string) ([]Event, error) {
	var events []Event
	err := db.Select(&events, "SELECT * FROM events WHERE review_status = $1 ORDER BY start_time", status)
	if err != nil {
		return nil, fmt.Errorf("failed to get events by review status: %v", err)
	}
	return events, nil
}

// UpdateReviewStatus updates the review status of an event
func (db *DB) UpdateReviewStatus(uid string, status string) error {
	_, err := db.Exec("UPDATE events SET review_status = $1 WHERE uid = $2", status, uid)
	if err != nil {
		return fmt.Errorf("failed to update review status: %v", err)
	}
	return nil
}

// MarkPastEventsAsReviewed automatically marks all events in the past as reviewed
func (db *DB) MarkPastEventsAsReviewed() error {
	now := time.Now()
	result, err := db.Exec("UPDATE events SET review_status = 'reviewed' WHERE end_time < $1 AND review_status = 'pending'", now)
	if err != nil {
		return fmt.Errorf("failed to mark past events as reviewed: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %v", err)
	}

	if rowsAffected > 0 {
		fmt.Printf("Marked %d past events as reviewed\n", rowsAffected)
	}

	return nil
}