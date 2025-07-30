package server

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

// DiscordTokenResponse represents the response from Discord's token endpoint
type DiscordTokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Scope        string `json:"scope"`
}

// DiscordUser represents a Discord user
type DiscordUser struct {
	ID       string `json:"id"`
	Username string `json:"username"`
	Email    string `json:"email"`
	Avatar   string `json:"avatar"`
}

// discordHandler handles the Discord OAuth2 callback
func (s *Server) discordHandler(w http.ResponseWriter, r *http.Request) {
	// Get the authorization code from the URL
	code := r.URL.Query().Get("code")
	if code == "" {
		http.Error(w, "No authorization code provided", http.StatusBadRequest)
		return
	}

	// Exchange the code for an access token
	tokenResp, err := s.exchangeCodeForToken(code)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to exchange code for token: %v", err), http.StatusInternalServerError)
		return
	}

	// Get the user's information from Discord
	discordUser, err := s.getDiscordUser(tokenResp.AccessToken)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to get Discord user: %v", err), http.StatusInternalServerError)
		return
	}

	// Check if the user is authenticated in our database
	authenticated, err := s.db.IsDiscordUserAuthenticated(discordUser.ID)
	if err != nil {
		http.Error(w, fmt.Sprintf("Database error: %v", err), http.StatusInternalServerError)
		return
	}

	if !authenticated {
		http.Error(w, "User not authorized to access this application", http.StatusForbidden)
		return
	}

	// Get the user from our database
	user, err := s.db.GetDiscordUserByID(discordUser.ID)
	if err != nil {
		// User doesn't exist in database, they are unauthorized
		http.Error(w, "User not found in database - unauthorized", http.StatusForbidden)
		return
	}

	// Generate JWT token
	token, err := s.generateJWT(user)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to generate token: %v", err), http.StatusInternalServerError)
		return
	}

	// Set the token as a cookie and redirect to the main page
	http.SetCookie(w, &http.Cookie{
		Name:     "auth_token",
		Value:    token,
		Path:     "/",
		HttpOnly: true,
		Secure:   false, // Set to true in production with HTTPS
		MaxAge:   86400, // 24 hours
	})

	// Redirect to the main page
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

// exchangeCodeForToken exchanges the authorization code for an access token
func (s *Server) exchangeCodeForToken(code string) (*DiscordTokenResponse, error) {
	// Create form data for Discord OAuth2 token request
	formData := url.Values{}
	formData.Set("client_id", s.discordConfig.ClientID)
	formData.Set("client_secret", s.discordConfig.ClientSecret)
	formData.Set("grant_type", "authorization_code")
	formData.Set("code", code)
	formData.Set("redirect_uri", s.discordConfig.RedirectURI)

	// Create the POST request to Discord's token endpoint
	req, err := http.NewRequest("POST", "https://discord.com/api/oauth2/token", strings.NewReader(formData.Encode()))
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set the content type for form data
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to Discord: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord API error: %s - %s", resp.Status, string(body))
	}

	// Parse the response
	var tokenResp DiscordTokenResponse
	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return nil, fmt.Errorf("error parsing token response: %v", err)
	}

	return &tokenResp, nil
}

// getDiscordUser retrieves the user's information from Discord
func (s *Server) getDiscordUser(accessToken string) (*DiscordUser, error) {
	// Create the GET request to Discord's user endpoint
	req, err := http.NewRequest("GET", "https://discord.com/api/users/@me", nil)
	if err != nil {
		return nil, fmt.Errorf("error creating request: %v", err)
	}

	// Set the authorization header
	req.Header.Set("Authorization", "Bearer "+accessToken)

	// Make the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("error making request to Discord: %v", err)
	}
	defer resp.Body.Close()

	// Read the response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading response: %v", err)
	}

	// Check if the request was successful
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Discord API error: %s - %s", resp.Status, string(body))
	}

	// Parse the response
	var discordUser DiscordUser
	if err := json.Unmarshal(body, &discordUser); err != nil {
		return nil, fmt.Errorf("error parsing user response: %v", err)
	}

	return &discordUser, nil
}
