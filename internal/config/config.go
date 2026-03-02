// Package config manages CLI configuration stored in ~/.reposwarm/config.json.
package config

import (
	"encoding/json"
	"github.com/loki-bedlam/reposwarm-cli/internal/bootstrap"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config holds all CLI configuration.
type Config struct {
	APIUrl       string `json:"apiUrl"`
	APIToken     string `json:"apiToken"`
	Region       string `json:"region"`
	DefaultModel string `json:"defaultModel"`
	ChunkSize    int    `json:"chunkSize"`
	OutputFormat string `json:"outputFormat"`
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		APIUrl:       "http://localhost:3000/v1",
		Region:       "us-east-1",
		DefaultModel: bootstrap.DefaultModel,
		ChunkSize:    10,
		OutputFormat: "pretty",
	}
}

// ValidKeys returns the list of settable config keys.
func ValidKeys() []string {
	return []string{"apiUrl", "apiToken", "region", "defaultModel", "chunkSize", "outputFormat"}
}

// ConfigDir returns the config directory path.
func ConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("finding home directory: %w", err)
	}
	return filepath.Join(home, ".reposwarm"), nil
}

// ConfigPath returns the config file path.
func ConfigPath() (string, error) {
	dir, err := ConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads config from disk, falling back to defaults.
// Environment variables REPOSWARM_API_URL and REPOSWARM_API_TOKEN override file values.
func Load() (*Config, error) {
	cfg := DefaultConfig()

	path, err := ConfigPath()
	if err != nil {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			applyEnvOverrides(cfg)
			return cfg, nil
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	applyEnvOverrides(cfg)
	return cfg, nil
}

func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("REPOSWARM_API_URL"); v != "" {
		cfg.APIUrl = v
	}
	if v := os.Getenv("REPOSWARM_API_TOKEN"); v != "" {
		cfg.APIToken = v
	}
}

// Save writes config to disk.
func Save(cfg *Config) error {
	dir, err := ConfigDir()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("encoding config: %w", err)
	}

	path := filepath.Join(dir, "config.json")
	return os.WriteFile(path, data, 0600)
}

// Set updates a single config key.
func Set(cfg *Config, key, value string) error {
	switch key {
	case "apiUrl":
		cfg.APIUrl = value
	case "apiToken":
		cfg.APIToken = value
	case "region":
		cfg.Region = value
	case "defaultModel":
		cfg.DefaultModel = value
	case "chunkSize":
		var n int
		if _, err := fmt.Sscanf(value, "%d", &n); err != nil {
			return fmt.Errorf("chunkSize must be a number")
		}
		cfg.ChunkSize = n
	case "outputFormat":
		if value != "pretty" && value != "json" {
			return fmt.Errorf("outputFormat must be 'pretty' or 'json'")
		}
		cfg.OutputFormat = value
	default:
		return fmt.Errorf("unknown config key: %s (valid: %s)", key, strings.Join(ValidKeys(), ", "))
	}
	return nil
}

// MaskedToken returns a token with most characters replaced by *.
func MaskedToken(token string) string {
	if len(token) <= 8 {
		return "***"
	}
	return "***..." + token[len(token)-6:]
}
