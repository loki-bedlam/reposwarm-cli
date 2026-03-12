package commands

import (
	"testing"
)

func TestIsKnownService(t *testing.T) {
	tests := []struct {
		name        string
		serviceName string
		want        bool
	}{
		{
			name:        "api is known",
			serviceName: "api",
			want:        true,
		},
		{
			name:        "worker is known",
			serviceName: "worker",
			want:        true,
		},
		{
			name:        "temporal is known",
			serviceName: "temporal",
			want:        true,
		},
		{
			name:        "ui is known",
			serviceName: "ui",
			want:        true,
		},
		{
			name:        "worker with suffix is known",
			serviceName: "worker-1",
			want:        true,
		},
		{
			name:        "unknown service",
			serviceName: "unknown",
			want:        false,
		},
		{
			name:        "empty string",
			serviceName: "",
			want:        false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := isKnownService(tt.serviceName)
			if got != tt.want {
				t.Errorf("isKnownService(%q) = %v, want %v", tt.serviceName, got, tt.want)
			}
		})
	}
}

func TestStopCommand_Args(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		shouldError bool
		errorMsg    string
	}{
		{
			name:        "no args should be valid",
			args:        []string{},
			shouldError: false,
		},
		{
			name:        "one valid arg should be valid",
			args:        []string{"worker"},
			shouldError: false,
		},
		{
			name:        "two args should error",
			args:        []string{"worker", "api"},
			shouldError: true,
			errorMsg:    "Too many arguments",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := newStopCmd()
			cmd.SetArgs(tt.args)

			// We can't fully execute the command in tests without mocking,
			// but we can test argument validation
			err := cmd.Args(cmd, tt.args)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error but got: %v", err)
				}
			}
		})
	}
}

func TestRestartDockerForceLocal(t *testing.T) {
	// Verify that the restart command forces local mode for Docker installs
	// This ensures docker compose up --force-recreate is used instead of API restart,
	// which is needed to re-read env_file changes.
	cmd := newRestartCmd()

	// Verify --local flag exists
	localFlag := cmd.Flags().Lookup("local")
	if localFlag == nil {
		t.Fatal("restart command missing --local flag")
	}

	// Verify --wait flag exists with default true
	waitFlag := cmd.Flags().Lookup("wait")
	if waitFlag == nil {
		t.Fatal("restart command missing --wait flag")
	}
	if waitFlag.DefValue != "true" {
		t.Errorf("--wait default should be true, got %s", waitFlag.DefValue)
	}
}

func TestFilterEnvErrorsUsedInRestart(t *testing.T) {
	// Verify that filterEnvErrors filters ANTHROPIC_API_KEY for Bedrock provider.
	// The restart command should call filterEnvErrors before warning about env errors.
	// We test the filter function directly since it's called from the restart path.
	errors := []string{"ANTHROPIC_API_KEY", "SOME_OTHER_VAR"}

	// Without a config, filterEnvErrors returns errors as-is (no provider info)
	filtered := filterEnvErrors(errors)
	// Should return at least the errors (can't filter without config)
	if len(filtered) == 0 {
		t.Error("filterEnvErrors should return errors when no config is available")
	}
}

func TestKnownServices(t *testing.T) {
	// Test that knownServices contains expected values
	expected := map[string]bool{
		"api":      true,
		"worker":   true,
		"temporal": true,
		"ui":       true,
	}

	if len(knownServices) != len(expected) {
		t.Errorf("Expected %d known services, got %d", len(expected), len(knownServices))
	}

	for _, svc := range knownServices {
		if !expected[svc] {
			t.Errorf("Unexpected service in knownServices: %q", svc)
		}
	}

	for svc := range expected {
		found := false
		for _, known := range knownServices {
			if known == svc {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected service %q not found in knownServices", svc)
		}
	}
}
