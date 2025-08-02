package event

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseICS(t *testing.T) {
	icsPath := filepath.Join("..", "..", "testdata", "sample.ics")
	data, err := os.ReadFile(icsPath)
	if err != nil {
		t.Fatalf("failed to read ICS file: %v", err)
	}

	events, err := ParseICS(string(data), "TestOrg")
	if err != nil {
		t.Fatalf("ParseICS failed: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	e := events[0]
	if e.UID != "test-uid-1234@example.com" {
		t.Errorf("unexpected UID: %s", e.UID)
	}
	if e.Summary != "Test Event" {
		t.Errorf("unexpected Summary: %s", e.Summary)
	}
	if e.Organization != "TestOrg" {
		t.Errorf("unexpected Organization: %s", e.Organization)
	}
	if e.Location != "Test Location" {
		t.Errorf("unexpected Location: %s", e.Location)
	}
	if e.Status != "CONFIRMED" {
		t.Errorf("unexpected Status: %s", e.Status)
	}
	if e.Transparency != "OPAQUE" {
		t.Errorf("unexpected Transparency: %s", e.Transparency)
	}
	if !e.StartTime.Equal(time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected StartTime: %v", e.StartTime)
	}
	if !e.EndTime.Equal(time.Date(2025, 1, 1, 13, 0, 0, 0, time.UTC)) {
		t.Errorf("unexpected EndTime: %v", e.EndTime)
	}
}

func TestParseICSWithRecurrence(t *testing.T) {
	icsPath := filepath.Join("..", "..", "testdata", "sample_with_recurrence.ics")
	data, err := os.ReadFile(icsPath)
	if err != nil {
		t.Fatalf("failed to read ICS file: %v", err)
	}

	events, err := ParseICS(string(data), "TestOrg")
	if err != nil {
		t.Fatalf("ParseICS failed: %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("expected 3 events, got %d", len(events))
	}

	// Test first event (recurring event)
	e1 := events[0]
	if e1.UID != "recurring-event-1234@example.com" {
		t.Errorf("unexpected UID: %s", e1.UID)
	}
	if e1.Summary != "Recurring Test Event" {
		t.Errorf("unexpected Summary: %s", e1.Summary)
	}
	if e1.Sequence != 2 {
		t.Errorf("unexpected Sequence: %d", e1.Sequence)
	}
	if e1.RRule != "FREQ=WEEKLY;COUNT=10" {
		t.Errorf("unexpected RRule: %s", e1.RRule)
	}
	if e1.RecurrenceID != "" {
		t.Errorf("unexpected RecurrenceID: %s", e1.RecurrenceID)
	}

	// Test second event (modified instance)
	e2 := events[1]
	if e2.UID != "recurring-event-1234@example.com" {
		t.Errorf("unexpected UID: %s", e2.UID)
	}
	if e2.Summary != "Modified Recurring Event" {
		t.Errorf("unexpected Summary: %s", e2.Summary)
	}
	if e2.Sequence != 3 {
		t.Errorf("unexpected Sequence: %d", e2.Sequence)
	}
	if e2.RecurrenceID != "20250108T120000Z" {
		t.Errorf("unexpected RecurrenceID: %s", e2.RecurrenceID)
	}

	// Test third event (simple event)
	e3 := events[2]
	if e3.UID != "simple-event-5678@example.com" {
		t.Errorf("unexpected UID: %s", e3.UID)
	}
	if e3.Summary != "Simple Event" {
		t.Errorf("unexpected Summary: %s", e3.Summary)
	}
	if e3.Sequence != 1 {
		t.Errorf("unexpected Sequence: %d", e3.Sequence)
	}
}

func TestParseDateTime(t *testing.T) {
	cases := []struct {
		input    string
		expected time.Time
		ok       bool
	}{
		{"20250101T120000", time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), true},
		{"20250101T120000Z", time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC), true},
		{"20250101", time.Date(2025, 1, 1, 0, 0, 0, 0, time.UTC), true},
		{"badformat", time.Time{}, false},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got, err := parseDateTime(c.input)
			if c.ok && err != nil {
				t.Errorf("expected success, got error: %v", err)
			}
			if !c.ok && err == nil {
				t.Errorf("expected error, got success")
			}
			if c.ok && !got.Equal(c.expected) {
				t.Errorf("expected %v, got %v", c.expected, got)
			}
		})
	}
}

func TestParseSequence(t *testing.T) {
	cases := []struct {
		input    string
		expected int
		ok       bool
	}{
		{"0", 0, true},
		{"1", 1, true},
		{"42", 42, true},
		{"badformat", 0, false},
		{"", 0, false},
	}
	for _, c := range cases {
		t.Run(c.input, func(t *testing.T) {
			got, err := parseSequence(c.input)
			if c.ok && err != nil {
				t.Errorf("expected success, got error: %v", err)
			}
			if !c.ok && err == nil {
				t.Errorf("expected error, got success")
			}
			if c.ok && got != c.expected {
				t.Errorf("expected %d, got %d", c.expected, got)
			}
		})
	}
}