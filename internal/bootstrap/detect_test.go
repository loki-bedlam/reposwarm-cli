package bootstrap

import (
	"strings"
	"testing"
)

func TestDetect(t *testing.T) {
	env := Detect()

	if env.OS == "" {
		t.Error("OS should not be empty")
	}
	if env.Arch == "" {
		t.Error("Arch should not be empty")
	}
	if env.HomeDir == "" {
		t.Error("HomeDir should not be empty")
	}
	if env.WorkDir == "" {
		t.Error("WorkDir should not be empty")
	}
}

func TestDetectGit(t *testing.T) {
	env := Detect()
	// Git should be available in CI/test environments
	if !env.HasGit {
		t.Skip("git not available")
	}
	if env.GitVer == "" {
		t.Error("GitVer should not be empty when git is available")
	}
}

func TestSummary(t *testing.T) {
	env := Detect()
	summary := env.Summary()
	if summary == "" {
		t.Error("Summary should not be empty")
	}
	if !strings.Contains(summary, "System:") {
		t.Error("Summary should contain 'System:'")
	}
	if !strings.Contains(summary, "Runtimes:") {
		t.Error("Summary should contain 'Runtimes:'")
	}
}

func TestMissingDeps(t *testing.T) {
	env := &Environment{
		HasDocker:  false,
		HasCompose: false,
		HasNode:    true,
		HasPython:  true,
		HasGit:     true,
	}
	missing := env.MissingDeps()
	if len(missing) != 2 {
		t.Errorf("expected 2 missing deps, got %d: %v", len(missing), missing)
	}
}

func TestMissingDepsAllPresent(t *testing.T) {
	env := &Environment{
		HasDocker:  true,
		HasCompose: true,
		HasNode:    true,
		HasPython:  true,
		HasGit:     true,
	}
	missing := env.MissingDeps()
	if len(missing) != 0 {
		t.Errorf("expected 0 missing deps, got %d: %v", len(missing), missing)
	}
}

func TestAgentName(t *testing.T) {
	tests := []struct {
		env  Environment
		want string
	}{
		{Environment{HasClaudeCode: true, HasCodex: true}, "claude"},
		{Environment{HasCodex: true}, "codex"},
		{Environment{HasCursor: true}, "cursor"},
		{Environment{HasAider: true}, "aider"},
		{Environment{}, ""},
	}

	for _, tt := range tests {
		got := tt.env.AgentName()
		if got != tt.want {
			t.Errorf("AgentName() = %s, want %s", got, tt.want)
		}
	}
}

func TestInstallDir(t *testing.T) {
	env := &Environment{WorkDir: "/home/user/projects"}
	dir := env.InstallDir()
	if !strings.HasSuffix(dir, "/reposwarm") {
		t.Errorf("InstallDir() = %s, want suffix /reposwarm", dir)
	}
}

func TestGenerateGuide(t *testing.T) {
	env := &Environment{
		OS: "linux", Arch: "arm64",
		HasDocker: true, HasCompose: true,
		HasNode: true, HasPython: true, HasGit: true,
		AWSRegion: "us-east-1",
	}
	guide := GenerateGuide(env, "/tmp/test")
	if !strings.Contains(guide, "# RepoSwarm Local Installation Guide") {
		t.Error("guide should have title")
	}
	if !strings.Contains(guide, "docker compose up") {
		t.Error("guide should have docker compose step")
	}
	if !strings.Contains(guide, "npm install") {
		t.Error("guide should have npm install step")
	}
}

func TestGenerateAgentGuide(t *testing.T) {
	env := &Environment{
		OS: "darwin", Arch: "arm64",
		HasDocker: true, HasCompose: true,
		HasNode: true, HasPython: true, HasGit: true,
		AWSRegion: "us-west-2",
	}
	guide := GenerateAgentGuide(env, "/tmp/test")
	if !strings.Contains(guide, "Agent Instructions") {
		t.Error("agent guide should have agent title")
	}
	if !strings.Contains(guide, "Step 1:") {
		t.Error("agent guide should have numbered steps")
	}
	if !strings.Contains(guide, "us-west-2") {
		t.Error("agent guide should use detected region")
	}
	if !strings.Contains(guide, "**Verify:**") {
		t.Error("agent guide should have verification steps")
	}
}

func TestGenerateGuideWithMissing(t *testing.T) {
	env := &Environment{
		OS: "linux", Arch: "amd64",
		HasDocker: false, HasCompose: false,
		HasNode: false, HasPython: true, HasGit: true,
		HasBrew: false, HasApt: true,
		AWSRegion: "us-east-1",
	}
	guide := GenerateGuide(env, "/tmp/test")
	if !strings.Contains(guide, "Missing dependencies") {
		t.Error("guide should mention missing deps")
	}
}

func TestGenerateAgentGuideWithMissing(t *testing.T) {
	env := &Environment{
		OS: "darwin", Arch: "arm64",
		HasDocker: true, HasCompose: false,
		HasNode: true, HasPython: true, HasGit: true,
		HasBrew: true,
		AWSRegion: "us-east-1",
	}
	guide := GenerateAgentGuide(env, "/tmp/test")
	if !strings.Contains(guide, "Step 0:") {
		t.Error("agent guide should have Step 0 for missing deps")
	}
}
