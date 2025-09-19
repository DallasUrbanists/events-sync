package server

import (
	"context"
	"log/slog"
	"net/http"

	"github.com/dallasurbanists/events-sync/internal/logger"
	"github.com/google/uuid"
)

func (s *Server) LoggerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		req_id := uuid.New()

		s.Logger.Debug("server received request",
			"request_id", req_id.String(),
			"pattern", r.Pattern,
			"host", r.Host,
			"form", r.PostForm,
			"remote", r.RemoteAddr,
			"referer", r.Referer(),
		)

		l := logger.NewLogger().With(
			slog.String("request_id", req_id.String()),
		)

		ctx := context.WithValue(r.Context(), "logger", l)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (s *Server) getLogger(r *http.Request) *slog.Logger {
	l := s.Logger

	ctxlogger := r.Context().Value("logger")
	if ctxlogger != nil {
		l = ctxlogger.(*slog.Logger)
	}

	return l
}

func (s *Server) setLogger(l *slog.Logger, r *http.Request) *http.Request {
	ctx := context.WithValue(r.Context(), "logger", l)
	return r.WithContext(ctx)
}

