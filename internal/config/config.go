package config

import (
	"encoding/json"
	"fmt"
	"os"
)

// Config represents the configuration file structure
type Config struct {
	Organizations map[string]string `json:"organizations"`
}

// LoadConfig loads the configuration from config.json
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