package calendar

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/dallasurbanists/events-sync/internal/database"
	"github.com/dallasurbanists/events-sync/pkg/event"
)

// Calendar represents a calendar with events
type Calendar struct {
	ID     string       `json:"id"`
	Events []event.Event `json:"events"`
	DB     *database.DB `json:"-"`
}

// NewCalendar creates a new calendar with the given organization ID
func NewCalendar(organizationID string) *Calendar {
	return &Calendar{
		ID:     organizationID,
		Events: []event.Event{},
	}
}

// SetDatabase sets the database connection for the calendar
func (c *Calendar) SetDatabase(db *database.DB) {
	c.DB = db
}

// FetchAndParse fetches an ICS file from a URL and parses it into events
func (c *Calendar) FetchAndParse(url string) error {
	// Fetch the ICS file
	fmt.Printf("Fetching ICS file from: %s\n", url)

	content, err := fetchICS(url)
	if err != nil {
		return fmt.Errorf("error fetching ICS: %v", err)
	}

	// Parse the ICS content
	events, err := event.ParseICS(content, c.ID)
	if err != nil {
		return fmt.Errorf("error parsing ICS: %v", err)
	}

	c.Events = events

	// Store events in database if DB is available
	if c.DB != nil {
		return c.storeEventsInDB()
	}

	return nil
}

// storeEventsInDB stores all events in the database
func (c *Calendar) storeEventsInDB() error {
	for _, e := range c.Events {
		if err := c.DB.UpsertEvent(e); err != nil {
			return fmt.Errorf("failed to upsert event %s: %v", e.UID, err)
		}
	}
	return nil
}

// LoadFromDB loads events from the database
func (c *Calendar) LoadFromDB() error {
	if c.DB == nil {
		return fmt.Errorf("database not set")
	}

	dbEvents, err := c.DB.GetEvents()
	if err != nil {
		return fmt.Errorf("failed to load events from DB: %v", err)
	}

	// Convert database events to package events
	c.Events = make([]event.Event, len(dbEvents))
	for i, dbEvent := range dbEvents {
		c.Events[i] = databaseEventToEvent(dbEvent)
	}

	return nil
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

// fetchICS fetches an ICS file from a URL with proper headers
func fetchICS(url string) (string, error) {
	client := &http.Client{}
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("error creating request: %v", err)
	}

	// Add headers to mimic a browser request
	req.Header.Set("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36")
	req.Header.Set("Accept", "text/calendar,text/plain,*/*")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("error making request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("HTTP error: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("error reading response body: %v", err)
	}

	return string(body), nil
}

// SortEvents sorts the calendar events by start time
func (c *Calendar) SortEvents() {
	// This will be implemented when we add sorting functionality
	// For now, we'll keep the events in the order they were parsed
}

// GetEvents returns all events in the calendar
func (c *Calendar) GetEvents() []event.Event {
	return c.Events
}

// GetEventCount returns the number of events in the calendar
func (c *Calendar) GetEventCount() int {
	return len(c.Events)
}