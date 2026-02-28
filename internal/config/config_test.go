package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.APIUrl == "" {
		t.Error("default APIUrl should not be empty")
	}
	if cfg.Region != "us-east-1" {
		t.Errorf("default region = %s, want us-east-1", cfg.Region)
	}
	if cfg.ChunkSize != 10 {
		t.Errorf("default chunkSize = %d, want 10", cfg.ChunkSize)
	}
}

func TestSaveAndLoad(t *testing.T) {
	// Use temp dir
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	cfg := DefaultConfig()
	cfg.APIUrl = "https://test.example.com/v1"
	cfg.APIToken = "test-token-123"

	if err := Save(cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Verify file exists
	path := filepath.Join(dir, ".reposwarm", "config.json")
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("config file not created: %v", err)
	}

	loaded, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if loaded.APIUrl != cfg.APIUrl {
		t.Errorf("APIUrl = %s, want %s", loaded.APIUrl, cfg.APIUrl)
	}
	if loaded.APIToken != cfg.APIToken {
		t.Errorf("APIToken = %s, want %s", loaded.APIToken, cfg.APIToken)
	}
}

func TestEnvOverrides(t *testing.T) {
	dir := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", dir)
	defer os.Setenv("HOME", origHome)

	os.Setenv("REPOSWARM_API_URL", "https://env.example.com")
	os.Setenv("REPOSWARM_API_TOKEN", "env-token")
	defer os.Unsetenv("REPOSWARM_API_URL")
	defer os.Unsetenv("REPOSWARM_API_TOKEN")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if cfg.APIUrl != "https://env.example.com" {
		t.Errorf("APIUrl = %s, want env override", cfg.APIUrl)
	}
	if cfg.APIToken != "env-token" {
		t.Errorf("APIToken = %s, want env override", cfg.APIToken)
	}
}

func TestSetValidKeys(t *testing.T) {
	cfg := DefaultConfig()
	tests := []struct {
		key, value string
		wantErr    bool
	}{
		{"apiUrl", "https://new.com", false},
		{"apiToken", "new-token", false},
		{"region", "eu-west-1", false},
		{"chunkSize", "20", false},
		{"chunkSize", "notanumber", true},
		{"outputFormat", "json", false},
		{"outputFormat", "xml", true},
		{"bogusKey", "value", true},
	}

	for _, tt := range tests {
		err := Set(cfg, tt.key, tt.value)
		if (err != nil) != tt.wantErr {
			t.Errorf("Set(%s, %s) err = %v, wantErr = %v", tt.key, tt.value, err, tt.wantErr)
		}
	}
}

func TestMaskedToken(t *testing.T) {
	tests := []struct {
		input, want string
	}{
		{"", "***"},
		{"short", "***"},
		{"abcdefghijklmnop", "***...klmnop"},
	}
	for _, tt := range tests {
		got := MaskedToken(tt.input)
		if got != tt.want {
			t.Errorf("MaskedToken(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestValidKeys(t *testing.T) {
	keys := ValidKeys()
	if len(keys) == 0 {
		t.Error("ValidKeys should not be empty")
	}
}
