package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/dallasurbanists/events-sync/internal/middleware"
)

func (s *Server) newConfiguredRouter() *http.ServeMux {
	router := http.NewServeMux()

	open_ms := middleware.CreateMiddlewareStack(
		s.LoggerMiddleware,
		s.PanicRecoveryMiddleware,
	)

	authed_ms :=middleware.CreateMiddlewareStack(
		open_ms,
		s.AuthMiddleware,
	)

	// Public routes (no authentication required)
	router.Handle("GET /", open_ms(s.authRedirect(s.serveIndex)))

	router.Handle("GET /login", open_ms(http.HandlerFunc(s.loginHandler)))
	router.Handle("GET /logout", open_ms(http.HandlerFunc(s.logoutHandler)))
	router.Handle("GET /auth/discord/redirect", open_ms(http.HandlerFunc(s.discordHandler)))
	router.Handle("GET /ical", open_ms(http.HandlerFunc(s.generateICal))) // Public iCal endpoint

	// Protected routes (authentication required)
	router.Handle("GET /api/events", authed_ms(http.HandlerFunc(s.getUpcomingEvents)))
	router.Handle("PATCH /api/events/{uid}", authed_ms(http.HandlerFunc(s.updateEvent)))
	router.Handle("GET /api/events/stats", authed_ms(http.HandlerFunc(s.getEventStats)))
	router.Handle("POST /api/events/{uid}/overlay", authed_ms(http.HandlerFunc(s.setEventOverlay)))
	router.Handle("DELETE /api/events/{uid}/overlay/{field}", authed_ms(http.HandlerFunc(s.removeEventOverlay)))
	router.Handle("GET /api/version", open_ms(http.HandlerFunc(s.getVersion)))

	// Wrap the entire router with panic recovery for public routes too
	wrappedRouter := http.NewServeMux()
	wrappedRouter.Handle("/", s.PanicRecoveryMiddleware(router))
	return wrappedRouter
}

// authRedirect checks if user is authenticated and redirects to login if not
func (s *Server) authRedirect(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Check for auth cookie
		cookie, err := r.Cookie("auth_token")
		if err != nil || cookie.Value == "" {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// Validate the token
		_, err = s.parseJWT(cookie.Value)
		if err != nil {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}

		// User is authenticated, serve the requested page
		next.ServeHTTP(w, r)
	}
}

// serveIndex serves the main application page
func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	http.FileServer(http.Dir("web")).ServeHTTP(w, r)
}

// loginHandler serves the login page
func (s *Server) loginHandler(w http.ResponseWriter, r *http.Request) {
	// Create the Discord OAuth2 authorization URL
	authURL := fmt.Sprintf(
		"https://discord.com/api/oauth2/authorize?client_id=%s&redirect_uri=%s&response_type=code&scope=identify",
		s.discordConfig.ClientID,
		url.QueryEscape(s.discordConfig.RedirectURI),
	)

	// Serve a simple login page
	w.Header().Set("Content-Type", "text/html")
	fmt.Fprintf(w, `
<!DOCTYPE html>
<html>
<head>
    <title>Login - Events Sync</title>
    <style>
        body { font-family: Arial, sans-serif; text-align: center; padding: 50px; }
        .login-button {
            background: #5865F2;
            color: white;
            padding: 15px 30px;
            border: none;
            border-radius: 5px;
            font-size: 16px;
            cursor: pointer;
            text-decoration: none;
            display: inline-block;
        }
        .login-button:hover { background: #4752C4; }
    </style>
</head>
<body>
    <h1>Welcome to Events Sync</h1>
    <p>Please log in with your Discord account to continue.</p>
    <a href="%s" class="login-button">Login with Discord</a>
</body>
</html>
`, authURL)
}

// logoutHandler handles user logout
func (s *Server) logoutHandler(w http.ResponseWriter, r *http.Request) {
	// Clear the auth cookie
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		MaxAge:   -1, // Delete the cookie
	})

	// Redirect to login page
	http.Redirect(w, r, "/login", http.StatusSeeOther)
}
