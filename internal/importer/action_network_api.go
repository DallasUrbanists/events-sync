package importer

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
	_ "time/tzdata"

	"github.com/dallasurbanists/events-sync/pkg/event"
)

type ActionNetworkAPIResponse struct {
	TotalPages   int `json:"total_pages"`
	PerPage      int `json:"per_page"`
	Page         int `json:"page"`
	TotalRecords int `json:"total_records"`
	Embedded     struct {
		Events []ActionNetworkEvent `json:"osdi:events"`
	} `json:"_embedded"`
	Links struct {
		Next struct {
			Href string `json:"href"`
		} `json:"next"`
	} `json:"_links"`
}

type ActionNetworkEvent struct {
	Identifiers []string  `json:"identifiers"`
	CreatedDate time.Time `json:"created_date"`
	Description string    `json:"description"`
	StartDate   time.Time `json:"start_date"`
	EndDate     time.Time `json:"end_date,omitempty"`
	Title       string    `json:"title"`
	Name        string    `json:"name"`
	Status      string    `json:"status"`
	Location    struct {
		Venue        string   `json:"venue"`
		AddressLines []string `json:"address_lines"`
		Locality     string   `json:"locality"`
		Region       string   `json:"region"`
		PostalCode   string   `json:"postal_code"`
		Country      string   `json:"country"`
	} `json:"location"`
	BrowserURL string `json:"browser_url"`
}

func action_network_api_importer(url string, organization string, options map[string]string) ([]*event.Event, error) {
	baseURL := "https://actionnetwork.org/api/v2/events"
	if url != "" {
		baseURL = url // honestly just keeping this here to match the other importers and for future testing/proofing
	}

	apiKey, ok := options["api_key"]
	if !ok {
		return nil, fmt.Errorf("Action Network api_key not found in options for organization %s", organization)
	}

	events, err := fetchActionNetworkEvents(apiKey, baseURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch events from Action Network API: %v", err)
	}

	var convertedEvents []*event.Event
	for _, anEvent := range events {
		e, err := convertActionNetworkEventToEvent(anEvent, organization)
		if err != nil {
			return nil, fmt.Errorf("failed to convert event %s: %v", anEvent.Title, err)
		}
		convertedEvents = append(convertedEvents, &e)
	}

	return convertedEvents, nil
}

func fetchActionNetworkEvents(apiKey string, baseURL string) ([]ActionNetworkEvent, error) {
	var allEvents []ActionNetworkEvent
	page := 1

	for {
		url := baseURL
		if page > 1 {
			url = fmt.Sprintf("%s?page=%d", baseURL, page)
		}

		events, hasNext, err := fetchActionNetworkPage(url, apiKey)
		if err != nil {
			return nil, err
		}

		allEvents = append(allEvents, events...)

		if !hasNext {
			break
		}
		page++
	}

	return allEvents, nil
}

func fetchActionNetworkPage(url, apiKey string) ([]ActionNetworkEvent, bool, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, false, fmt.Errorf("failed to create request: %v", err)
	}

	req.Header.Set("OSDI-API-Token", apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, false, fmt.Errorf("failed to make request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return nil, false, fmt.Errorf("API request failed with status %d: %s", resp.StatusCode, string(body))
	}
	if err != nil {
		return nil, false, fmt.Errorf("failed to read response body: %v", err)
	}

	var apiResponse ActionNetworkAPIResponse
	if err := json.Unmarshal(body, &apiResponse); err != nil {
		return nil, false, fmt.Errorf("failed to parse JSON response: %v", err)
	}

	hasNext := apiResponse.Links.Next.Href != ""

	return apiResponse.Embedded.Events, hasNext, nil
}

func fixTimezone(t time.Time) (time.Time, error) {
	defaultLoc, err := time.LoadLocation("America/Chicago")
	if err != nil {
		return time.Now(), err
	}

	return time.Date(t.Year(), t.Month(), t.Day(), t.Hour(), t.Minute(), t.Second(), t.Nanosecond(), defaultLoc), nil
}

func convertActionNetworkEventToEvent(anEvent ActionNetworkEvent, organization string) (event.Event, error) {
	startTime, err := fixTimezone(anEvent.StartDate)
	if err != nil {
		return event.Event{}, err
	}

	endTime := startTime.Add(1 * time.Hour)
	if !anEvent.EndDate.IsZero() {
		endTime, err = fixTimezone(anEvent.EndDate)
		if err != nil {
			return event.Event{}, err
		}
	}

	createdTime := anEvent.CreatedDate

	var locationParts []string
	if anEvent.Location.Venue != "" {
		locationParts = append(locationParts, anEvent.Location.Venue)
	}
	if len(anEvent.Location.AddressLines) > 0 {
		locationParts = append(locationParts, strings.Join(anEvent.Location.AddressLines, ", "))
	}
	if anEvent.Location.Locality != "" {
		locationParts = append(locationParts, anEvent.Location.Locality)
	}
	if anEvent.Location.Region != "" {
		locationParts = append(locationParts, anEvent.Location.Region)
	}
	if anEvent.Location.PostalCode != "" {
		locationParts = append(locationParts, anEvent.Location.PostalCode)
	}
	location := strings.Join(locationParts, ", ")

	var uid string
	if len(anEvent.Identifiers) > 0 {
		for _, id := range anEvent.Identifiers {
			parts := strings.Split(id, ":")
			if len(parts) > 1 && parts[0] == "action_network" {
				uid = id
				break
			}
		}
	}

	description := fmt.Sprintf("Register for this event from %s on Action Network: %s", organization, anEvent.BrowserURL)
	escapedDescription := escape(description)
	status := strings.ToUpper(anEvent.Status)
	transparency := "OPAQUE"

	e := event.Event{
		Organization: organization,
		UID:          uid,
		Summary:      anEvent.Title,
		Description:  &escapedDescription,
		Location:     &location,
		StartTime:    startTime,
		EndTime:      endTime,
		Created:      &createdTime,
		Modified:     &createdTime,
		Status:       &status,
		Transparency: &transparency,
	}

	return e, nil
}
