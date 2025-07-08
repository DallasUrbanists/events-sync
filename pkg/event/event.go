package event

import (
	"fmt"
	"strings"
	"time"
)

// Event represents a calendar event
type Event struct {
	Organization string    `json:"organization"`
	UID          string    `json:"uid"`
	Summary      string    `json:"summary"`
	Description  string    `json:"description"`
	Location     string    `json:"location"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Created      time.Time `json:"created"`
	Modified     time.Time `json:"modified"`
	Status       string    `json:"status"`
	Transparency string    `json:"transparency"`
}

// ParseICS parses an ICS file content and returns a slice of events
func ParseICS(content string, organization string) ([]Event, error) {
	var events []Event
	lines := strings.Split(content, "\n")

	var currentEvent *Event
	var currentValue string
	var currentKey string
	var inMultiLineValue bool

	for i, line := range lines {
		// Check if this is a continuation line (starts with space or tab)
		isContinuation := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")

		// If we're in a multi-line value and this is a continuation line
		if inMultiLineValue && isContinuation {
			// Add the continuation content (without the leading space/tab)
			currentValue += line[1:]
			continue
		}

		// If we're in a multi-line value but this is not a continuation line,
		// we've reached the end of the multi-line value
		if inMultiLineValue && !isContinuation {
			// Process the completed key-value pair
			if currentKey != "" && currentEvent != nil {
				processEventField(currentEvent, currentKey, currentValue)
			}
			currentKey = ""
			currentValue = ""
			inMultiLineValue = false
		}

		// Skip empty lines
		if strings.TrimSpace(line) == "" {
			continue
		}

		// Parse new line
		if strings.Contains(line, ":") {
			parts := strings.SplitN(line, ":", 2)
			if len(parts) == 2 {
				currentKey = strings.TrimSpace(parts[0])
				currentValue = strings.TrimSpace(parts[1])

				// Check if the next line is a continuation
				if i+1 < len(lines) && (strings.HasPrefix(lines[i+1], " ") || strings.HasPrefix(lines[i+1], "\t")) {
					inMultiLineValue = true
					continue
				}

				// Single line value - process immediately
				if currentEvent != nil {
					processEventField(currentEvent, currentKey, currentValue)
				}
				currentKey = ""
				currentValue = ""
			}
		}

		// Start new event
		if strings.TrimSpace(line) == "BEGIN:VEVENT" {
			currentEvent = &Event{Organization: organization}
		}

		// End event
		if strings.TrimSpace(line) == "END:VEVENT" {
			if currentEvent != nil {
				// Process any remaining key-value pair
				if currentKey != "" {
					processEventField(currentEvent, currentKey, currentValue)
				}
				events = append(events, *currentEvent)
				currentEvent = nil
			}
			currentKey = ""
			currentValue = ""
			inMultiLineValue = false
		}
	}

	// Process any remaining key-value pair at the end
	if currentKey != "" && currentEvent != nil {
		processEventField(currentEvent, currentKey, currentValue)
	}

	return events, nil
}

// processEventField processes a single field and sets it on the event
func processEventField(event *Event, key, value string) {
	if event == nil {
		return
	}

	// Remove any parameters from the key (e.g., "DTSTART;TZID=America/Chicago")
	key = strings.Split(key, ";")[0]

	switch key {
	case "UID":
		event.UID = value
	case "SUMMARY":
		event.Summary = value
	case "DESCRIPTION":
		event.Description = value
	case "LOCATION":
		event.Location = value
	case "DTSTART":
		if t, err := parseDateTime(value); err == nil {
			event.StartTime = t
		}
	case "DTEND":
		if t, err := parseDateTime(value); err == nil {
			event.EndTime = t
		}
	case "CREATED":
		if t, err := parseDateTime(value); err == nil {
			event.Created = t
		}
	case "LAST-MODIFIED":
		if t, err := parseDateTime(value); err == nil {
			event.Modified = t
		}
	case "STATUS":
		event.Status = value
	case "TRANSP":
		event.Transparency = value
	}
}

// parseDateTime parses various date-time formats used in ICS files
func parseDateTime(value string) (time.Time, error) {
	// Remove any timezone info for now
	value = strings.Split(value, "TZID=")[0]
	value = strings.TrimSuffix(value, "Z")

	// Try different formats
	formats := []string{
		"20060102T150405",
		"20060102T1504",
		"20060102T15",
		"20060102",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
	}

	for _, format := range formats {
		if t, err := time.Parse(format, value); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", value)
}