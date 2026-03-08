package commands

import (
	"encoding/json"
	"fmt"
	"os"
	osexec "os/exec"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/reposwarm/reposwarm-cli/internal/api"
	"github.com/reposwarm/reposwarm-cli/internal/bootstrap"
	"github.com/reposwarm/reposwarm-cli/internal/config"
	"github.com/reposwarm/reposwarm-cli/internal/output"
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

// waitForWorkflow polls GET /workflows/<id> until the workflow reaches a terminal state.
// Returns the final status string ("Completed", "Failed", etc.) and any error.
func waitForWorkflow(client *api.Client, workflowID string, interval int) (string, error) {
	for {
		var wf api.WorkflowExecution
		if err := client.Get(ctx(), "/workflows/"+workflowID, &wf); err != nil {
			// Transient error — retry
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}

		lower := strings.ToLower(wf.Status)
		if lower == "completed" || lower == "failed" || lower == "terminated" || lower == "timed_out" || lower == "cancelled" {
			return wf.Status, nil
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

// ensureWorkerParallel sets REPOSWARM_PARALLEL in worker.env and restarts the
// worker if the value has changed. This is the mechanism by which the CLI
// --parallel flag dynamically controls worker concurrency.
func ensureWorkerParallel(client *api.Client, parallel int) error {
	cfg, err := config.Load()
	if err != nil {
		return nil // non-Docker install or no config — skip silently
	}

	installDir := cfg.EffectiveInstallDir()
	if !bootstrap.IsDockerInstall(installDir) {
		return nil // not a Docker install — nothing to configure
	}

	// Read current worker.env
	envVars, err := bootstrap.ReadWorkerEnvFile(installDir)
	if err != nil {
		envVars = make(map[string]string)
	}

	desired := strconv.Itoa(parallel)
	current := envVars["REPOSWARM_PARALLEL"]

	if current == desired {
		return nil // already set — no restart needed
	}

	// Check for running workflows before restarting
	var wfResult api.WorkflowsResponse
	if err := client.Get(ctx(), "/workflows?pageSize=50", &wfResult); err == nil {
		for _, w := range wfResult.Executions {
			if strings.EqualFold(w.Status, "Running") {
				return fmt.Errorf("cannot change parallelism: workflow %s is still running\n  Wait for it to finish or terminate it first: reposwarm workflows terminate %s", w.WorkflowID, w.WorkflowID)
			}
		}
	}

	// Write new value
	envVars["REPOSWARM_PARALLEL"] = desired
	envPath := filepath.Join(installDir, bootstrap.ComposeSubDir, "worker.env")
	if err := writeWorkerEnvMap(envPath, envVars); err != nil {
		return fmt.Errorf("writing worker.env: %w", err)
	}

	// Also sync via API (best-effort)
	body := map[string]string{"value": desired}
	var resp any
	_ = client.Put(ctx(), "/workers/worker-1/env/REPOSWARM_PARALLEL", body, &resp)

	// Restart worker to pick up the new value
	output.F.Info(fmt.Sprintf("Setting REPOSWARM_PARALLEL=%s, restarting worker...", desired))
	composeDir := filepath.Join(installDir, bootstrap.ComposeSubDir)
	stopCmd := osexec.Command("docker", "compose", "stop", "worker")
	stopCmd.Dir = composeDir
	stopCmd.CombinedOutput() // ignore error if already stopped

	restartCmd := osexec.Command("docker", "compose", "up", "-d", "--force-recreate", "worker")
	restartCmd.Dir = composeDir
	if out, err := restartCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("could not restart worker: %v (%s)", err, string(out))
	}

	// Wait for worker to be ready
	time.Sleep(5 * time.Second)
	output.Successf("Worker restarted with REPOSWARM_PARALLEL=%s", desired)
	return nil
}

// syncReposFromJSON reads repos.json and registers each repo with the API.
// Search order:
//  1. <installDir>/repos.json (local override)
//  2. Docker: docker exec reposwarm-worker cat /app/prompts/repos.json
//     Source: <installDir>/worker/prompts/repos.json
//
// Returns the number of repos successfully added.
func syncReposFromJSON(client *api.Client) (int, error) {
	cfg, err := config.Load()
	if err != nil {
		return 0, fmt.Errorf("loading config: %w", err)
	}

	installDir := cfg.EffectiveInstallDir()

	// Read repos.json content — check local override first
	var data []byte
	localPath := filepath.Join(installDir, "repos.json")
	if out, err := os.ReadFile(localPath); err == nil {
		data = out
	} else if bootstrap.IsDockerInstall(installDir) {
		cmd := osexec.Command("docker", "exec", "reposwarm-worker", "cat", "/app/prompts/repos.json")
		out, err := cmd.Output()
		if err != nil {
			return 0, fmt.Errorf("reading repos.json from worker container: %w", err)
		}
		data = out
	} else {
		path := filepath.Join(installDir, "worker", "prompts", "repos.json")
		out, err := os.ReadFile(path)
		if err != nil {
			return 0, fmt.Errorf("reading %s: %w", path, err)
		}
		data = out
	}

	// Parse repos.json — repositories values can be objects or comment strings
	var raw struct {
		Repositories map[string]json.RawMessage `json:"repositories"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, fmt.Errorf("parsing repos.json: %w", err)
	}

	type repoEntry struct {
		URL         string `json:"url"`
		Description string `json:"description"`
		Type        string `json:"type"`
	}

	repos := make(map[string]repoEntry)
	for name, rawVal := range raw.Repositories {
		var entry repoEntry
		if err := json.Unmarshal(rawVal, &entry); err != nil {
			continue // skip comment strings like "_comment_react": "=== ..."
		}
		if entry.URL == "" {
			continue // skip entries without a URL
		}
		repos[name] = entry
	}

	if len(repos) == 0 {
		return 0, nil
	}

	// Sort names for deterministic output
	var names []string
	for name := range repos {
		names = append(names, name)
	}
	sort.Strings(names)

	output.F.Info(fmt.Sprintf("No repos registered — found repos.json with %d repos, syncing...", len(names)))

	added := 0
	for _, name := range names {
		repo := repos[name]
		body := map[string]any{
			"name":   name,
			"url":    repo.URL,
			"source": detectSourceFromURL(repo.URL),
		}
		var result any
		if err := client.Post(ctx(), "/repos", body, &result); err != nil {
			output.F.Warning(fmt.Sprintf("  failed to add %s: %v", name, err))
			continue
		}
		output.F.Printf("    + %s\n", name)
		added++
	}

	if added > 0 {
		output.Successf("Synced %d repos from repos.json", added)
	}
	return added, nil
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
