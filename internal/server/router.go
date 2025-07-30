package server

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/dallasurbanists/events-sync/internal/middleware"
)

func (s *Server) newConfiguredRouter() *http.ServeMux {
	router := http.NewServeMux()

	// Public routes (no authentication required)
	router.Handle("GET /", s.authRedirect(s.serveIndex))
	router.HandleFunc("GET /login", s.loginHandler)
	router.HandleFunc("GET /logout", s.logoutHandler)
	router.HandleFunc("GET /api/auth/discord/redirect", s.discordHandler)
	router.HandleFunc("GET /api/events/ical", s.generateICal) // Public iCal endpoint

	// Protected routes (authentication required)
	private := http.NewServeMux()
	private.HandleFunc("GET /api/events", s.getUpcomingEvents)
	private.HandleFunc("PUT /api/events/{uid}/status", s.updateEventStatus)
	private.HandleFunc("GET /api/events/stats", s.getEventStats)

	ms := middleware.CreateMiddlewareStack(
		s.AuthMiddleware,
	)

	router.Handle("/", ms(private))
	return router
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
