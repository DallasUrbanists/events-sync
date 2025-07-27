package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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

type UpdateStatusRequest struct {
	Status string `json:"status"`
}

func (s *Server) getUpcomingEvents(w http.ResponseWriter, r *http.Request) {
	events, err := s.db.GetUpcomingEvents()
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get events: %v", err), http.StatusInternalServerError)
		return
	}

	// Convert to response format
	var result []EventResponse
	for _, event := range events {
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
		result = append(result, eventResp)
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(result)
}

func (s *Server) updateEventStatus(w http.ResponseWriter, r *http.Request) {
	uid := r.PathValue("uid")

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

