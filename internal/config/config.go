package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

type Organization struct {
	URL      string `json:"url"`
	Importer string `json:"importer"`
}

type Config struct {
	Organizations map[string]Organization `json:"organizations"`
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

	env_config_entries := Config{
		Organizations: map[string]Organization{},
	}

	prefix := "CONFIG_ORGANIZATIONS_"
	url_suffix := "_URL"
	importer_suffix := "_IMPORTER"

	envs := os.Environ()

	fmt.Printf("Environment variables with prefix '%s':\n", prefix)

	caser := cases.Title(language.English)
	for _, env := range envs {
		if strings.HasPrefix(env, prefix) {
			parts := strings.SplitN(env, "=", 2)
			if len(parts) == 2 {
				original_key := parts[0]
				key := strings.TrimPrefix(original_key, prefix)

				if strings.HasSuffix(key, url_suffix) {
					key = strings.TrimSuffix(key, url_suffix)
				} else if strings.HasSuffix(key, importer_suffix) {
					key = strings.TrimSuffix(key, importer_suffix)
				} else {
					continue
				}

				key = strings.ReplaceAll(key, "_", " ")
				key = caser.String(key)

				org, ok := env_config_entries.Organizations[key]
				if !ok {
					org = Organization{}
				}

				if strings.HasSuffix(original_key, url_suffix) {
					org.URL = parts[1]
				} else if strings.HasSuffix(original_key, importer_suffix) {
					org.Importer = parts[1]
				}

				env_config_entries.Organizations[key] = org
			}
		}
	}

	for k, v := range env_config_entries.Organizations {
		if v.Importer != "" && v.URL != "" {
			config.Organizations[k] = v
		}
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
