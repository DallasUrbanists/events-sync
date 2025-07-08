package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/dallasurbanists/events-sync/internal/database"
	"github.com/gorilla/mux"
)

// Server wraps the HTTP server and database
type Server struct {
	db     *database.DB
	router *mux.Router
}

// EventResponse represents an event in the API response
type EventResponse struct {
	ID           int       `json:"id"`
	UID          string    `json:"uid"`
	Organization string    `json:"organization"`
	Summary      string    `json:"summary"`
	Description  *string   `json:"description"`
	Location     *string   `json:"location"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	ReviewStatus string    `json:"review_status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EventsByDate groups events by date
type EventsByDate struct {
	Date   string           `json:"date"`
	Events []EventResponse  `json:"events"`
}

// UpdateStatusRequest represents a status update request
type UpdateStatusRequest struct {
	Status string `json:"status"`
}

// NewServer creates a new server instance
func NewServer(db *database.DB) *Server {
	s := &Server{
		db:     db,
		router: mux.NewRouter(),
	}
	s.setupRoutes()
	return s
}

// setupRoutes configures all HTTP routes
func (s *Server) setupRoutes() {
	// API routes
	s.router.HandleFunc("/api/events", s.getEvents).Methods("GET")
	s.router.HandleFunc("/api/events/{uid}/status", s.updateEventStatus).Methods("PUT")
	s.router.HandleFunc("/api/events/stats", s.getEventStats).Methods("GET")

	// Serve static files (HTML, CSS, JS)
	s.router.PathPrefix("/").Handler(http.FileServer(http.Dir("web-static")))
}

// getEvents returns all events grouped by date
func (s *Server) getEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.db.GetEvents()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	// Filter out past events and get current time
	now := time.Now()
	var filteredEvents []database.Event
	for _, event := range events {
		if event.EndTime.After(now) {
			filteredEvents = append(filteredEvents, event)
		}
	}

	// Group events by date
	eventsByDate := make(map[string][]EventResponse)
	for _, event := range filteredEvents {
		date := event.StartTime.Format("2006-01-02")
		eventResp := EventResponse{
			ID:           event.ID,
			UID:          event.UID,
			Organization: event.Organization,
			Summary:      event.Summary,
			Description:  event.Description,
			Location:     event.Location,
			StartTime:    event.StartTime,
			EndTime:      event.EndTime,
			ReviewStatus: event.ReviewStatus,
			CreatedAt:    event.CreatedAt,
			UpdatedAt:    event.UpdatedAt,
		}
		eventsByDate[date] = append(eventsByDate[date], eventResp)
	}

	// Convert to slice and sort by date proximity to current date
	var result []EventsByDate
	for date, events := range eventsByDate {
		result = append(result, EventsByDate{
			Date:   date,
			Events: events,
		})
	}

	// Sort by date proximity (closest to current date first)
	nowDate := now.Format("2006-01-02")
	for i := 0; i < len(result)-1; i++ {
		for j := i + 1; j < len(result); j++ {
			// Calculate days difference from current date
			dateI, _ := time.Parse("2006-01-02", result[i].Date)
			dateJ, _ := time.Parse("2006-01-02", result[j].Date)
			nowParsed, _ := time.Parse("2006-01-02", nowDate)

			diffI := dateI.Sub(nowParsed).Hours() / 24
			diffJ := dateJ.Sub(nowParsed).Hours() / 24

			// Sort by absolute difference (closest first)
			if abs(diffI) > abs(diffJ) {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

// updateEventStatus updates the review status of an event
func (s *Server) updateEventStatus(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	uid := vars["uid"]

	var req UpdateStatusRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Validate status
	if req.Status != "pending" && req.Status != "reviewed" && req.Status != "rejected" {
		http.Error(w, "Invalid status. Must be 'pending', 'reviewed', or 'rejected'", http.StatusBadRequest)
		return
	}

	if err := s.db.UpdateReviewStatus(uid, req.Status); err != nil {
		http.Error(w, fmt.Sprintf("Failed to update status: %v", err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"status": "success"})
}

// getEventStats returns statistics about events
func (s *Server) getEventStats(w http.ResponseWriter, r *http.Request) {
	stats := make(map[string]int)
	statuses := []string{"pending", "reviewed", "rejected"}

	for _, status := range statuses {
		events, err := s.db.GetEventsByReviewStatus(status)
		if err != nil {
			http.Error(w, fmt.Sprintf("Failed to get events with status %s: %v", status, err), http.StatusInternalServerError)
			return
		}
		stats[status] = len(events)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

// Start starts the HTTP server
func (s *Server) Start(addr string) error {
	fmt.Printf("Starting server on %s\n", addr)
	return http.ListenAndServe(addr, s.router)
}

// abs returns the absolute value of a float64
func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}