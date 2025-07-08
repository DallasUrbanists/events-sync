package calendar

import (
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/dallasurbanists/events-sync/pkg/event"
)

func TestNewCalendar(t *testing.T) {
	cal := NewCalendar("TestOrg")

	if cal.ID != "TestOrg" {
		t.Errorf("expected ID 'TestOrg', got '%s'", cal.ID)
	}

	if len(cal.Events) != 0 {
		t.Errorf("expected empty events slice, got %d events", len(cal.Events))
	}
}

func TestCalendarFetchAndParse(t *testing.T) {
	// Serve a sample ICS file
	sampleICS := `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:test-uid-1@example.com
SUMMARY:Sample Event
DTSTART:20250101T120000Z
DTEND:20250101T130000Z
END:VEVENT
END:VCALENDAR`

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		io.WriteString(w, sampleICS)
	}))
	defer ts.Close()

	cal := NewCalendar("TestOrg")
	err := cal.FetchAndParse(ts.URL)
	if err != nil {
		t.Fatalf("FetchAndParse failed: %v", err)
	}

	if cal.GetEventCount() != 1 {
		t.Errorf("expected 1 event, got %d", cal.GetEventCount())
	}

	events := cal.GetEvents()
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	if events[0].UID != "test-uid-1@example.com" {
		t.Errorf("unexpected UID: %s", events[0].UID)
	}

	if events[0].Organization != "TestOrg" {
		t.Errorf("unexpected Organization: %s", events[0].Organization)
	}
}

func TestCalendarGetEventCount(t *testing.T) {
	cal := NewCalendar("TestOrg")

	if cal.GetEventCount() != 0 {
		t.Errorf("expected 0 events, got %d", cal.GetEventCount())
	}

	// Add some events manually
	cal.Events = []event.Event{
		{Organization: "TestOrg", UID: "1", Summary: "Event 1"},
		{Organization: "TestOrg", UID: "2", Summary: "Event 2"},
	}

	if cal.GetEventCount() != 2 {
		t.Errorf("expected 2 events, got %d", cal.GetEventCount())
	}
}

func TestCalendarGetEvents(t *testing.T) {
	cal := NewCalendar("TestOrg")

	events := cal.GetEvents()
	if len(events) != 0 {
		t.Errorf("expected empty events slice, got %d events", len(events))
	}

	// Add an event manually
	cal.Events = []event.Event{
		{Organization: "TestOrg", UID: "test-1", Summary: "Test Event"},
	}

	events = cal.GetEvents()
	if len(events) != 1 {
		t.Errorf("expected 1 event, got %d", len(events))
	}

	if events[0].UID != "test-1" {
		t.Errorf("unexpected UID: %s", events[0].UID)
	}
}