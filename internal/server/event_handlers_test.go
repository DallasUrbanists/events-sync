package server

import (
	"strings"
	"testing"
	"time"

	"github.com/dallasurbanists/events-sync/internal/database"
)

func TestGenerateICalContent(t *testing.T) {
	// Create sample events
	description1 := "Test event description"
	location1 := "Test Location"
	description2 := "Another test event"
	description3 := "Rejected event"

	events := []database.Event{
		{
			ID:           1,
			UID:          "test-uid-1@example.com",
			Organization: "TestOrg",
			Summary:      "Test Event 1",
			Description:  &description1,
			Location:     &location1,
			StartTime:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			EndTime:      time.Date(2025, 1, 1, 13, 0, 0, 0, time.UTC),
			Rejected:     false,
			CreatedAt:    time.Date(2024, 12, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 12, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:           2,
			UID:          "test-uid-2@example.com",
			Organization: "TestOrg",
			Summary:      "Test Event 2",
			Description:  &description2,
			Location:     nil,
			StartTime:    time.Date(2025, 1, 2, 14, 0, 0, 0, time.UTC),
			EndTime:      time.Date(2025, 1, 2, 15, 0, 0, 0, time.UTC),
			Rejected:     false,
			CreatedAt:    time.Date(2024, 12, 1, 11, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 12, 1, 11, 0, 0, 0, time.UTC),
		},
		{
			ID:           3,
			UID:          "test-uid-3@example.com",
			Organization: "TestOrg",
			Summary:      "Rejected Event",
			Description:  &description3,
			Location:     nil,
			StartTime:    time.Date(2025, 1, 3, 16, 0, 0, 0, time.UTC),
			EndTime:      time.Date(2025, 1, 3, 17, 0, 0, 0, time.UTC),
			Rejected:     true,
			CreatedAt:    time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 12, 1, 12, 0, 0, 0, time.UTC),
		},
	}

	// Generate iCal content
	icalContent := generateICalContent(events)

	// Verify basic structure
	if !strings.Contains(icalContent, "BEGIN:VCALENDAR") {
		t.Error("Missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(icalContent, "END:VCALENDAR") {
		t.Error("Missing END:VCALENDAR")
	}
	if !strings.Contains(icalContent, "VERSION:2.0") {
		t.Error("Missing VERSION:2.0")
	}
	if !strings.Contains(icalContent, "PRODID:-//Dallas Urbanists//Events Sync//EN") {
		t.Error("Missing updated PRODID")
	}
	if !strings.Contains(icalContent, "NAME:Dallas Urbanists Synced Events (V3)") {
		t.Error("Missing calendar name")
	}
	if !strings.Contains(icalContent, "X-WR-CALNAME:Dallas Urbanists Synced Events (V3)") {
		t.Error("Missing X-WR-CALNAME")
	}
	if !strings.Contains(icalContent, "X-WR-TIMEZONE:America/Chicago") {
		t.Error("Missing timezone")
	}

	// Verify timezone information
	if !strings.Contains(icalContent, "BEGIN:VTIMEZONE") {
		t.Error("Missing BEGIN:VTIMEZONE")
	}
	if !strings.Contains(icalContent, "TZID:America/Chicago") {
		t.Error("Missing TZID")
	}
	if !strings.Contains(icalContent, "BEGIN:STANDARD") {
		t.Error("Missing BEGIN:STANDARD")
	}
	if !strings.Contains(icalContent, "BEGIN:DAYLIGHT") {
		t.Error("Missing BEGIN:DAYLIGHT")
	}
	if !strings.Contains(icalContent, "END:VTIMEZONE") {
		t.Error("Missing END:VTIMEZONE")
	}

	// Verify events (should only include non-rejected events)
	if !strings.Contains(icalContent, "BEGIN:VEVENT") {
		t.Error("Missing BEGIN:VEVENT")
	}
	if !strings.Contains(icalContent, "END:VEVENT") {
		t.Error("Missing END:VEVENT")
	}

	// Verify specific event content (only pending and reviewed events)
	if !strings.Contains(icalContent, "UID:test-uid-1@example.com") {
		t.Error("Missing first event UID")
	}
	if !strings.Contains(icalContent, "UID:test-uid-2@example.com") {
		t.Error("Missing second event UID")
	}
	if strings.Contains(icalContent, "UID:test-uid-3@example.com") {
		t.Error("Rejected event should not be included")
	}
	if !strings.Contains(icalContent, "SUMMARY:Test Event 1") {
		t.Error("Missing first event summary")
	}
	if !strings.Contains(icalContent, "SUMMARY:Test Event 2") {
		t.Error("Missing second event summary")
	}
	if strings.Contains(icalContent, "SUMMARY:Rejected Event") {
		t.Error("Rejected event summary should not be included")
	}
	if !strings.Contains(icalContent, "DESCRIPTION:Test event description") {
		t.Error("Missing first event description")
	}
	if !strings.Contains(icalContent, "LOCATION:Test Location") {
		t.Error("Missing first event location")
	}
	if !strings.Contains(icalContent, "X-REJECTED:false") {
		t.Error("Missing first event rejected status")
	}
	if !strings.Contains(icalContent, "X-REJECTED:false") {
		t.Error("Missing second event rejected status")
	}

	// Verify organization field format change
	if !strings.Contains(icalContent, "X-ORGANIZING-GROUP:TestOrg") {
		t.Error("Missing X-ORGANIZING-GROUP field")
	}
	if strings.Contains(icalContent, "ORGANIZER;CN=TestOrg:mailto:noreply@events-sync.com") {
		t.Error("Old ORGANIZER format should not be present")
	}

	// Verify date formats
	if !strings.Contains(icalContent, "DTSTART:20250101T120000Z") {
		t.Error("Missing or incorrect DTSTART for first event")
	}
	if !strings.Contains(icalContent, "DTEND:20250101T130000Z") {
		t.Error("Missing or incorrect DTEND for first event")
	}
	if !strings.Contains(icalContent, "DTSTART:20250102T140000Z") {
		t.Error("Missing or incorrect DTSTART for second event")
	}
	if !strings.Contains(icalContent, "DTEND:20250102T150000Z") {
		t.Error("Missing or incorrect DTEND for second event")
	}

	// Verify line endings
	lines := strings.Split(icalContent, "\r\n")
	if len(lines) < 20 {
		t.Error("iCal content seems too short")
	}

	// Verify no empty lines in the middle
	for i, line := range lines {
		if i > 0 && i < len(lines)-1 && line == "" {
			t.Error("Found empty line in middle of iCal content")
		}
	}
}

func TestEscapeICalText(t *testing.T) {
	cases := []struct {
		input    string
		expected string
	}{
		{"Normal text", "Normal text"},
		{"Text with; semicolon", "Text with\\; semicolon"},
		{"Text with, comma", "Text with\\, comma"},
		{"Text with\\ backslash", "Text with\\\\ backslash"},
		{"Text with\n newline", "Text with\\n newline"},
		{"Text with\r carriage return", "Text with\\r carriage return"},
		{"Complex: text; with, multiple\\ characters\n", "Complex: text\\; with\\, multiple\\\\ characters\\n"},
	}

	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got := escapeICalText(c.input)
			if got != c.expected {
				t.Errorf("escapeICalText(%q) = %q, want %q", c.input, got, c.expected)
			}
		})
	}
}

func TestGenerateICalContentWithRecurrence(t *testing.T) {
	// Create sample events with recurrence fields
	description1 := "Recurring test event"
	location1 := "Test Location"
	rrule1 := "FREQ=WEEKLY;COUNT=10"
	recurrenceID1 := "20250108T120000Z"
	sequence1 := 2

	events := []database.Event{
		{
			ID:           1,
			UID:          "recurring-event-1234@example.com",
			Organization: "TestOrg",
			Summary:      "Recurring Test Event",
			Description:  &description1,
			Location:     &location1,
			StartTime:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			EndTime:      time.Date(2025, 1, 1, 13, 0, 0, 0, time.UTC),
			Sequence:     sequence1,
			RRule:        &rrule1,
			Rejected:     false,
			CreatedAt:    time.Date(2024, 12, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 12, 1, 10, 0, 0, 0, time.UTC),
		},
		{
			ID:           2,
			UID:          "recurring-event-1234@example.com",
			Organization: "TestOrg",
			Summary:      "Modified Recurring Event",
			Description:  &description1,
			Location:     &location1,
			StartTime:    time.Date(2025, 1, 8, 12, 0, 0, 0, time.UTC),
			EndTime:      time.Date(2025, 1, 8, 13, 0, 0, 0, time.UTC),
			Sequence:     3,
			RecurrenceID: &recurrenceID1,
			Rejected:     false,
			CreatedAt:    time.Date(2024, 12, 1, 11, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 12, 1, 11, 0, 0, 0, time.UTC),
		},
	}

	// Generate iCal content
	icalContent := generateICalContent(events)

	// Verify recurrence fields are included
	if !strings.Contains(icalContent, "SEQUENCE:2") {
		t.Error("Missing SEQUENCE field for first event")
	}
	if !strings.Contains(icalContent, "RRULE:FREQ=WEEKLY;COUNT=10") {
		t.Error("Missing RRULE field for first event")
	}
	if !strings.Contains(icalContent, "SEQUENCE:3") {
		t.Error("Missing SEQUENCE field for second event")
	}
	if !strings.Contains(icalContent, "RECURRENCE-ID:20250108T120000Z") {
		t.Error("Missing RECURRENCE-ID field for second event")
	}

	// Verify events are included
	if !strings.Contains(icalContent, "UID:recurring-event-1234@example.com") {
		t.Error("Missing recurring event UID")
	}
	if !strings.Contains(icalContent, "SUMMARY:Recurring Test Event") {
		t.Error("Missing first event summary")
	}
	if !strings.Contains(icalContent, "SUMMARY:Modified Recurring Event") {
		t.Error("Missing second event summary")
	}
}

func TestICalEndpoint(t *testing.T) {
	// Create a test database with some events
	// This is a simplified test that just verifies the endpoint structure
	// In a real scenario, you'd want to use a test database

	// Create sample events with recurrence fields
	description1 := "Recurring test event"
	location1 := "Test Location"
	rrule1 := "FREQ=WEEKLY;COUNT=10"

	events := []database.Event{
		{
			ID:           1,
			UID:          "recurring-event-1234@example.com",
			Organization: "TestOrg",
			Summary:      "Recurring Test Event",
			Description:  &description1,
			Location:     &location1,
			StartTime:    time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC),
			EndTime:      time.Date(2025, 1, 1, 13, 0, 0, 0, time.UTC),
			Sequence:     2,
			RRule:        &rrule1,
			Rejected:     false,
			CreatedAt:    time.Date(2024, 12, 1, 10, 0, 0, 0, time.UTC),
			UpdatedAt:    time.Date(2024, 12, 1, 10, 0, 0, 0, time.UTC),
		},
	}

	// Generate iCal content
	icalContent := generateICalContent(events)

	// Verify the content includes the recurrence fields
	if !strings.Contains(icalContent, "SEQUENCE:2") {
		t.Error("Missing SEQUENCE field")
	}
	if !strings.Contains(icalContent, "RRULE:FREQ=WEEKLY;COUNT=10") {
		t.Error("Missing RRULE field")
	}
	if !strings.Contains(icalContent, "UID:recurring-event-1234@example.com") {
		t.Error("Missing event UID")
	}

	// Verify the content has the correct iCal structure
	if !strings.Contains(icalContent, "BEGIN:VCALENDAR") {
		t.Error("Missing BEGIN:VCALENDAR")
	}
	if !strings.Contains(icalContent, "END:VCALENDAR") {
		t.Error("Missing END:VCALENDAR")
	}
	if !strings.Contains(icalContent, "BEGIN:VEVENT") {
		t.Error("Missing BEGIN:VEVENT")
	}
	if !strings.Contains(icalContent, "END:VEVENT") {
		t.Error("Missing END:VEVENT")
	}
}