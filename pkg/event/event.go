package event

import (
	"fmt"
	"strings"
	"time"
	_ "time/tzdata"
)

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
	Sequence     int       `json:"sequence"`
	RecurrenceID string    `json:"recurrence_id"`
	RRule        string    `json:"rrule"`
	RDate        string    `json:"rdate"`
	ExDate       string    `json:"exdate"`
}

type parseUtils struct {
	defaultLoc *time.Location
}

func ParseICS(content string, organization string) ([]Event, error) {
	var events []Event

	defaultLoc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return nil, err
	}

	u := parseUtils{
		defaultLoc: defaultLoc,
	}

	lines := strings.Split(content, "\r\n")

	var currentEvent *Event
	var currentValue string
	var currentKey string
	var inMultiLineValue bool

	for i, line := range lines {
		isContinuation := strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t")

		if inMultiLineValue && isContinuation {
			currentValue += line[1:]
			continue
		}

		// If we're in a multi-line value but this is not a continuation line,
		// we've reached the end of the multi-line value. Process it, then continue
		// processing the current line
		if inMultiLineValue && !isContinuation {
			if currentKey != "" && currentEvent != nil {
				processEventField(currentEvent, u, currentKey, currentValue)
			}
			currentKey = ""
			currentValue = ""
			inMultiLineValue = false
		}

		if strings.TrimSpace(line) == "" {
			continue
		}

		if strings.TrimSpace(line) == "BEGIN:VEVENT" {
			currentEvent = &Event{Organization: organization}
			continue
		}

		if strings.TrimSpace(line) == "END:VEVENT" {
			if currentEvent != nil {
				// Process any remaining key-value pair
				if currentKey != "" {
					processEventField(currentEvent, u, currentKey, currentValue)
				}
				events = append(events, *currentEvent)
				currentEvent = nil
			}
			currentKey = ""
			currentValue = ""
			inMultiLineValue = false

			continue
		}

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
					processEventField(currentEvent, u, currentKey, currentValue)
				}
				currentKey = ""
				currentValue = ""
			}
		}
	}

	if currentKey != "" && currentEvent != nil {
		processEventField(currentEvent, u, currentKey, currentValue)
	}

	return events, nil
}

func processEventField(event *Event, u parseUtils, key, value string) {
	if event == nil {
		return
	}

	keyParts := strings.Split(key, ";")
	keyPrefix := keyParts[0]

	switch keyPrefix {
	case "UID":
		event.UID = value
	case "SUMMARY":
		event.Summary = value
	case "DESCRIPTION":
		event.Description = value
	case "LOCATION":
		event.Location = value
	case "DTSTART":
		if t, err := parseDateTime(value, u, keyParts[1:]); err == nil {
			event.StartTime = t
		}
	case "DTEND":
		if t, err := parseDateTime(value, u, keyParts[1:]); err == nil {
			event.EndTime = t
		}
	case "CREATED":
		if t, err := parseDateTime(value, u, keyParts[1:]); err == nil {
			event.Created = t
		}
	case "LAST-MODIFIED":
		if t, err := parseDateTime(value, u, keyParts[1:]); err == nil {
			event.Modified = t
		}
	case "STATUS":
		event.Status = value
	case "TRANSP":
		event.Transparency = value
	case "SEQUENCE":
		if seq, err := parseSequence(value); err == nil {
			event.Sequence = seq
		}
	case "RECURRENCE-ID":
		event.RecurrenceID = value
	case "RRULE":
		event.RRule = value
	case "RDATE":
		event.RDate = value
	case "EXDATE":
		event.ExDate = value
	}
}

// parseDateTime parses various date-time formats used in ICS files
func parseDateTime(value string, u parseUtils, keyParams []string) (time.Time, error) {
	utcFmt := "20060102T150405Z"
	if strings.HasSuffix(strings.ToLower(value), "z") {
		return time.Parse(utcFmt, value)
	}

	var err error
	var zeroTime time.Time

	loc := u.defaultLoc

	for _, keyParam := range keyParams {
		if strings.HasPrefix(strings.ToLower(keyParam), "tzid=") {
			locValue := strings.Split(keyParam, "=")[1]
			loc, err = time.LoadLocation(locValue)
			if err != nil {
				return zeroTime, err
			}
			break
		}
	}

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
		if t, err := time.ParseInLocation(format, value, loc); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse date: %s", value)
}

// parseSequence parses the SEQUENCE field from ICS
func parseSequence(value string) (int, error) {
	var seq int
	_, err := fmt.Sscanf(value, "%d", &seq)
	if err != nil {
		return 0, fmt.Errorf("unable to parse sequence: %s", value)
	}
	return seq, nil
}


