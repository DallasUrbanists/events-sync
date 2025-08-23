package importer

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/dallasurbanists/events-sync/pkg/event"
)

type Importers map[string]Importer

type Importer func(string, string, map[string]string) ([]event.Event, error)

func RegisterImporters() Importers {
	i := Importers{}
	i["custom_dallas_bicycle_coalition"] = custom_dallas_bicycle_coalition
	i["action_network_api"] = action_network_api_importer
	i["ical"] = ical_importer

	return i
}

func fetch(url string) ([]byte, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch data: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("request failed with status: %s", resp.Status)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %v", err)
	}

	return body, nil
}

func escape(i string) string {
	o := i

	o = strings.ReplaceAll(o, "\\", "\\\\")

	o = strings.ReplaceAll(o, ";", "\\;")

	o = strings.ReplaceAll(o, ",", "\\,")

	o = strings.ReplaceAll(o, "\n", "\\n")
	o = strings.ReplaceAll(o, "\r", "\\r")

	return o
}
