package commands

import (
	"fmt"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWorkersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workers",
		Aliases: []string{"worker"},
		Short:   "Manage and inspect workers",
	}
	cmd.AddCommand(newWorkersListCmd())
	cmd.AddCommand(newWorkersShowCmd())
	return cmd
}

func newWorkersListCmd() *cobra.Command {
	var verbose bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all workers with health and activity status",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var resp api.WorkersResponse
			if err := client.Get(ctx(), "/workers", &resp); err != nil {
				return fmt.Errorf("failed to list workers: %w", err)
			}

			if flagJSON {
				return output.JSON(resp)
			}

			F := output.F
			F.Section(fmt.Sprintf("Workers (%d configured, %d healthy)", resp.Total, resp.Healthy))

			if len(resp.Workers) == 0 {
				F.Warning("No workers detected")
				F.Info("Workers register with Temporal when they start.")
				F.Info("Check: reposwarm logs worker")
				return nil
			}

			headers := []string{"Name", "Status", "Queue", "Current Task", "Last Activity", "Env"}
			if verbose {
				headers = append(headers, "PID", "Host")
			}
			var rows [][]string
			for _, w := range resp.Workers {
				statusStr := formatWorkerStatus(w.Status)
				envStr := output.Green("OK")
				if len(w.EnvErrors) > 0 {
					envStr = output.Red(fmt.Sprintf("%d env errors", len(w.EnvErrors)))
				}
				currentTask := w.CurrentTask
				if currentTask == "" {
					currentTask = output.Dim("idle")
				}
				lastAct := w.LastActivity
				if lastAct == "" {
					lastAct = output.Dim("never")
				}

				row := []string{w.Name, statusStr, w.TaskQueue, currentTask, lastAct, envStr}
				if verbose {
					pid := "—"
					if w.PID > 0 {
						pid = fmt.Sprint(w.PID)
					}
					host := w.Host
					if host == "" {
						host = "—"
					}
					row = append(row, pid, host)
				}
				rows = append(rows, row)
			}

			output.Table(headers, rows)
			F.Println()
			return nil
		},
	}

	cmd.Flags().BoolVarP(&verbose, "verbose", "v", false, "Include PID, uptime, host")
	return cmd
}

func newWorkersShowCmd() *cobra.Command {
	var logLines int
	var noLogs bool

	cmd := &cobra.Command{
		Use:   "show <name>",
		Short: "Deep-dive on a single worker: env, logs, current task",
		Args:  friendlyExactArgs(1, "reposwarm workers show <name>\n\nExample:\n  reposwarm workers show worker-1"),
		RunE: func(cmd *cobra.Command, args []string) error {
			targetName := args[0]
			client, err := getClient()
			if err != nil {
				return err
			}

			// Get worker detail
			var worker api.WorkerInfo
			if err := client.Get(ctx(), "/workers/"+targetName, &worker); err != nil {
				return fmt.Errorf("worker '%s' not found: %w", targetName, err)
			}

			if flagJSON {
				result := map[string]any{"worker": worker}
				if !noLogs {
					var logResp struct {
						Lines []string `json:"lines"`
					}
					if err := client.Get(ctx(), fmt.Sprintf("/services/worker/logs?lines=%d", logLines), &logResp); err == nil {
						result["logs"] = logResp.Lines
					}
				}
				return output.JSON(result)
			}

			F := output.F
			F.Section(fmt.Sprintf("Worker: %s", worker.Name))
			F.KeyValue("Status", formatWorkerStatus(worker.Status))
			F.KeyValue("Identity", worker.Identity)
			if worker.PID > 0 {
				F.KeyValue("PID", fmt.Sprint(worker.PID))
			}
			F.KeyValue("Task Queue", worker.TaskQueue)
			F.KeyValue("Host", orDash(worker.Host))
			F.KeyValue("Current Task", orDash(worker.CurrentTask))
			F.KeyValue("Last Activity", orDash(worker.LastActivity))
			if worker.Model != "" {
				F.KeyValue("Model", worker.Model)
			}

			// Environment from API
			F.Println()
			F.Section("Environment")
			var envResp struct {
				Entries []struct {
					Key    string `json:"key"`
					Value  string `json:"value"`
					Source string `json:"source"`
					Set    bool   `json:"set"`
				} `json:"entries"`
			}
			if err := client.Get(ctx(), "/workers/"+targetName+"/env", &envResp); err == nil {
				for _, e := range envResp.Entries {
					if !e.Set && !strings.Contains(e.Key, "ANTHROPIC") && !strings.Contains(e.Key, "GITHUB") && !strings.Contains(e.Key, "AWS_ACCESS") {
						continue // Only show important unset vars
					}
					if e.Set {
						F.Printf("  %s %s: set (%s)\n", output.Green("[OK]"), e.Key, e.Source)
					} else {
						F.Printf("  %s %s: NOT SET\n", output.Red("[FAIL]"), e.Key)
					}
				}
			}

			// Logs from API
			if !noLogs {
				F.Println()
				var logResp struct {
					Lines []string `json:"lines"`
				}
				if err := client.Get(ctx(), fmt.Sprintf("/services/worker/logs?lines=%d", logLines), &logResp); err == nil && len(logResp.Lines) > 0 {
					F.Section(fmt.Sprintf("Recent Logs (last %d lines)", logLines))
					for _, l := range logResp.Lines {
						F.Printf("  %s\n", l)
					}
				} else {
					F.Section("Logs")
					F.Info("No worker log file found")
				}
			}

			return nil
		},
	}

	cmd.Flags().IntVar(&logLines, "logs", 10, "Number of log tail lines to include")
	cmd.Flags().BoolVar(&noLogs, "no-logs", false, "Skip log section")
	return cmd
}

// gatherWorkerInfo fetches worker info from the API.
func gatherWorkerInfo(client *api.Client) []api.WorkerInfo {
	var resp api.WorkersResponse
	if err := client.Get(ctx(), "/workers", &resp); err != nil {
		return nil
	}
	return resp.Workers
}

func formatWorkerStatus(status string) string {
	switch status {
	case "healthy":
		return output.Green("✅ healthy")
	case "degraded":
		return output.Yellow("⚠ degraded")
	case "failed":
		return output.Red("❌ failed")
	case "stopped":
		return output.Dim("⏹ stopped")
	default:
		return status
	}
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}
