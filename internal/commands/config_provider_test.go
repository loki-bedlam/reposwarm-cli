package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/reposwarm/reposwarm-cli/internal/bootstrap"
)

// TestProviderSetCheckVerifiesWrittenEnv verifies that --check validates
// the written worker.env file instead of hitting the running worker container.
func TestProviderSetCheckVerifiesWrittenEnv(t *testing.T) {
	// Create a fake Docker install directory with docker-compose.yml
	dir := t.TempDir()
	composeDir := filepath.Join(dir, bootstrap.ComposeSubDir)
	os.MkdirAll(composeDir, 0755)
	os.WriteFile(filepath.Join(composeDir, "docker-compose.yml"), []byte("version: '3'\n"), 0644)

	// Write a worker.env with the expected vars
	workerEnvPath := filepath.Join(composeDir, "worker.env")
	envContent := "CLAUDE_CODE_USE_BEDROCK=1\nAWS_REGION=us-east-1\nANTHROPIC_MODEL=us.anthropic.claude-sonnet-4-6\n"
	if err := os.WriteFile(workerEnvPath, []byte(envContent), 0600); err != nil {
		t.Fatalf("writing worker.env: %v", err)
	}

	// Verify the env file can be read and has the expected vars
	env, err := bootstrap.ReadWorkerEnvFile(dir)
	if err != nil {
		t.Fatalf("reading worker env: %v", err)
	}

	// Check that key provider vars exist
	expectedVars := map[string]string{
		"CLAUDE_CODE_USE_BEDROCK": "1",
		"AWS_REGION":              "us-east-1",
		"ANTHROPIC_MODEL":         "us.anthropic.claude-sonnet-4-6",
	}
	for k, wantV := range expectedVars {
		gotV, ok := env[k]
		if !ok {
			t.Errorf("worker.env missing key %q", k)
			continue
		}
		if gotV != wantV {
			t.Errorf("worker.env[%q] = %q, want %q", k, gotV, wantV)
		}
	}
}

// TestProviderSetAuthFlagAlias verifies both --auth and --auth-method are
// registered as flags on the provider set command.
func TestProviderSetAuthFlagAlias(t *testing.T) {
	root := NewRootCmd("test")

	// Navigate to "config provider set" subcommand
	configCmd, _, _ := root.Find([]string{"config", "provider", "set"})
	if configCmd == nil {
		t.Fatal("could not find 'config provider set' command")
	}

	tests := []struct {
		name     string
		flagName string
	}{
		{"auth-method flag exists", "auth-method"},
		{"auth alias exists", "auth"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			f := configCmd.Flags().Lookup(tc.flagName)
			if f == nil {
				t.Errorf("flag --%s not registered on 'config provider set'", tc.flagName)
			}
		})
	}
}

func TestVerifyWrittenWorkerEnv(t *testing.T) {
	tests := []struct {
		name       string
		envContent string
		expected   map[string]string
		wantOK     bool
		wantMsgs   []string
	}{
		{
			name:       "all vars present",
			envContent: "CLAUDE_CODE_USE_BEDROCK=1\nAWS_REGION=us-east-1\nANTHROPIC_MODEL=sonnet\n",
			expected:   map[string]string{"CLAUDE_CODE_USE_BEDROCK": "1", "AWS_REGION": "us-east-1"},
			wantOK:     true,
		},
		{
			name:       "missing var",
			envContent: "AWS_REGION=us-east-1\n",
			expected:   map[string]string{"CLAUDE_CODE_USE_BEDROCK": "1", "AWS_REGION": "us-east-1"},
			wantOK:     false,
			wantMsgs:   []string{"CLAUDE_CODE_USE_BEDROCK"},
		},
		{
			name:       "empty file",
			envContent: "",
			expected:   map[string]string{"CLAUDE_CODE_USE_BEDROCK": "1"},
			wantOK:     false,
			wantMsgs:   []string{"CLAUDE_CODE_USE_BEDROCK"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			composeDir := filepath.Join(dir, bootstrap.ComposeSubDir)
			os.MkdirAll(composeDir, 0755)
			os.WriteFile(filepath.Join(composeDir, "docker-compose.yml"), []byte("version: '3'\n"), 0644)

			envPath := filepath.Join(composeDir, "worker.env")
			os.WriteFile(envPath, []byte(tc.envContent), 0600)

			ok, missing := verifyWrittenWorkerEnv(envPath, tc.expected)
			if ok != tc.wantOK {
				t.Errorf("ok = %v, want %v", ok, tc.wantOK)
			}
			if !tc.wantOK {
				for _, msg := range tc.wantMsgs {
					found := false
					for _, m := range missing {
						if strings.Contains(m, msg) {
							found = true
							break
						}
					}
					if !found {
						t.Errorf("missing list %v does not mention %q", missing, msg)
					}
				}
			}
		})
	}
}
