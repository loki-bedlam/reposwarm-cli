package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWatchCmd() *cobra.Command {
	var interval int

	cmd := &cobra.Command{
		Use:   "watch [workflow-id]",
		Short: "Watch workflow status in real-time",
		Long: `Poll workflow status and display updates until completion.

Without workflow-id: shows all running workflows.
With workflow-id: watches a specific workflow until it finishes.

Examples:
  reposwarm watch                              # All running
  reposwarm watch investigate-single-my-repo   # Specific workflow
  reposwarm watch --interval 10                # Poll every 10s`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			if len(args) > 0 {
				return watchSingle(client, args[0], interval)
			}
			return watchAll(client, interval)
		},
	}

	cmd.Flags().IntVar(&interval, "interval", 5, "Poll interval in seconds")
	return cmd
}

func watchSingle(client *api.Client, workflowID string, interval int) error {
	fmt.Printf("\n  %s %s (Ctrl+C to stop)\n\n", output.Bold("Watching"), output.Cyan(workflowID))

	lastStatus := ""
	for {
		var wf api.WorkflowExecution
		if err := client.Get(ctx(), "/workflows/"+workflowID, &wf); err != nil {
			output.Errorf("Poll failed: %s", err)
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}

		if wf.Status != lastStatus {
			ts := time.Now().Format("15:04:05")
			fmt.Printf("  %s  %s → %s\n", output.Dim(ts), wf.Type, output.StatusColor(wf.Status))
			lastStatus = wf.Status
		}

		lower := strings.ToLower(wf.Status)
		if lower == "completed" || lower == "failed" || lower == "terminated" || lower == "timed_out" || lower == "cancelled" {
			fmt.Printf("\n  %s Workflow finished: %s\n\n", output.Bold("✓"), output.StatusColor(wf.Status))
			return nil
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func watchAll(client *api.Client, interval int) error {
	fmt.Printf("\n  %s (Ctrl+C to stop)\n\n", output.Bold("Watching running workflows"))

	for {
		var result api.WorkflowsResponse
		if err := client.Get(ctx(), "/workflows?pageSize=50", &result); err != nil {
			output.Errorf("Poll failed: %s", err)
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}

		var running []api.WorkflowExecution
		for _, w := range result.Executions {
			if strings.EqualFold(w.Status, "Running") {
				running = append(running, w)
			}
		}

		ts := time.Now().Format("15:04:05")
		if len(running) == 0 {
			fmt.Printf("  %s  No running workflows\n", output.Dim(ts))
		} else {
			fmt.Printf("  %s  %d running:\n", output.Dim(ts), len(running))
			for _, w := range running {
				id := w.WorkflowID
				if len(id) > 60 {
					id = id[:57] + "..."
				}
				fmt.Printf("           %s %s\n", output.Yellow("▸"), id)
			}
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}
