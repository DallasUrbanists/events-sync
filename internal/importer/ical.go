package importer

import (
	"fmt"
	"io"
	"net/http"

	"github.com/dallasurbanists/events-sync/pkg/event"
)

func ical_importer(url string, organization string, options map[string]string) ([]event.Event, error) {
	fmt.Printf("Fetching ICS file from: %s\n", url)

	content, err := fetchICS(url)
	if err != nil {
		return nil, fmt.Errorf("error fetching ICS: %v", err)
	}

	events, err := event.ParseICS(content, organization)
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
