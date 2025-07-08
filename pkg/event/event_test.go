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