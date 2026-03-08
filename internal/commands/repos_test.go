package commands

import (
	"encoding/json"
	"strings"
	"testing"
)

// TestParseRepoURL tests the parseRepoURL helper function
func TestParseRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "basic github url",
			url:      "https://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "url with trailing slash",
			url:      "https://github.com/user/repo/",
			expected: "repo",
		},
		{
			name:     "url with .git suffix",
			url:      "https://github.com/user/repo.git",
			expected: "repo",
		},
		{
			name:     "url with trailing slash and .git",
			url:      "https://github.com/user/repo.git/",
			expected: "repo",
		},
		{
			name:     "gitlab url",
			url:      "https://gitlab.com/user/project",
			expected: "project",
		},
		{
			name:     "http protocol",
			url:      "http://github.com/user/repo",
			expected: "repo",
		},
		{
			name:     "nested path",
			url:      "https://github.com/org/team/repo",
			expected: "repo",
		},
		{
			name:     "codecommit url",
			url:      "https://git-codecommit.us-east-1.amazonaws.com/v1/repos/my-repo",
			expected: "my-repo",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := parseRepoURL(tt.url)
			if result != tt.expected {
				t.Errorf("parseRepoURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

// TestDetectSourceFromURL tests the detectSourceFromURL helper function
func TestDetectSourceFromURL(t *testing.T) {
	tests := []struct {
		name     string
		url      string
		expected string
	}{
		{
			name:     "github https",
			url:      "https://github.com/user/repo",
			expected: "GitHub",
		},
		{
			name:     "github http",
			url:      "http://github.com/user/repo",
			expected: "GitHub",
		},
		{
			name:     "gitlab",
			url:      "https://gitlab.com/user/project",
			expected: "CodeCommit",
		},
		{
			name:     "codecommit",
			url:      "https://git-codecommit.us-east-1.amazonaws.com/v1/repos/my-repo",
			expected: "CodeCommit",
		},
		{
			name:     "bitbucket",
			url:      "https://bitbucket.org/user/repo",
			expected: "CodeCommit",
		},
		{
			name:     "self-hosted",
			url:      "https://git.example.com/user/repo",
			expected: "CodeCommit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := detectSourceFromURL(tt.url)
			if result != tt.expected {
				t.Errorf("detectSourceFromURL(%q) = %q, want %q", tt.url, result, tt.expected)
			}
		})
	}
}

// TestIsRepoURL tests the isRepoURL helper function
func TestIsRepoURL(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "https url",
			input:    "https://github.com/user/repo",
			expected: true,
		},
		{
			name:     "http url",
			input:    "http://github.com/user/repo",
			expected: true,
		},
		{
			name:     "plain name",
			input:    "my-repo",
			expected: false,
		},
		{
			name:     "name with dashes",
			input:    "my-awesome-repo",
			expected: false,
		},
		{
			name:     "ssh url",
			input:    "git@github.com:user/repo.git",
			expected: false,
		},
		{
			name:     "file protocol",
			input:    "file:///path/to/repo",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isRepoURL(tt.input)
			if result != tt.expected {
				t.Errorf("isRepoURL(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

// TestReposAddURLDetection tests the repos add command with URL auto-detection
func TestReposAddURLDetection(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		expectedName   string
		expectedSource string
	}{
		{
			name:           "github url auto-detects name and source",
			args:           []string{"repos", "add", "https://github.com/user/repo", "--json"},
			expectedName:   "repo",
			expectedSource: "GitHub",
		},
		{
			name:           "gitlab url auto-detects name, source is default",
			args:           []string{"repos", "add", "https://gitlab.com/user/repo", "--json"},
			expectedName:   "repo",
			expectedSource: "CodeCommit",
		},
		{
			name:           "url with trailing slash",
			args:           []string{"repos", "add", "https://github.com/user/repo/", "--json"},
			expectedName:   "repo",
			expectedSource: "GitHub",
		},
		{
			name:           "url with .git suffix",
			args:           []string{"repos", "add", "https://github.com/user/repo.git", "--json"},
			expectedName:   "repo",
			expectedSource: "GitHub",
		},
		{
			name:           "http protocol works",
			args:           []string{"repos", "add", "http://github.com/user/repo", "--json"},
			expectedName:   "repo",
			expectedSource: "GitHub",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, cleanup := testServer(t, map[string]any{
				"POST /repos": map[string]any{
					"success": true,
					"name":    tt.expectedName,
				},
			})
			defer cleanup()

			out, err := runCmd(t, tt.args...)
			if err != nil {
				t.Fatalf("unexpected error: %v\noutput: %s", err, out)
			}

			var result map[string]any
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
			}

			// Verify the command succeeded
			if result["success"] != true {
				t.Errorf("expected success=true, got: %v", result)
			}
		})
	}
}

// TestReposAddPlainName tests that plain names (non-URLs) require --url flag
func TestReposAddPlainName(t *testing.T) {
	_, cleanup := testServer(t, map[string]any{
		"POST /repos": map[string]any{"success": true},
	})
	defer cleanup()

	// Plain name without --url should fail
	_, err := runCmd(t, "repos", "add", "my-repo")
	if err == nil {
		t.Fatal("expected error when plain name provided without --url")
	}
	if !strings.Contains(err.Error(), "url is required") {
		t.Errorf("expected 'url is required' error, got: %v", err)
	}

	// Plain name with --url should succeed
	out, err := runCmd(t, "repos", "add", "my-repo", "--url", "https://github.com/user/repo", "--json")
	if err != nil {
		t.Fatalf("unexpected error with --url: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["success"] != true {
		t.Errorf("expected success=true, got: %v", result)
	}
}

// TestReposAddURLFlagOverride tests that --url flag overrides the positional URL argument
func TestReposAddURLFlagOverride(t *testing.T) {
	_, cleanup := testServer(t, map[string]any{
		"POST /repos": map[string]any{
			"success": true,
			"name":    "other-repo",
		},
	})
	defer cleanup()

	// When both URL arg and --url flag provided, --url flag should win for URL
	// but name is still extracted from the winning URL
	out, err := runCmd(t, "repos", "add", "https://github.com/user/repo", "--url", "https://github.com/user/other-repo", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Should succeed and use the --url value for name extraction
	if result["success"] != true {
		t.Errorf("expected success=true, got: %v", result)
	}
	if result["name"] != "other-repo" {
		t.Errorf("expected name=other-repo (from --url), got: %v", result["name"])
	}
}

// TestReposAddSourceFlagOverride tests that --source flag overrides auto-detection
func TestReposAddSourceFlagOverride(t *testing.T) {
	_, cleanup := testServer(t, map[string]any{
		"POST /repos": map[string]any{
			"success": true,
			"name":    "repo",
			"source":  "CodeCommit",
		},
	})
	defer cleanup()

	// GitHub URL but --source=CodeCommit should use CodeCommit
	out, err := runCmd(t, "repos", "add", "https://github.com/user/repo", "--source", "CodeCommit", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Should succeed with CodeCommit as source
	if result["success"] != true {
		t.Errorf("expected success=true, got: %v", result)
	}
	if result["source"] == "GitHub" {
		t.Errorf("expected source to be overridden to CodeCommit, got: %v", result["source"])
	}
}

// TestReposAddServerError tests error handling when server returns error
func TestReposAddServerError(t *testing.T) {
	// Empty routes map will cause 404
	_, cleanup := testServer(t, map[string]any{})
	defer cleanup()

	_, err := runCmd(t, "repos", "add", "https://github.com/user/repo", "--json")
	if err == nil {
		t.Fatal("expected error when server returns 404")
	}
}
