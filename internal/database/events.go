package database

import (
	"database/sql"
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
	ID           int        `db:"id"`
  CreatedAt    time.Time  `db:"created_at"`
  UpdatedAt    time.Time  `db:"updated_at"`

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
}

func marshal(d *Event) *event.Event{
	e := event.Event{
    UID: d.UID,
    Organization: d.Organization,
    Summary: d.Summary,
    Description: d.Description,
    Location: d.Location,
    StartTime: d.StartTime,
    EndTime: d.EndTime,
    Created: d.CreatedTime,
    Modified: d.ModifiedTime,
    Rejected: d.Rejected,
    Status: d.Status,
    Transparency: d.Transparency,
    Sequence: d.Sequence,
    RecurrenceID: d.RecurrenceID,
    RRule: d.RRule,
    RDate: d.RDate,
    ExDate: d.ExDate,
	}

	return &e
}

func unmarshal(e *event.Event) *Event{
	d := Event{
  	UID: e.UID,
  	Organization: e.Organization,
  	Summary: e.Summary,
  	Description: e.Description,
  	Location: e.Location,
  	StartTime: e.StartTime,
  	EndTime: e.EndTime,
  	CreatedTime: e.Created,
  	ModifiedTime: e.Modified,
  	Status: e.Status,
  	Transparency: e.Transparency,
  	Sequence: e.Sequence,
  	RecurrenceID: e.RecurrenceID,
  	RRule: e.RRule,
  	RDate: e.RDate,
  	ExDate: e.ExDate,
  	Rejected: e.Rejected,
	}

	empty := ""
	if d.RecurrenceID == nil {
		d.RecurrenceID = &empty
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
    recurrence_id, rrule, rdate, exdate,
    rejected
  ) VALUES (
    :uid, :organization,
    :summary, :description,
    :location, :start_time, :end_time,
    :created_time, :modified_time,
    :status, :transparency, :sequence,
    :recurrence_id, :rrule, :rdate, :exdate,
    :rejected
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

func (db *EventRepository) GetEvent(i *event.GetEventInput) (*event.Event, error){
	getEventQuery := fmt.Sprintf(`
		SELECT %v FROM events
		WHERE
			uid = $1 AND
			recurrence_id = $2
	`, DBColumns[Event]())

	existing := &Event{}

	empty := ""
	recurrenceID := &empty
	if i.RecurrenceID != nil {
		recurrenceID = i.RecurrenceID
	}

	err := db.Get(existing, getEventQuery, i.UID, *recurrenceID)
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
  for _, event := range(dbEvents) {
    events = append(events, marshal(event))
  }

  return events, nil
}

func (db *EventRepository) PatchEvent(gi *event.GetEventInput, pi *event.PatchEventInput) error {
	updateQuery := "UPDATE events SET "
	idx := 0
	args := []interface{}{}

	if pi == nil {
		return errors.New("failed to patch event, no patch input given")
	}

	if pi.Organization != nil {
		idx++
		updateQuery += fmt.Sprintf("organization = $%d ", idx)
		args = append(args, *pi.Organization)
	}

	if pi.Rejected != nil {
		idx++
		updateQuery += fmt.Sprintf("rejected = $%d ", idx)
		args = append(args, *pi.Rejected)
	}

	idx++
	updateQuery += fmt.Sprintf("WHERE uid = $%d ", idx)
	args = append(args, gi.UID)

	if gi.RecurrenceID != nil {
		idx++
		updateQuery += fmt.Sprintf("AND recurrence_id = $%d ", idx)
		args = append(args, *gi.RecurrenceID)
	}

	_, err := db.Exec(updateQuery, args...)
  return err
}

func (db *EventRepository) SyncEvent(gi *event.GetEventInput, si *event.SyncEventInput) error {
	updateQuery := "UPDATE events SET "
	idx := 0
	args := []interface{}{}

	updatePrefix := ""

	if si.Summary != nil {
		idx++
		updateQuery += fmt.Sprintf("%v summary = $%d ", updatePrefix, idx)
		args = append(args, si.Summary)
		updatePrefix = ","
	}

	if si.Description != nil {
		idx++
		updateQuery += fmt.Sprintf("%v description = $%d ", updatePrefix, idx)
		args = append(args, si.Description)
		updatePrefix = ","
	}

	if si.Location != nil {
		idx++
		updateQuery += fmt.Sprintf("%v location = $%d ", updatePrefix, idx)
		args = append(args, si.Location)
		updatePrefix = ","
	}

	if si.StartTime != nil {
		idx++
		updateQuery += fmt.Sprintf("%v start_time = $%d ", updatePrefix, idx)
		args = append(args, si.StartTime)
		updatePrefix = ","
	}

	if si.EndTime != nil {
		idx++
		updateQuery += fmt.Sprintf("%v end_time = $%d ", updatePrefix, idx)
		args = append(args, si.EndTime)
		updatePrefix = ","
	}

	if si.Rejected != nil {
		idx++
		updateQuery += fmt.Sprintf("%v rejected = $%d ", updatePrefix, idx)
		args = append(args, si.Rejected)
		updatePrefix = ","
	}

	if si.Status != nil {
		idx++
		updateQuery += fmt.Sprintf("%v status = $%d ", updatePrefix, idx)
		args = append(args, si.Status)
		updatePrefix = ","
	}

	if si.Transparency != nil {
		idx++
		updateQuery += fmt.Sprintf("%v transparency = $%d ", updatePrefix, idx)
		args = append(args, si.Transparency)
		updatePrefix = ","
	}

	if si.Sequence != nil {
		idx++
		updateQuery += fmt.Sprintf("%v sequence = $%d ", updatePrefix, idx)
		args = append(args, si.Sequence)
		updatePrefix = ","
	}

	if si.RRule != nil {
		idx++
		updateQuery += fmt.Sprintf("%v rrule = $%d ", updatePrefix, idx)
		args = append(args, si.RRule)
		updatePrefix = ","
	}

	if si.RDate != nil {
		idx++
		updateQuery += fmt.Sprintf("%v rdate = $%d ", updatePrefix, idx)
		args = append(args, si.RDate)
		updatePrefix = ","
	}

	if si.ExDate != nil {
		idx++
		updateQuery += fmt.Sprintf("%v exdate = $%d ", updatePrefix, idx)
		args = append(args, si.ExDate)
		updatePrefix = ","
	}

	idx++
	updateQuery += fmt.Sprintf("WHERE uid = $%d ", idx)
	args = append(args, gi.UID)

	if gi.RecurrenceID != nil {
		idx++
		updateQuery += fmt.Sprintf("AND recurrence_id = $%d ", idx)
		args = append(args, gi.RecurrenceID)
	}

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

      if dbEvent.RecurrenceID != nil && *dbEvent.RecurrenceID != "" {
        deleteQuery += fmt.Sprintf("AND recurrence_id = $%d ", len(args) + 1)
        args = append(args, dbEvent.RecurrenceID)
      }

      _, err := db.Exec(deleteQuery, args...)
      if err != nil {
        fmt.Printf("Error deleting event (%s): %v\n", deleteQuery, err)
      }
    }
  }

  return nil
}
