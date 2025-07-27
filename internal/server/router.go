package server

import "net/http"

func (s *Server) newConfiguredRouter() *http.ServeMux {
	router := http.NewServeMux()

	router.Handle("GET /", http.FileServer(http.Dir("web")))

	router.HandleFunc("GET /api/events", s.getUpcomingEvents)
	router.HandleFunc("PUT /api/events/{uid}/status", s.updateEventStatus)
	router.HandleFunc("GET /api/events/stats", s.getEventStats)
	router.HandleFunc("GET /api/events/ical", s.generateICal)

	return router
}
