package event

import (
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
	Type         string    `json:"type"`
}

// EventType constants for convenience
const (
	EventTypeCivicMeeting   = "civic_meeting"
	EventTypeSocialGathering = "social_gathering"
	EventTypeVolunteerAction = "volunteer_action"
)

// EventTypes maps event type keys to their display names
var EventTypes = map[string]string{
	EventTypeCivicMeeting:   "Civic Meeting",
	EventTypeSocialGathering: "Social Gathering",
	EventTypeVolunteerAction: "Volunteer Action",
}

// IsValidEventType checks if the given event type is valid
func IsValidEventType(eventType string) bool {
	_, exists := EventTypes[eventType]
	return exists
}
