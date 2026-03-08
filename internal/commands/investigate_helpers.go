package commands

import (
	"fmt"
	"time"

	"github.com/reposwarm/reposwarm-cli/internal/api"
)

// checkRecentInvestigations returns a map of repo names that have completed
// investigations within the last 24 hours, along with a human-readable time ago string.
func checkRecentInvestigations(client *api.Client, repoNames []string) map[string]string {
	recentMap := make(map[string]string)

	// Query recent workflows (last 100 should be enough to catch 24h of activity)
	var wfResult api.WorkflowsResponse
	if err := client.Get(ctx(), "/workflows?pageSize=100", &wfResult); err != nil {
		// If we can't fetch workflows, don't skip any repos
		return recentMap
	}

	now := time.Now()
	cutoff := now.Add(-24 * time.Hour)

	for _, wf := range wfResult.Executions {
		// Only check completed investigations
		if wf.Status != "Completed" && wf.Status != "COMPLETED" {
			continue
		}
		if wf.CloseTime == "" {
			continue
		}

		// Parse close time (RFC3339 format expected)
		closeTime, err := time.Parse(time.RFC3339, wf.CloseTime)
		if err != nil {
			// Try alternate format if RFC3339 fails
			closeTime, err = time.Parse("2006-01-02T15:04:05.999Z", wf.CloseTime)
			if err != nil {
				continue
			}
		}

		// Skip if older than 24 hours
		if closeTime.Before(cutoff) {
			continue
		}

		// Extract repo name from workflow ID
		repo := repoName(wf.WorkflowID)

		// Check if this repo is in our list
		found := false
		for _, r := range repoNames {
			if r == repo {
				found = true
				break
			}
		}
		if !found {
			continue
		}

		// Calculate time ago
		duration := now.Sub(closeTime)
		timeAgo := formatTimeAgo(duration)

		// Only store the most recent investigation per repo
		if _, exists := recentMap[repo]; !exists {
			recentMap[repo] = timeAgo
		}
	}

	return recentMap
}

// formatTimeAgo formats a duration as a human-readable "time ago" string.
func formatTimeAgo(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		mins := int(d.Minutes())
		if mins == 1 {
			return "1 min ago"
		}
		return fmt.Sprintf("%d mins ago", mins)
	}
	if d < 24*time.Hour {
		hours := int(d.Hours())
		if hours == 1 {
			return "1h ago"
		}
		return fmt.Sprintf("%dh ago", hours)
	}
	days := int(d.Hours() / 24)
	if days == 1 {
		return "1 day ago"
	}
	return fmt.Sprintf("%d days ago", days)
}
