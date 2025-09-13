package event

import (
	"fmt"
	"time"
	_ "time/tzdata"
)

// EventType constants
const (
	EventTypeCivicMeeting   = "civic_meeting"
	EventTypeSocialGathering = "social_gathering"
	EventTypeVolunteerAction = "volunteer_action"
)

// EventTypeDisplayName maps event type keys to their display names
var EventTypeDisplayName = map[string]string{
	EventTypeCivicMeeting:   "Civic Meeting",
	EventTypeSocialGathering: "Social Gathering",
	EventTypeVolunteerAction: "Volunteer Action",
}

type Event struct {
	UID          string     `json:"uid"`
	Organization string     `json:"organization"`
	Summary      string     `json:"summary"`
	Description  *string    `json:"description"`
	Location     *string    `json:"location"`
	StartTime    time.Time  `json:"start_time"`
	EndTime      time.Time  `json:"end_time"`
	Created      *time.Time `json:"created"`
	Modified     *time.Time `json:"modified"`
	Rejected     bool       `json:"rejected"`
	Status       *string    `json:"status"`
	Transparency *string    `json:"transparency"`
	Sequence     int        `json:"sequence"`
	RecurrenceID *string    `json:"recurrence_id"`
	RRule        *string    `json:"rrule"`
	RDate        *string    `json:"rdate"`
	ExDate       *string    `json:"exdate"`
	Type         string     `json:"type"`
}

type GetEventInput struct {
	UID          string
	RecurrenceID *string
}

type GetEventsInput struct {
	Rejected     *bool
	Organization *string
	UpcomingOnly bool
	Type         *string
}

type NoEventsError struct {
	original error
}

func (e NoEventsError) Error() string {
	return fmt.Sprintf("no events found, original error: %v", e.original.Error())
}

func NewNoEventsError(o error) NoEventsError { return NoEventsError{o} }

type PatchEventInput struct {
	Organization *string
	Rejected     *bool
	Type         *string
}

type SyncEventInput struct {
	Summary      *string
	Description  *string
	Location     *string
	StartTime    *time.Time
	EndTime      *time.Time
	Rejected     *bool
	Status       *string
	Transparency *string
	Sequence     *int
	RRule        *string
	RDate        *string
	ExDate       *string
}

type PruneOrganizationEventsInput struct {
	Organization   string
	ExistingEvents []GetEventInput
}

type Repository interface {
	InsertEvent(*Event) error
	GetEvent(*GetEventInput) (*Event, error)
	GetEvents(*GetEventsInput) ([]*Event, error)
	PatchEvent(*GetEventInput, *PatchEventInput) error
	SyncEvent(*GetEventInput, *SyncEventInput) error

	PruneOrganizationEvents(*PruneOrganizationEventsInput) error
}
