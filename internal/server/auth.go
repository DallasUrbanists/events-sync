package server

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/dallasurbanists/events-sync/pkg/discord"
	"github.com/golang-jwt/jwt/v5"
)

// Claims represents the JWT claims
type Claims struct {
	DiscordID string `json:"discord_id"`
	Username  string `json:"username"`
	jwt.RegisteredClaims
}

// AuthMiddleware wraps an http.Handler and checks for valid JWT authentication
func (s *Server) AuthMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		l := s.getLogger(r)

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			l.Warn("auth header missing, falling back onto cookie based auth")

			cookie, err := r.Cookie("auth_token")
			if err != nil || cookie.Value == "" {
				l.Error("no auth present, unauthorized")

				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}
			authHeader = "Bearer " + cookie.Value
		}

		if !strings.HasPrefix(authHeader, "Bearer ") {
			http.Error(w, "Invalid authorization header", http.StatusUnauthorized)
			return
		}
		tokenString := strings.TrimPrefix(authHeader, "Bearer ")

		claims, err := s.parseJWT(tokenString)
		if err != nil {
			l.Error(fmt.Sprintf("failed to parse JWT: %v", err))
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		user, err := s.db.AuthenticatedDiscordUsers.GetDiscordUserByID(claims.DiscordID)
		if err != nil {
			l.Error(fmt.Sprintf("DB error while verifying discord user %v: %v", claims.DiscordID, err))
			http.Error(w, "Database error", http.StatusInternalServerError)
			return
		}
		if user == nil {
			l.Error(fmt.Sprintf("user %v from claim no longer authenticated: %v", claims.DiscordID, err))
			http.Error(w, "User no longer authenticated", http.StatusUnauthorized)
			return
		}

		l = l.With("user", user.DiscordID)
		authed_req := s.setLogger(l, r)
		ctx := context.WithValue(r.Context(), "user", claims)

		next.ServeHTTP(w, authed_req.WithContext(ctx))
	})
}

// parseJWT parses and validates a JWT token
func (s *Server) parseJWT(tokenString string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &Claims{}, func(token *jwt.Token) (interface{}, error) {
		return []byte(s.jwtConfig.Secret), nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %v", err)
	}

	if claims, ok := token.Claims.(*Claims); ok && token.Valid {
		return claims, nil
	}

	return nil, fmt.Errorf("invalid token")
}

// generateJWT creates a new JWT token for a user
func (s *Server) generateJWT(user *discord.AuthenticatedUser) (string, error) {
	claims := &Claims{
		DiscordID: user.DiscordID,
		Username:  user.Username,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)), // 24 hour expiration
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.jwtConfig.Secret))
}

// GetUserFromContext extracts user information from the request context
func GetUserFromContext(ctx context.Context) (*Claims, bool) {
	user, ok := ctx.Value("user").(*Claims)
	return user, ok
}
