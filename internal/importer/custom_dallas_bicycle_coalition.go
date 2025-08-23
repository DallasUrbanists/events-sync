package importer

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/dallasurbanists/events-sync/pkg/event"
)

type DBCEventDate struct {
	Type      string    `json:"_type"`
	StartDate time.Time `json:"startDate"`
	EndDate   time.Time `json:"endDate"`
}

type DBCDescriptionChild struct {
	Text    string `json:"text"`
	Ignored string `json:"-"`
}

type DBCDescription struct {
	Children []DBCDescriptionChild `json:"children"`
	Ignored  string                `json:"-"`
}

type DBCEvent struct {
	ID        string    `json:"_id"`
	CreatedAt time.Time `json:"_createdAt"`

	Title       string           `json:"title"`
	Excerpt     string           `json:"excerpt"`
	Description []DBCDescription `json:"description"`

	AllDay   bool         `json:"allDay"`
	Date     DBCEventDate `json:"date"`
	Location string       `json:"location"`

	Ignored string `json:"-"`
}

type SanityResponse struct {
	Result []DBCEvent `json:"result"`
}

func custom_dallas_bicycle_coalition(url string, organization string, options map[string]string) ([]event.Event, error) {
	b, err := fetch(url)
	if err != nil {
		return nil, err
	}

	var sanityResp SanityResponse
	if err := json.Unmarshal(b, &sanityResp); err != nil {
		return nil, fmt.Errorf("failed to parse JSON: %v", err)
	}

	converted := []event.Event{}
	for _, e := range sanityResp.Result {
		converted = append(converted, convertToEvent(e, organization))
	}

	return converted, nil
}

func convertToEvent(i DBCEvent, organization string) event.Event {
	o := event.Event{
		Organization: organization,
		UID:          fmt.Sprintf("dbc_%v", i.ID),
		Summary:      i.Title,
		Location:     i.Location,
		StartTime:    i.Date.StartDate,
		Created:      i.CreatedAt,
	}

	o.EndTime = i.Date.EndDate
	if o.EndTime.IsZero() {
		o.EndTime = i.Date.StartDate.Add(1 * time.Hour)
	}

	if strings.TrimSpace(i.Excerpt) != "" {
		o.Description = fmt.Sprintf("%v\\n", escape(i.Excerpt))
	}

	for _, d := range i.Description {
		for _, c := range d.Children {
			o.Description += fmt.Sprintf("%v\\n", escape(c.Text))
		}
	}

	return o
}
