package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dallasurbanists/events-sync/internal/database"
)

type EventResponse struct {
	ID           int       `json:"id"`
	UID          string    `json:"uid"`
	Organization string    `json:"organization"`
	Summary      string    `json:"summary"`
	Description  *string   `json:"description"`
	Location     *string   `json:"location"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Rejected     bool      `json:"rejected"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type UpdateRejectedRequest struct {
	Rejected bool `json:"rejected"`
}

func (s *Server) getUpcomingEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.db.GetUpcomingEvents()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var result []EventResponse
	for _, event := range events {
		eventResp := EventResponse{
			ID:           event.ID,
			UID:          event.UID,
			Organization: event.Organization,
			Summary:      event.Summary,
			Description:  event.Description,
			Location:     event.Location,
			StartTime:    event.StartTime,
			EndTime:      event.EndTime,
			Rejected:     event.Rejected,
			CreatedAt:    event.CreatedAt,
			UpdatedAt:    event.UpdatedAt,
		}
		result = append(result, eventResp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) updateEventStatus(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")

	var req UpdateRejectedRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateRejectedStatus(uid, req.Rejected); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update rejected status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) getEventStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]int)

	// Get rejected events
	rejectedEvents, err := s.db.GetEventsByRejectedStatus(true)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get rejected events: %v", err), http.StatusInternalServerError)
		return
	}
	stats["rejected"] = len(rejectedEvents)

	// Get non-rejected events
	nonRejectedEvents, err := s.db.GetEventsByRejectedStatus(false)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get non-rejected events: %v", err), http.StatusInternalServerError)
		return
	}
	stats["approved"] = len(nonRejectedEvents)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) generateICal(w http.ResponseWriter, r *http.Request) {
	// Get all events from the database
	events, err := s.db.GetEvents()
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

func generateICalContent(events []database.Event) string {
	var builder strings.Builder

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

		// Required fields
		builder.WriteString(fmt.Sprintf("UID:%s\r\n", escapeICalText(event.UID)))
		builder.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", time.Now().UTC().Format("20060102T150405Z")))
		builder.WriteString(fmt.Sprintf("DTSTART:%s\r\n", event.StartTime.UTC().Format("20060102T150405Z")))
		builder.WriteString(fmt.Sprintf("DTEND:%s\r\n", event.EndTime.UTC().Format("20060102T150405Z")))

		// Optional fields
		if event.Summary != "" {
			builder.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", escapeICalText(event.Summary)))
		}

		if event.Description != nil && *event.Description != "" {
			builder.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", escapeICalText(*event.Description)))
		}

		if event.Location != nil && *event.Location != "" {
			builder.WriteString(fmt.Sprintf("LOCATION:%s\r\n", escapeICalText(*event.Location)))
		}

		if event.Organization != "" {
			builder.WriteString(fmt.Sprintf("X-ORGANIZING-GROUP:%s\r\n", escapeICalText(event.Organization)))
		}

		// Add rejected status as a custom property
		builder.WriteString(fmt.Sprintf("X-REJECTED:%t\r\n", event.Rejected))

		// Add created and modified times if available
		builder.WriteString(fmt.Sprintf("CREATED:%s\r\n", event.CreatedAt.UTC().Format("20060102T150405Z")))
		builder.WriteString(fmt.Sprintf("LAST-MODIFIED:%s\r\n", event.UpdatedAt.UTC().Format("20060102T150405Z")))

		builder.WriteString("END:VEVENT\r\n")
	}

	// Write iCal footer
	builder.WriteString("END:VCALENDAR\r\n")

	return builder.String()
}

// escapeICalText escapes special characters in iCal text fields
func escapeICalText(text string) string {
	// Replace backslashes with double backslashes
	text = strings.ReplaceAll(text, "\\", "\\\\")

	// Replace semicolons with backslash-semicolon
	text = strings.ReplaceAll(text, ";", "\\;")

	// Replace commas with backslash-comma
	text = strings.ReplaceAll(text, ",", "\\,")

	// Replace newlines with \n
	text = strings.ReplaceAll(text, "\n", "\\n")
	text = strings.ReplaceAll(text, "\r", "\\r")

	return text
}

