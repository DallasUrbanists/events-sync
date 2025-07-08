package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	// Use the sample config in testdata
	configPath := filepath.Join("..", "..", "testdata", "sample_config.json")
	orig := os.Getenv("CONFIG_PATH")
	defer os.Setenv("CONFIG_PATH", orig)
	os.Setenv("CONFIG_PATH", configPath)

	file, err := os.Open(configPath)
	if err != nil {
		t.Fatalf("failed to open sample config: %v", err)
	}
	defer file.Close()

	var cfg Config
	if err := loadConfigFromReader(file, &cfg); err != nil {
		t.Fatalf("loadConfigFromReader failed: %v", err)
	}

	if len(cfg.Organizations) != 1 {
		t.Errorf("expected 1 organization, got %d", len(cfg.Organizations))
	}
	if url, ok := cfg.Organizations["TestOrg"]; !ok || url != "https://example.com/test.ics" {
		t.Errorf("unexpected organization mapping: %v", cfg.Organizations)
	}
}

// loadConfigFromReader is a helper for testing
func loadConfigFromReader(f *os.File, cfg *Config) error {
	decoder := json.NewDecoder(f)
	return decoder.Decode(cfg)
}