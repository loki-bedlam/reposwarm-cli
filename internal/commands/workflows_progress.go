package commands

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWorkflowsProgressCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "progress",
		Short: "Show progress of the active daily investigation",
		Long: `Shows a summary of the currently running daily investigation workflow,
including completed, in-progress, and pending repositories.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			// Fetch all workflows (up to 100)
			var result api.WorkflowsResponse
			if err := client.Get(ctx(), "/workflows?pageSize=100", &result); err != nil {
				return err
			}

			// Find the active daily workflow
			var daily *api.WorkflowExecution
			for i, w := range result.Executions {
				if w.Type == "InvestigateReposWorkflow" && w.Status == "Running" {
					daily = &result.Executions[i]
					break
				}
			}

			if daily == nil {
				if flagJSON {
					return output.JSON(map[string]any{"error": "no active daily workflow"})
				}
				output.Infof("No active daily investigation workflow found")
				return nil
			}

			// Collect child workflows started after the daily
			var children []api.WorkflowExecution
			for _, w := range result.Executions {
				if w.Type != "InvestigateSingleRepoWorkflow" {
					continue
				}
				if w.StartTime >= daily.StartTime {
					children = append(children, w)
				}
			}

			// Categorize
			var running, completed, failed []api.WorkflowExecution
			for _, w := range children {
				switch w.Status {
				case "Running":
					running = append(running, w)
				case "Completed":
					completed = append(completed, w)
				case "Failed":
					failed = append(failed, w)
				}
			}

			sort.Slice(completed, func(i, j int) bool {
				return completed[i].CloseTime < completed[j].CloseTime
			})
			sort.Slice(running, func(i, j int) bool {
				return running[i].WorkflowID < running[j].WorkflowID
			})

			// Count total repos from repo list
			var repos []api.Repository
			totalRepos := 36 // fallback
			if err := client.Get(ctx(), "/repos", &repos); err == nil {
				enabled := 0
				for _, r := range repos {
					if r.Enabled {
						enabled++
					}
				}
				if enabled > 0 {
					totalRepos = enabled
				}
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"dailyWorkflowId": daily.WorkflowID,
					"startTime":       daily.StartTime,
					"totalRepos":      totalRepos,
					"completed":       len(completed),
					"running":         len(running),
					"failed":          len(failed),
					"pending":         totalRepos - len(children),
					"completedRepos":  repoNames(completed),
					"runningRepos":    repoNames(running),
					"failedRepos":     repoNames(failed),
				})
			}

			// Pretty output
			fmt.Println()
			fmt.Printf("  %s\n", output.Bold("📊 Daily Investigation Progress"))
			fmt.Printf("  %s  %s\n", output.Dim("Workflow"), daily.WorkflowID)
			fmt.Printf("  %s  %s\n", output.Dim("Started "), daily.StartTime[:19])
			fmt.Printf("  %s  %s\n", output.Dim("Elapsed "), elapsed(daily.StartTime))
			fmt.Println()

			pending := totalRepos - len(children)
			pct := 0
			if totalRepos > 0 {
				pct = len(completed) * 100 / totalRepos
			}

			// Progress bar
			barWidth := 30
			filled := barWidth * len(completed) / totalRepos
			bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
			fmt.Printf("  %s %d%% (%d/%d)\n", bar, pct, len(completed), totalRepos)
			fmt.Println()

			fmt.Printf("  %s %-3d  %s %-3d  %s %-3d  %s %-3d\n",
				output.Green("✅"), len(completed),
				"🔄", len(running),
				output.Error("❌"), len(failed),
				output.Dim("⏳"), pending,
			)
			fmt.Println()

			if len(completed) > 0 {
				fmt.Printf("  %s\n", output.Dim("── Completed ──"))
				for _, w := range completed {
					fmt.Printf("  ✅ %-35s %s\n", repoName(w.WorkflowID), duration(w))
				}
				fmt.Println()
			}

			if len(running) > 0 {
				fmt.Printf("  %s\n", output.Dim("── In Progress ──"))
				for _, w := range running {
					fmt.Printf("  🔄 %-35s %s elapsed\n", repoName(w.WorkflowID), elapsed(w.StartTime))
				}
				fmt.Println()
			}

			if len(failed) > 0 {
				fmt.Printf("  %s\n", output.Dim("── Failed ──"))
				for _, w := range failed {
					fmt.Printf("  ❌ %-35s %s\n", repoName(w.WorkflowID), duration(w))
				}
				fmt.Println()
			}

			if pending > 0 {
				fmt.Printf("  %s %d repos waiting to start\n", output.Dim("⏳"), pending)
				fmt.Println()
			}

			return nil
		},
	}

	return cmd
}

func repoName(workflowID string) string {
	return strings.TrimPrefix(workflowID, "investigate-single-repo-")
}

func repoNames(wfs []api.WorkflowExecution) []string {
	names := make([]string, len(wfs))
	for i, w := range wfs {
		names[i] = repoName(w.WorkflowID)
	}
	return names
}

func elapsed(startTime string) string {
	t, err := time.Parse(time.RFC3339Nano, startTime)
	if err != nil {
		// Try without nanoseconds
		t, err = time.Parse("2006-01-02T15:04:05Z", startTime)
		if err != nil {
			return "?"
		}
	}
	d := time.Since(t)
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}

func duration(w api.WorkflowExecution) string {
	if w.CloseTime == "" {
		return elapsed(w.StartTime)
	}
	start, err1 := time.Parse(time.RFC3339Nano, w.StartTime)
	end, err2 := time.Parse(time.RFC3339Nano, w.CloseTime)
	if err1 != nil || err2 != nil {
		return "?"
	}
	d := end.Sub(start)
	return fmt.Sprintf("%dm%02ds", int(d.Minutes()), int(d.Seconds())%60)
}
