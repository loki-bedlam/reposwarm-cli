package commands

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/reposwarm/reposwarm-cli/internal/api"
)

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		duration time.Duration
		want     string
	}{
		{
			name:     "less than a minute",
			duration: 30 * time.Second,
			want:     "just now",
		},
		{
			name:     "one minute",
			duration: 1 * time.Minute,
			want:     "1 min ago",
		},
		{
			name:     "multiple minutes",
			duration: 45 * time.Minute,
			want:     "45 mins ago",
		},
		{
			name:     "one hour",
			duration: 1 * time.Hour,
			want:     "1h ago",
		},
		{
			name:     "multiple hours",
			duration: 5 * time.Hour,
			want:     "5h ago",
		},
		{
			name:     "one day",
			duration: 24 * time.Hour,
			want:     "1 day ago",
		},
		{
			name:     "multiple days",
			duration: 72 * time.Hour,
			want:     "3 days ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatTimeAgo(tt.duration)
			if got != tt.want {
				t.Errorf("formatTimeAgo(%v) = %q, want %q", tt.duration, got, tt.want)
			}
		})
	}
}

func TestCheckRecentInvestigations(t *testing.T) {
	// Create a test server that returns workflow data
	tests := []struct {
		name          string
		workflows     []api.WorkflowExecution
		repoNames     []string
		expectSkipped []string
	}{
		{
			name: "recent completed investigation should be skipped",
			workflows: []api.WorkflowExecution{
				{
					WorkflowID: "investigate-single-is-odd-1234567890",
					Status:     "Completed",
					CloseTime:  time.Now().Add(-2 * time.Hour).Format(time.RFC3339),
				},
			},
			repoNames:     []string{"is-odd", "is-even"},
			expectSkipped: []string{"is-odd"},
		},
		{
			name: "old completed investigation should not be skipped",
			workflows: []api.WorkflowExecution{
				{
					WorkflowID: "investigate-single-is-odd-1234567890",
					Status:     "Completed",
					CloseTime:  time.Now().Add(-25 * time.Hour).Format(time.RFC3339),
				},
			},
			repoNames:     []string{"is-odd"},
			expectSkipped: []string{},
		},
		{
			name: "running investigation should not be skipped",
			workflows: []api.WorkflowExecution{
				{
					WorkflowID: "investigate-single-is-odd-1234567890",
					Status:     "Running",
					CloseTime:  "",
				},
			},
			repoNames:     []string{"is-odd"},
			expectSkipped: []string{},
		},
		{
			name: "multiple repos with mixed states",
			workflows: []api.WorkflowExecution{
				{
					WorkflowID: "investigate-single-is-odd-1234567890",
					Status:     "Completed",
					CloseTime:  time.Now().Add(-1 * time.Hour).Format(time.RFC3339),
				},
				{
					WorkflowID: "investigate-single-is-even-9876543210",
					Status:     "Completed",
					CloseTime:  time.Now().Add(-30 * time.Hour).Format(time.RFC3339),
				},
			},
			repoNames:     []string{"is-odd", "is-even", "is-number"},
			expectSkipped: []string{"is-odd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test server
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path == "/workflows" {
					resp := api.WorkflowsResponse{
						Executions: tt.workflows,
					}
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]any{"data": resp})
					return
				}
				http.NotFound(w, r)
			}))
			defer server.Close()

			// Create API client
			client := api.New(server.URL, "test-token")

			// Call the function (ctx() is already available in the package)
			result := checkRecentInvestigations(client, tt.repoNames)

			// Verify results
			if len(result) != len(tt.expectSkipped) {
				t.Errorf("Expected %d skipped repos, got %d", len(tt.expectSkipped), len(result))
			}

			for _, expectedRepo := range tt.expectSkipped {
				if _, found := result[expectedRepo]; !found {
					t.Errorf("Expected repo %q to be in skipped list, but it wasn't", expectedRepo)
				}
			}

			// Verify that non-skipped repos are not in the result
			for _, repo := range tt.repoNames {
				isExpectedSkipped := false
				for _, skipped := range tt.expectSkipped {
					if repo == skipped {
						isExpectedSkipped = true
						break
					}
				}
				if !isExpectedSkipped {
					if _, found := result[repo]; found {
						t.Errorf("Repo %q should not be skipped, but it was", repo)
					}
				}
			}
		})
	}
}

func TestCheckRecentInvestigations_APIError(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	// Create API client
	client := api.New(server.URL, "test-token")

	// Call the function - should return empty map on error
	result := checkRecentInvestigations(client, []string{"is-odd"})

	if len(result) != 0 {
		t.Errorf("Expected empty map on API error, got %d entries", len(result))
	}
}
