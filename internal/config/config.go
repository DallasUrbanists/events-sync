package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Organizations map[string]string `json:"organizations"`
}

// DiscordConfig holds Discord OAuth configuration from environment variables
type DiscordConfig struct {
	ClientID     string
	ClientSecret string
	RedirectURI  string
}

// JWTConfig holds JWT configuration from environment variables
type JWTConfig struct {
	Secret string
}

func LoadConfig() (*Config, error) {
	file, err := os.Open("config.json")
	if err != nil {
		return nil, fmt.Errorf("error opening config file: %v", err)
	}
	defer file.Close()

	var config Config
	decoder := json.NewDecoder(file)
	err = decoder.Decode(&config)
	if err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	return &config, nil
}

// LoadDiscordConfig loads Discord configuration from environment variables
func LoadDiscordConfig() (*DiscordConfig, error) {
	clientID := os.Getenv("DISCORD_CLIENT_ID")
	if clientID == "" {
		return nil, fmt.Errorf("DISCORD_CLIENT_ID environment variable is required")
	}

	clientSecret := os.Getenv("DISCORD_CLIENT_SECRET")
	if clientSecret == "" {
		return nil, fmt.Errorf("DISCORD_CLIENT_SECRET environment variable is required")
	}

	redirectURI := os.Getenv("DISCORD_REDIRECT_URI")
	if redirectURI == "" {
		return nil, fmt.Errorf("DISCORD_REDIRECT_URI environment variable is required")
	}

	return &DiscordConfig{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		RedirectURI:  redirectURI,
	}, nil
}

// LoadJWTConfig loads JWT configuration from environment variables
func LoadJWTConfig() (*JWTConfig, error) {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		return nil, fmt.Errorf("JWT_SECRET environment variable is required")
	}

	return &JWTConfig{
		Secret: secret,
	}, nil
}