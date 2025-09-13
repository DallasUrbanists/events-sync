package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dallasurbanists/events-sync/pkg/event"
)

type EventResponse struct {
	UID          string     `json:"uid"`
	Organization string     `json:"organization"`
	Summary      string     `json:"summary"`
	Description  *string    `json:"description"`
	Location     *string    `json:"location"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      time.Time  `json:"end_time"`
	Rejected     bool       `json:"rejected"`
	RecurrenceID *string    `json:"recurrence_id"`
	RRule        *string    `json:"rrule"`
	RDate        *string    `json:"rdate"`
	ExDate       *string    `json:"exdate"`
	Created      *time.Time `json:"created"`
	Modified     *time.Time `json:"modified"`
	Type         string     `json:"type"`
}

type UpdateEventRequest struct {
	RecurrenceID string  `json:"recurrence_id"`
	Rejected     *bool   `json:"rejected,omitempty"`
	Organization *string `json:"organization,omitempty"`
	Type         *string `json:"type,omitempty"`
}

func (s *Server) getUpcomingEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.db.Events.GetEvents(&event.GetEventsInput{UpcomingOnly: true})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var result []EventResponse
	for _, event := range events {
		eventResp := EventResponse{
			UID:          event.UID,
			Organization: event.Organization,
			Summary:      event.Summary,
			Description:  event.Description,
			Location:     event.Location,
			StartTime:    event.StartTime,
			EndTime:      event.EndTime,
			Rejected:     event.Rejected,
			RecurrenceID: event.RecurrenceID,
			RRule:        event.RRule,
			RDate:        event.RDate,
			ExDate:       event.ExDate,
			Created:      event.Created,
			Modified:     event.Modified,
			Type:         event.Type,
		}
		result = append(result, eventResp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) updateEvent(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")

	// First decode to raw data to validate fields
	var rawData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawData); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate that only allowed fields are being updated
	allowedFields := map[string]bool{"recurrence_id": true, "rejected": true, "organization": true, "type": true}
	for key := range rawData {
		if !allowedFields[key] {
			http.Error(w, fmt.Sprintf("Field '%s' is not allowed to be updated", key), http.StatusBadRequest)
			return
		}
	}

	var req UpdateEventRequest
	reqBody, err := json.Marshal(rawData)
	if err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v \n %v", reqBody, err), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(reqBody, &req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid request body: %v \n %v", reqBody, err), http.StatusBadRequest)
		return
	}

	gi := &event.GetEventInput{UID: uid}
	if req.RecurrenceID != "" {
		gi.RecurrenceID = &req.RecurrenceID
	}

	pi := &event.PatchEventInput{}

	if req.Rejected != nil {
		pi.Rejected = req.Rejected
	}

	if req.Organization != nil {
		if *req.Organization == "" {
			http.Error(w, "Organization cannot be empty", http.StatusBadRequest)
			return
		}
		pi.Organization = req.Organization
	}

	if req.Type != nil {
		if *req.Type == "" {
			http.Error(w, "Event type cannot be empty", http.StatusBadRequest)
			return
		}
		if _, ok := event.EventTypeDisplayName[*req.Type]; !ok {
			http.Error(w, "Invalid event type", http.StatusBadRequest)
			return
		}
		pi.Type = req.Type
	}

	if pi.Rejected == nil && pi.Organization == nil && pi.Type == nil {
		http.Error(w, "At least one field (rejected, organization, or type) must be provided", http.StatusBadRequest)
		return
	}

	if err := s.db.Events.PatchEvent(gi, pi); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) getEventStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]int)

	// Get rejected events
	t := true
	rejectedEvents, err := s.db.Events.GetEvents(&event.GetEventsInput{Rejected: &t})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get rejected events: %v", err), http.StatusInternalServerError)
		return
	}
	stats["rejected"] = len(rejectedEvents)

	// Get non-rejected events
	f := false
	nonRejectedEvents, err := s.db.Events.GetEvents(&event.GetEventsInput{Rejected: &f})
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get non-rejected events: %v", err), http.StatusInternalServerError)
		return
	}
	stats["approved"] = len(nonRejectedEvents)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) generateICal(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("type")

	var events []*event.Event
	var err error
	gi := &event.GetEventsInput{}

	if eventType != "" {
		if _, ok := event.EventTypeDisplayName[eventType]; !ok {
			validTypes := ""
			for _, v := range event.EventTypeDisplayName {
				validTypes += v + ", "
			}
			validTypes = validTypes[:len(validTypes)-2]
			http.Error(w, fmt.Sprintf("Invalid event type. Valid values are: %v", validTypes), http.StatusBadRequest)
			return
		}

		gi.Type = &eventType
	}

	events, err = s.db.Events.GetEvents(gi)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate iCal content
	icalContent := generateICalContent(events)

	// Set headers for iCal file download
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"events.ics\"")
	w.Header().Set("Cache-Control", "no-cache")

	// Write the iCal content
	w.Write([]byte(icalContent))
}

func generateICalContent(events []*event.Event) string {
	var builder strings.Builder

	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		panic(err)
	}

	// Write iCal header
	builder.WriteString("BEGIN:VCALENDAR\r\n")
	builder.WriteString("VERSION:2.0\r\n")
	builder.WriteString("PRODID:-//Dallas Urbanists//Events Sync//EN\r\n")
	builder.WriteString("CALSCALE:GREGORIAN\r\n")
	builder.WriteString("METHOD:PUBLISH\r\n")
	builder.WriteString("NAME:Dallas Urbanists Synced Events (V3)\r\n")
	builder.WriteString("X-WR-CALNAME:Dallas Urbanists Synced Events (V3)\r\n")
	builder.WriteString("X-WR-TIMEZONE:America/Chicago\r\n")
	builder.WriteString("BEGIN:VTIMEZONE\r\n")
	builder.WriteString("TZID:America/Chicago\r\n")
	builder.WriteString("X-LIC-LOCATION:America/Chicago\r\n")
	builder.WriteString("BEGIN:STANDARD\r\n")
	builder.WriteString("TZOFFSETFROM:-0500\r\n")
	builder.WriteString("TZOFFSETTO:-0600\r\n")
	builder.WriteString("TZNAME:CST\r\n")
	builder.WriteString("DTSTART:19701101T020000\r\n")
	builder.WriteString("RRULE:FREQ=YEARLY;BYMONTH=11;BYDAY=1SU\r\n")
	builder.WriteString("END:STANDARD\r\n")
	builder.WriteString("BEGIN:DAYLIGHT\r\n")
	builder.WriteString("TZOFFSETFROM:-0600\r\n")
	builder.WriteString("TZOFFSETTO:-0500\r\n")
	builder.WriteString("TZNAME:CDT\r\n")
	builder.WriteString("DTSTART:19700308T020000\r\n")
	builder.WriteString("RRULE:FREQ=YEARLY;BYMONTH=3;BYDAY=2SU\r\n")
	builder.WriteString("END:DAYLIGHT\r\n")
	builder.WriteString("END:VTIMEZONE\r\n")

	// Write each event
	for _, event := range events {
		if event.Rejected {
			continue
		}

		builder.WriteString("BEGIN:VEVENT\r\n")

		builder.WriteString(fmt.Sprintf("UID:%s\r\n", event.UID))
		builder.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", time.Now().UTC().Format("20060102T150405Z")))
		builder.WriteString(fmt.Sprintf("DTSTART;TZID=America/Chicago:%s\r\n", event.StartTime.In(loc).Format("20060102T150405")))
		builder.WriteString(fmt.Sprintf("DTEND;TZID=America/Chicago:%s\r\n", event.EndTime.In(loc).Format("20060102T150405")))

		// Optional fields
		if event.Summary != "" {
			builder.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", event.Summary))
		}

		if event.Description != nil && *event.Description != "" {
			builder.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", *event.Description))
		}

		if event.Location != nil && *event.Location != "" {
			builder.WriteString(fmt.Sprintf("LOCATION:%s\r\n", *event.Location))
		}

		if event.Organization != "" {
			builder.WriteString(fmt.Sprintf("X-ORGANIZING-GROUP:%s\r\n", event.Organization))
			builder.WriteString(fmt.Sprintf("X-TEAMUP-WHO:%s\r\n", event.Organization))
		}

		builder.WriteString(fmt.Sprintf("X-EVENT-TYPE:%s\r\n", event.Type))

		// Add rejected status as a custom property
		builder.WriteString(fmt.Sprintf("X-REJECTED:%t\r\n", event.Rejected))

		// Add sequence if greater than 0
		if event.Sequence > 0 {
			builder.WriteString(fmt.Sprintf("SEQUENCE:%d\r\n", event.Sequence))
		}

		// Add recurrence fields if present
		if event.RecurrenceID != nil && *event.RecurrenceID != "" {
			builder.WriteString(fmt.Sprintf("RECURRENCE-ID:%s\r\n", *event.RecurrenceID))
		}

		if event.RRule != nil && *event.RRule != "" {
			builder.WriteString(fmt.Sprintf("RRULE:%s\r\n", *event.RRule))
		}

		if event.RDate != nil && *event.RDate != "" {
			builder.WriteString(fmt.Sprintf("RDATE:%s\r\n", *event.RDate))
		}

		if event.ExDate != nil && *event.ExDate != "" {
			builder.WriteString(fmt.Sprintf("EXDATE:%s\r\n", *event.ExDate))
		}

		// Add created and modified times if available
		if event.Created != nil {
			builder.WriteString(fmt.Sprintf("CREATED:%s\r\n", event.Created.UTC().Format("20060102T150405Z")))
		}
		if event.Modified != nil {
			builder.WriteString(fmt.Sprintf("LAST-MODIFIED:%s\r\n", event.Modified.UTC().Format("20060102T150405Z")))
		}

		builder.WriteString("END:VEVENT\r\n")
	}

	// Write iCal footer
	builder.WriteString("END:VCALENDAR\r\n")

	return builder.String()
}
