package event

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
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

func ParseICS(content string, organization string) ([]Event, error) {
	var events []Event
	lines := strings.Split(content, "\n")

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
				processEventField(currentEvent, currentKey, currentValue)
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
					processEventField(currentEvent, currentKey, currentValue)
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
					processEventField(currentEvent, currentKey, currentValue)
				}
				currentKey = ""
				currentValue = ""
			}
		}
	}

	if currentKey != "" && currentEvent != nil {
		processEventField(currentEvent, currentKey, currentValue)
	}

	return events, nil
}

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

// parseSequence parses the SEQUENCE field from ICS
func parseSequence(value string) (int, error) {
	var seq int
	_, err := fmt.Sscanf(value, "%d", &seq)
	if err != nil {
		return 0, fmt.Errorf("unable to parse sequence: %s", value)
	}
	return seq, nil
}

func FetchAndParseEvents(url string, organization string) ([]Event, error) {
	fmt.Printf("Fetching ICS file from: %s\n", url)

	content, err := fetchICS(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching ICS: %v", err)
	}

	events, err := ParseICS(content, organization)
	if err != nil {
		return nil, fmt.Errorf("error parsing ICS: %v", err)
	}

	return events, nil
}

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