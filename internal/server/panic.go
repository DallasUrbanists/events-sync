package server

import (
	"fmt"
	"net/http"
	"runtime/debug"
)

// PanicRecoveryMiddleware recovers from panics and logs them
func (s *Server) PanicRecoveryMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := s.getLogger(r)

		defer func() {
			if err := recover(); err != nil {
				l.Error(fmt.Sprintf("PANIC: %v\nStack trace:\n%s", err, debug.Stack()))
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
			}
		}()

		next.ServeHTTP(w, r)
	})
}
