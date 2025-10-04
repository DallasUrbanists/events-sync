package database

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/dallasurbanists/events-sync/pkg/event"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
)

type EventRepository struct {
	*sqlx.DB
}

// Event represents an event in the database
type Event struct {
	ID        int       `db:"id"`
	CreatedAt time.Time `db:"created_at"`
	UpdatedAt time.Time `db:"updated_at"`

	UID          string     `db:"uid"`
	Organization string     `db:"organization"`
	Summary      string     `db:"summary"`
	Description  *string    `db:"description"`
	Location     *string    `db:"location"`
	StartTime    time.Time  `db:"start_time"`
	EndTime      time.Time  `db:"end_time"`
	CreatedTime  *time.Time `db:"created_time"`
	ModifiedTime *time.Time `db:"modified_time"`
	Status       *string    `db:"status"`
	Transparency *string    `db:"transparency"`
	Rejected     bool       `db:"rejected"`
	Sequence     int        `db:"sequence"`
	RecurrenceID *string    `db:"recurrence_id"`
	RRule        *string    `db:"rrule"`
	RDate        *string    `db:"rdate"`
	ExDate       *string    `db:"exdate"`
	ExDateManual *string    `db:"exdate_manual"`
	Type         string     `db:"type"`
	Overlay      *string    `db:"overlay"`
}

func marshal(d *Event) *event.Event {
	e := event.Event{
		UID:          d.UID,
		Organization: d.Organization,
		Summary:      d.Summary,
		Description:  d.Description,
		Location:     d.Location,
		StartTime:    d.StartTime,
		EndTime:      d.EndTime,
		Created:      d.CreatedTime,
		Modified:     d.ModifiedTime,
		Rejected:     d.Rejected,
		Status:       d.Status,
		Transparency: d.Transparency,
		Sequence:     d.Sequence,
		RecurrenceID: d.RecurrenceID,
		RRule:        d.RRule,
		RDate:        d.RDate,
		ExDate:       d.ExDate,
		ExDateManual: d.ExDateManual,
		Type:         d.Type,
	}

	// Handle overlay JSON conversion
	if d.Overlay != nil && *d.Overlay != "" {
		var overlay map[string]event.EventOverlay
		if err := json.Unmarshal([]byte(*d.Overlay), &overlay); err == nil {
			e.Overlay = overlay
		}
	}

	return &e
}

func unmarshal(e *event.Event) *Event {
	d := Event{
		UID:          e.UID,
		Organization: e.Organization,
		Summary:      e.Summary,
		Description:  e.Description,
		Location:     e.Location,
		StartTime:    e.StartTime,
		EndTime:      e.EndTime,
		CreatedTime:  e.Created,
		ModifiedTime: e.Modified,
		Status:       e.Status,
		Transparency: e.Transparency,
		Sequence:     e.Sequence,
		RecurrenceID: e.RecurrenceID,
		RRule:        e.RRule,
		RDate:        e.RDate,
		ExDate:       e.ExDate,
		ExDateManual: e.ExDateManual,
		Rejected:     e.Rejected,
		Type:         e.Type,
	}

	empty := ""
	if d.RecurrenceID == nil {
		d.RecurrenceID = &empty
	}

	// Handle overlay JSON conversion
	if len(e.Overlay) > 0 {
		overlayJSON, err := json.Marshal(e.Overlay)
		if err == nil {
			overlayStr := string(overlayJSON)
			d.Overlay = &overlayStr
		}
	}

	return &d
}

const insertEventQuery = `
  INSERT INTO events (
    uid, organization,
    summary, description,
    location, start_time, end_time,
    created_time, modified_time,
    status, transparency, sequence,
    recurrence_id, rrule, rdate, exdate, exdate_manual,
    rejected, type, overlay
  ) VALUES (
    :uid, :organization,
    :summary, :description,
    :location, :start_time, :end_time,
    :created_time, :modified_time,
    :status, :transparency, :sequence,
    :recurrence_id, :rrule, :rdate, :exdate, :exdate_manual,
    :rejected, :type, :overlay
  )
`

func (db *EventRepository) InsertEvent(e *event.Event) error {
	d := unmarshal(e)
	rows, err := db.NamedQuery(insertEventQuery, d)
	if err != nil {
		return fmt.Errorf("failed to insert event: %v", err)
	}
	defer rows.Close()

	return nil
}

func (db *EventRepository) GetEvent(i *event.GetEventInput) (*event.Event, error) {
	getEventQuery := fmt.Sprintf(`
		SELECT %v FROM events
		WHERE
			uid = $1 AND
			recurrence_id = $2
	`, DBColumns[Event]())

	existing := &Event{}

	empty := ""
	recurrenceID := empty
	if i.RecurrenceID != nil {
		recurrenceID = *i.RecurrenceID
	}

	err := db.Get(existing, getEventQuery, i.UID, recurrenceID)
	if err == sql.ErrNoRows {
		return nil, event.NewNoEventsError(err)
	} else if err != nil {
		return nil, err
	}

	return marshal(existing), nil
}

func (db *EventRepository) GetEvents(i *event.GetEventsInput) ([]*event.Event, error) {
	getEventQuery := fmt.Sprintf("SELECT %v FROM events ", DBColumns[Event]())
	idx := 0
	args := []interface{}{}

	var dbEvents []*Event

	if i != nil {
		filterPrefix := "WHERE"

		if i.UID != nil {
			idx++
			getEventQuery += fmt.Sprintf("%v uid = $%d ", filterPrefix, idx)
			args = append(args, i.UID)
			filterPrefix = "AND"
		}

		if i.Rejected != nil {
			idx++
			getEventQuery += fmt.Sprintf("%v rejected = $%d ", filterPrefix, idx)
			args = append(args, i.Rejected)
			filterPrefix = "AND"
		}

		if i.Organization != nil {
			idx++
			getEventQuery += fmt.Sprintf("%v organization = $%d ", filterPrefix, idx)
			args = append(args, i.Organization)
			filterPrefix = "AND"
		}

		if i.Type != nil {
			idx++
			getEventQuery += fmt.Sprintf("%v type = $%d ", filterPrefix, idx)
			args = append(args, i.Type)
			filterPrefix = "AND"
		}

		if i.UpcomingOnly {
			getEventQuery += fmt.Sprintf("%v start_time > NOW() ", filterPrefix)
			filterPrefix = "AND"
		}
	}

	getEventQuery += "ORDER BY start_time"

	err := db.Select(&dbEvents, getEventQuery, args...)
	if err == sql.ErrNoRows {
		return nil, event.NewNoEventsError(err)
	} else if err != nil {
		return nil, fmt.Errorf("failed to get events: %v", err)
	}

	events := []*event.Event{}
	for _, event := range dbEvents {
		events = append(events, marshal(event))
	}

	return events, nil
}

func (db *EventRepository) PatchEvent(gi *event.GetEventInput, pi *event.PatchEventInput) error {
	updateQuery := "UPDATE events SET "
	args := []interface{}{}

	if pi == nil {
		return errors.New("failed to patch event, no patch input given")
	}

	if pi.Organization != nil {
		args = append(args, *pi.Organization)
		updateQuery += fmt.Sprintf("organization = $%d ", len(args))
	}

	if pi.Rejected != nil {
		args = append(args, *pi.Rejected)
		updateQuery += fmt.Sprintf("rejected = $%d ", len(args))
	}

	if pi.Type != nil {
		args = append(args, *pi.Type)
		updateQuery += fmt.Sprintf("type = $%d ", len(args))
	}

	if pi.ExDateManual != nil {
		args = append(args, *pi.ExDateManual)
		updateQuery += fmt.Sprintf("exdate_manual = $%d ", len(args))
	}

	if pi.Overlay != nil {
		overlayJSON, err := json.Marshal(pi.Overlay)
		if err != nil {
			return fmt.Errorf("failed to marshal overlay: %v", err)
		}
		args = append(args, string(overlayJSON))
		updateQuery += fmt.Sprintf("overlay = $%d ", len(args))
	}

	args = append(args, gi.UID)
	updateQuery += fmt.Sprintf("WHERE uid = $%d ", len(args))

	if gi.RecurrenceID != nil {
		args = append(args, *gi.RecurrenceID)
	} else {
		args = append(args, "")
	}
	updateQuery += fmt.Sprintf("AND recurrence_id = $%d ", len(args))

	_, err := db.Exec(updateQuery, args...)
	return err
}

func (db *EventRepository) SyncEvent(gi *event.GetEventInput, si *event.SyncEventInput) error {
	updateQuery := "UPDATE events SET "
	args := []interface{}{}

	updatePrefix := ""

	if si.Summary != nil {
		args = append(args, si.Summary)
		updateQuery += fmt.Sprintf("%v summary = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.Description != nil {
		args = append(args, si.Description)
		updateQuery += fmt.Sprintf("%v description = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.Location != nil {
		args = append(args, si.Location)
		updateQuery += fmt.Sprintf("%v location = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.StartTime != nil {
		args = append(args, si.StartTime)
		updateQuery += fmt.Sprintf("%v start_time = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.EndTime != nil {
		args = append(args, si.EndTime)
		updateQuery += fmt.Sprintf("%v end_time = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.Rejected != nil {
		args = append(args, si.Rejected)
		updateQuery += fmt.Sprintf("%v rejected = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.Status != nil {
		args = append(args, si.Status)
		updateQuery += fmt.Sprintf("%v status = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.Transparency != nil {
		args = append(args, si.Transparency)
		updateQuery += fmt.Sprintf("%v transparency = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.Sequence != nil {
		args = append(args, si.Sequence)
		updateQuery += fmt.Sprintf("%v sequence = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.RRule != nil {
		args = append(args, si.RRule)
		updateQuery += fmt.Sprintf("%v rrule = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.RDate != nil {
		args = append(args, si.RDate)
		updateQuery += fmt.Sprintf("%v rdate = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	if si.ExDate != nil {
		args = append(args, si.ExDate)
		updateQuery += fmt.Sprintf("%v exdate = $%d ", updatePrefix, len(args))
		updatePrefix = ","
	}

	args = append(args, gi.UID)
	updateQuery += fmt.Sprintf("WHERE uid = $%d ", len(args))

	if gi.RecurrenceID != nil {
		args = append(args, gi.RecurrenceID)
	} else {
		args = append(args, "")
	}
	updateQuery += fmt.Sprintf("AND recurrence_id = $%d ", len(args))

	_, err := db.Exec(updateQuery, args...)
	return err
}

func (db *EventRepository) PruneOrganizationEvents(pi *event.PruneOrganizationEventsInput) error {
	organization := pi.Organization
	sourceEvents := pi.ExistingEvents

	sourceEventMap := make(map[string]bool)
	for _, e := range sourceEvents {
		key := e.UID
		if e.RecurrenceID != nil && *e.RecurrenceID != "" {
			key = e.UID + ":" + *e.RecurrenceID
		}
		sourceEventMap[key] = true
	}

	events, err := db.GetEvents(&event.GetEventsInput{Organization: &organization})
	if err != nil {
		return fmt.Errorf("failed to get events for organization %s: %v", organization, err)
	}

	var eventsToDelete []*event.Event
	for _, e := range events {
		key := e.UID
		if e.RecurrenceID != nil && *e.RecurrenceID != "" {
			key = e.UID + ":" + *e.RecurrenceID
		}

		if !sourceEventMap[key] {
			eventsToDelete = append(eventsToDelete, e)
		}
	}

	if len(eventsToDelete) > 0 {
		fmt.Printf("Deleting %d events for organization %s that are no longer in source calendar:\n", len(eventsToDelete), organization)

		// Convert to DB event object and
		// log summaries of events being deleted
		dbEventsToDelete := []*Event{}
		for i, e := range eventsToDelete {
			dbEvent := unmarshal(e)
			dbEventsToDelete = append(dbEventsToDelete, dbEvent)

			summary := dbEvent.Summary
			if len(summary) > 50 {
				summary = summary[:47] + "..."
			}
			fmt.Printf("  %d. %s (UID: %s)\n", i+1, summary, dbEvent.UID)
		}

		for _, dbEvent := range dbEventsToDelete {
			deleteQuery := "DELETE FROM events WHERE uid = $1 AND organization = $2 "
			args := []interface{}{dbEvent.UID, dbEvent.Organization}

			if dbEvent.RecurrenceID != nil {
				args = append(args, dbEvent.RecurrenceID)
			} else {
				args = append(args, "")
			}
			deleteQuery += fmt.Sprintf("AND recurrence_id = $%d ", len(args))

			_, err := db.Exec(deleteQuery, args...)
			if err != nil {
				fmt.Printf("Error deleting event (%s): %v\n", deleteQuery, err)
			}
		}
	}

	return nil
}
