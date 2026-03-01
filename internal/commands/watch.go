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
  reposwarm workflows watch                              # All running
  reposwarm workflows watch investigate-single-my-repo   # Specific workflow
  reposwarm workflows watch --interval 10                # Poll every 10s`,
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
	F := output.F
	F.Info(fmt.Sprintf("Watching %s (Ctrl+C to stop)", workflowID))
	F.Println()

	lastStatus := ""
	for {
		var wf api.WorkflowExecution
		if err := client.Get(ctx(), "/workflows/"+workflowID, &wf); err != nil {
			F.Error(fmt.Sprintf("Poll failed: %s", err))
			time.Sleep(time.Duration(interval) * time.Second)
			continue
		}

		if wf.Status != lastStatus {
			ts := time.Now().Format("15:04:05")
			F.Printf("  %s  %s -> %s\n", ts, wf.Type, F.StatusText(wf.Status))
			lastStatus = wf.Status
		}

		lower := strings.ToLower(wf.Status)
		if lower == "completed" || lower == "failed" || lower == "terminated" || lower == "timed_out" || lower == "cancelled" {
			F.Println()
			F.Success(fmt.Sprintf("Workflow finished: %s", wf.Status))
			return nil
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}

func watchAll(client *api.Client, interval int) error {
	F := output.F
	F.Info("Watching running workflows (Ctrl+C to stop)")
	F.Println()

	for {
		var result api.WorkflowsResponse
		if err := client.Get(ctx(), "/workflows?pageSize=50", &result); err != nil {
			F.Error(fmt.Sprintf("Poll failed: %s", err))
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
			F.Printf("  %s  No running workflows\n", ts)
		} else {
			F.Printf("  %s  %d running:\n", ts, len(running))
			for _, w := range running {
				id := w.WorkflowID
				if len(id) > 60 {
					id = id[:57] + "..."
				}
				F.Printf("           - %s\n", id)
			}
		}

		time.Sleep(time.Duration(interval) * time.Second)
	}
}
