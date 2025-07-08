# Events Sync

A Go application that fetches and parses ICS (iCalendar) files from multiple organization calendars.

## Features

- **Multi-organization support** via configuration file
- Fetches ICS files from Google Calendar and Meetup.com URLs
- Parses ICS format into structured Go data
- Extracts event details including:
  - Organization name, UID, Summary, Description, Location
  - Start and End times
  - Created and Modified timestamps
  - Status and Transparency settings
- Outputs events as formatted JSON
- Robust error handling for individual organization failures

## Configuration

Create a `config.json` file with your organization mappings:

```json
{
  "organizations": {
    "DATA": "https://calendar.google.com/calendar/ical/c_b84185fb0c5798bfc8d926ac5013d4ed1fdbd0c3fb79a960686fbb9250037595%40group.calendar.google.com/public/basic.ics",
    "Dallas Urbanists STLC": "https://www.meetup.com/dallasurbanists/events/ical/",
    "Dallas Bicycle Coalition": "https://calendar.google.com/calendar/ical/dallasbicyclecoalition.org_abc123@group.calendar.google.com/public/basic.ics"
  }
}
```

## Usage

```bash
go run main.go
```

The program will:
1. Load the configuration from `config.json`
2. Fetch ICS files from each organization's URL
3. Parse the ICS content into Event structs
4. Combine all events from all organizations
5. Output the complete event list as formatted JSON

## Event Structure

```go
type Event struct {
    Organization string    `json:"organization"`
    UID         string    `json:"uid"`
    Summary     string    `json:"summary"`
    Description string    `json:"description"`
    Location    string    `json:"location"`
    StartTime   time.Time `json:"start_time"`
    EndTime     time.Time `json:"end_time"`
    Created     time.Time `json:"created"`
    Modified    time.Time `json:"modified"`
    Status      string    `json:"status"`
    Transparency string   `json:"transparency"`
}
```

## Supported Calendar Sources

- **Google Calendar**: Public ICS URLs
- **Meetup.com**: Public ICS URLs (with proper headers)
- Any other calendar service that provides ICS format

## ICS Format Support

The parser handles:
- Multi-line values (folded lines)
- Date-time formats (with and without timezone)
- Date-only formats
- Various ICS properties (UID, SUMMARY, DESCRIPTION, etc.)
- Different timezone definitions

## Example Output

The program outputs events from all organizations in JSON format:

```json
[
  {
    "organization": "DATA",
    "uid": "example@google.com",
    "summary": "Event Title",
    "description": "Event description",
    "location": "Event location",
    "start_time": "2025-01-01T10:00:00Z",
    "end_time": "2025-01-01T11:00:00Z",
    "created": "2025-01-01T09:00:00Z",
    "modified": "2025-01-01T09:00:00Z",
    "status": "CONFIRMED",
    "transparency": "OPAQUE"
  },
  {
    "organization": "Dallas Urbanists STLC",
    "uid": "event_123@meetup.com",
    "summary": "Meetup Event",
    "description": "Meetup description",
    "location": "",
    "start_time": "2025-01-02T18:00:00Z",
    "end_time": "2025-01-02T20:00:00Z",
    "created": "2025-01-01T10:00:00Z",
    "modified": "2025-01-01T10:00:00Z",
    "status": "CONFIRMED",
    "transparency": ""
  }
]
```

## Error Handling

- Individual organization failures don't stop the entire process
- Detailed error messages for debugging
- Graceful handling of network issues and parsing errors

## Requirements

- Go 1.16 or later
- Internet connection to fetch ICS files
- Valid `config.json` file with organization mappings

## License

This project is part of the Dallas Urbanists organization.