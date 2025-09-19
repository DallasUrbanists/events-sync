package server

import (
	"encoding/json"
	"fmt"
	"log/slog"
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
	ExDateManual *string    `json:"exdate_manual"`
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
	l := s.getLogger(r)

	l.Debug("getting upcoming events")
	events, err := s.db.Events.GetEvents(&event.GetEventsInput{UpcomingOnly: true})
	if err != nil {
		l.Error(fmt.Sprintf("Failed to get events: %v", err))
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
			ExDateManual: event.ExDateManual,
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
	l := s.getLogger(r)

	l.Debug(fmt.Sprintf("updating event %v", uid))

	// First decode to raw data to validate fields
	var rawData map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&rawData); err != nil {
		l.Error(fmt.Sprintf("couldn't decode request body to update event %v : %v", uid, err))
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
		l.Error(fmt.Sprintf("Invalid request body: %v \n %v", reqBody, err))
		http.Error(w, fmt.Sprintf("Invalid request body: %v \n %v", reqBody, err), http.StatusBadRequest)
		return
	}
	if err := json.Unmarshal(reqBody, &req); err != nil {
		l.Error(fmt.Sprintf("Invalid request body, couldn't unmarshal into req: %v \n %v", reqBody, err))
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

	l.Info(fmt.Sprintf("updating event %v - %v", *gi, *pi))
	if err := s.db.Events.PatchEvent(gi, pi); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update: %v", err), http.StatusInternalServerError)
		return
	}

	if pi.Type != nil {
		l.Info(fmt.Sprintf("updating sibling event types %v - %v", *gi, *pi))
		err = s.updateEventType(gi, *pi.Type)
		if err != nil {
			l.Error(fmt.Sprintf("Failed to update sibling event types %v to %v: %v", gi, pi, err))
			http.Error(w, fmt.Sprintf("Failed to update sibling event types: %v", err), http.StatusInternalServerError)
			return
		}
	}

	if pi.Rejected != nil {
		l.Info(fmt.Sprintf("updating parent event rejection status %v - %v", *gi, *pi))
		err = s.updateRootExdate(gi, *pi.Rejected)
		if err != nil {
			l.Error(fmt.Sprintf("Failed to update root exdate for %v to %v: %v", gi, pi, err))
			http.Error(w, fmt.Sprintf("Failed to update root exdate: %v", err), http.StatusInternalServerError)
			return
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

func (s *Server) getEventStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]int)
	l := s.getLogger(r)

	l.Info("getting event stats")

	// Get rejected events
	t := true
	l.Debug("getting rejected events")
	rejectedEvents, err := s.db.Events.GetEvents(&event.GetEventsInput{Rejected: &t})
	if err != nil {
		l.Error(fmt.Sprintf("Failed to get rejected events: %v", err))

		http.Error(w, fmt.Sprintf("Failed to get rejected events: %v", err), http.StatusInternalServerError)
		return
	}
	stats["rejected"] = len(rejectedEvents)

	// Get non-rejected events
	f := false
	l.Debug("getting non-rejected events")
	nonRejectedEvents, err := s.db.Events.GetEvents(&event.GetEventsInput{Rejected: &f})
	if err != nil {
		l.Error(fmt.Sprintf("Failed to get non-rejected events: %v", err))

		http.Error(w, fmt.Sprintf("Failed to get non-rejected events: %v", err), http.StatusInternalServerError)
		return
	}
	stats["approved"] = len(nonRejectedEvents)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func (s *Server) generateICal(w http.ResponseWriter, r *http.Request) {
	eventType := r.URL.Query().Get("type")
	l := s.getLogger(r)

	l.Info("generating ical")

	var events []*event.Event
	var err error
	gi := &event.GetEventsInput{}

	if eventType != "" {
		l.Info(fmt.Sprintf("generating ical for type %v", eventType))

		if _, ok := event.EventTypeDisplayName[eventType]; !ok {
			validTypes := ""
			for k, _ := range event.EventTypeDisplayName {
				validTypes += k + ", "
			}
			validTypes = validTypes[:len(validTypes)-2]
			l.Error(fmt.Sprintf("invalid event type %v", eventType))
			http.Error(w, fmt.Sprintf("Invalid event type. Valid values are: %v", validTypes), http.StatusBadRequest)
			return
		}

		gi.Type = &eventType
	}

	l.Info(fmt.Sprintf("getting all events %v", gi))
	events, err = s.db.Events.GetEvents(gi)
	if err != nil {
		l.Error(fmt.Sprintf("Failed to get events: %v", err))
		http.Error(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	// Generate iCal content
	l.Info("generating ical content")
	icalContent, err := generateICalContent(events, l)
	if err != nil {
		l.Error(fmt.Sprintf("Failed to write calendar: %v", err))
		http.Error(w, fmt.Sprintf("Failed to write calendar: %v", err), http.StatusInternalServerError)
		return
	}

	// Set headers for iCal file download
	w.Header().Set("Content-Type", "text/calendar; charset=utf-8")
	w.Header().Set("Content-Disposition", "attachment; filename=\"events.ics\"")
	w.Header().Set("Cache-Control", "no-cache")

	// Write the iCal content
	w.Write([]byte(icalContent))
}

func generateICalContent(events []*event.Event, logger *slog.Logger) (string, error) {
	var builder strings.Builder

	l := logger
	if l == nil {
		l = slog.Default()
	}

	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return "", err
	}

	// Write iCal header
	l.Debug("writing ical preamble")

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

		identifier := event.UID
		if event.RecurrenceID != nil {
			identifier += fmt.Sprintf(" %v", event.RecurrenceID)
		}
		l.Debug(fmt.Sprintf("writing event %v", identifier))

		builder.WriteString("BEGIN:VEVENT\r\n")

		l.Debug(fmt.Sprintf("writing ID and timestamps for %v", identifier))
		builder.WriteString(fmt.Sprintf("UID:%s\r\n", event.UID))
		builder.WriteString(fmt.Sprintf("DTSTAMP:%s\r\n", time.Now().UTC().Format("20060102T150405Z")))
		builder.WriteString(fmt.Sprintf("DTSTART;TZID=America/Chicago:%s\r\n", event.StartTime.In(loc).Format("20060102T150405")))
		builder.WriteString(fmt.Sprintf("DTEND;TZID=America/Chicago:%s\r\n", event.EndTime.In(loc).Format("20060102T150405")))

		// Optional fields
		if event.Summary != "" {
			l.Debug(fmt.Sprintf("writing summary for %v", identifier))
			builder.WriteString(fmt.Sprintf("SUMMARY:%s\r\n", event.Summary))
		}

		if event.Description != nil && *event.Description != "" {
			l.Debug(fmt.Sprintf("writing description for %v", identifier))
			builder.WriteString(fmt.Sprintf("DESCRIPTION:%s\r\n", *event.Description))
		}

		if event.Location != nil && *event.Location != "" {
			l.Debug(fmt.Sprintf("writing location for %v", identifier))
			builder.WriteString(fmt.Sprintf("LOCATION:%s\r\n", *event.Location))
		}

		if event.Organization != "" {
			l.Debug(fmt.Sprintf("writing organization for %v", identifier))
			builder.WriteString(fmt.Sprintf("X-ORGANIZING-GROUP:%s\r\n", event.Organization))
			builder.WriteString(fmt.Sprintf("X-TEAMUP-WHO:%s\r\n", event.Organization))
		}

		l.Debug(fmt.Sprintf("writing custom properties for %v", identifier))
		builder.WriteString(fmt.Sprintf("X-EVENT-TYPE:%s\r\n", event.Type))
		builder.WriteString(fmt.Sprintf("X-REJECTED:%t\r\n", event.Rejected))

		// Add sequence if greater than 0
		if event.Sequence > 0 {
			l.Debug(fmt.Sprintf("writing sequence for %v", identifier))
			builder.WriteString(fmt.Sprintf("SEQUENCE:%d\r\n", event.Sequence))
		}

		// Add recurrence fields if present
		if event.RecurrenceID != nil && *event.RecurrenceID != "" {
			l.Debug(fmt.Sprintf("writing recurrence ID for %v", identifier))
			builder.WriteString(fmt.Sprintf("RECURRENCE-ID:%s\r\n", *event.RecurrenceID))
		}

		if event.RRule != nil && *event.RRule != "" {
			l.Debug(fmt.Sprintf("writing rrule for %v", identifier))
			builder.WriteString(fmt.Sprintf("RRULE:%s\r\n", *event.RRule))
		}

		if event.RDate != nil && *event.RDate != "" {
			l.Debug(fmt.Sprintf("writing rdate for %v", identifier))
			builder.WriteString(fmt.Sprintf("RDATE:%s\r\n", *event.RDate))
		}

		l.Debug(fmt.Sprintf("compiling exdate info for %v", identifier))
		exdates := []string{}
		if event.ExDate != nil && *event.ExDate != "" {
			l.Debug(fmt.Sprintf("synced exdates found for %v", identifier))
			exdates = append(exdates, strings.Split(*event.ExDate, ",")...)
		}

		if event.ExDateManual != nil && *event.ExDateManual != "" {
			l.Debug(fmt.Sprintf("manual exdates found for %v", identifier))
			exdates = append(exdates, strings.Split(*event.ExDateManual, ",")...)
		}

		if len(exdates) > 0 {
			l.Debug(fmt.Sprintf("combining exdates for %v", identifier))
			builder.WriteString(fmt.Sprintf("EXDATE:%s\r\n", strings.Join(exdates, ",")))
		}

		// Add created and modified times if available
		if event.Created != nil {
			l.Debug(fmt.Sprintf("getting created date for %v", identifier))
			builder.WriteString(fmt.Sprintf("CREATED:%s\r\n", event.Created.UTC().Format("20060102T150405Z")))
		}
		if event.Modified != nil {
			l.Debug(fmt.Sprintf("getting modified date for %v", identifier))
			builder.WriteString(fmt.Sprintf("LAST-MODIFIED:%s\r\n", event.Modified.UTC().Format("20060102T150405Z")))
		}

		l.Debug(fmt.Sprintf("ending writing event for %v", identifier))
		builder.WriteString("END:VEVENT\r\n")
	}

	// Write iCal footer
	l.Debug("completing ical write")
	builder.WriteString("END:VCALENDAR\r\n")

	return builder.String(), nil
}

func (s *Server) updateRootExdate(gi *event.GetEventInput, rejected bool) error {
	rootGi := &event.GetEventInput{UID: gi.UID}
	rootEvt, err := s.db.Events.GetEvent(rootGi)
	if err != nil {
		return fmt.Errorf("failed to find root event: %v", err)
	}

	loc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return err
	}

	affectedDateStr := rootEvt.StartTime.In(loc).Format("20060102T150405")
	if gi.RecurrenceID != nil && *gi.RecurrenceID != "" {
		affectedDateStr = *gi.RecurrenceID
	}

	if rejected {
		if rootEvt.ExDateManual == nil || *rootEvt.ExDateManual == "" {
			rootEvt.ExDateManual = &affectedDateStr
		} else {
			*rootEvt.ExDateManual += fmt.Sprintf(",%v", affectedDateStr)
		}
	} else {
		exdates := strings.Split(*rootEvt.ExDateManual, ",")
		newExdates := []string{}
		for _, exdate := range exdates {
			if exdate != affectedDateStr {
				newExdates = append(newExdates, exdate)
			}
		}

		*rootEvt.ExDateManual = strings.Join(newExdates, ",")
	}

	rootPi := &event.PatchEventInput{ExDateManual: rootEvt.ExDateManual}
	err = s.db.Events.PatchEvent(rootGi, rootPi)
	if err != nil {
		return fmt.Errorf("failed to patch root event: %v", err)
	}

	return nil
}

func (s *Server) updateEventType(gi *event.GetEventInput, eventType string) error {
	evts, err := s.db.Events.GetEvents(&event.GetEventsInput{UID: &gi.UID})
	if err != nil {
		return fmt.Errorf("could not get sibling events: %v", err)
	}

	for _, evt := range evts {
		evtGi := &event.GetEventInput{UID: evt.UID}
		if evt.RecurrenceID != nil && *evt.RecurrenceID != "" {
			evtGi.RecurrenceID = evt.RecurrenceID
		}

		err = s.db.Events.PatchEvent(evtGi, &event.PatchEventInput{Type: &eventType})
		if err != nil {
			return fmt.Errorf("could not update sibling events: %v", err)
		}
	}

	return nil
}