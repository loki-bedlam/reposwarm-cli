package commands

import (
	"fmt"
	"time"

	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Check API health and connection",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			start := time.Now()
			health, err := client.Health(ctx())
			latency := time.Since(start)

			if err != nil {
				if flagJSON {
					return output.JSON(map[string]any{
						"connected": false,
						"error":     err.Error(),
					})
				}
				output.Errorf("Connection failed: %s", err)
				return nil
			}

			cfg, _ := config.Load()

			if flagJSON {
				return output.JSON(map[string]any{
					"connected": true,
					"status":    health.Status,
					"version":   health.Version,
					"latency":   latency.Milliseconds(),
					"temporal":  health.Temporal.Connected,
					"dynamodb":  health.DynamoDB.Connected,
					"worker":    health.Worker.Connected,
					"apiUrl":    cfg.APIUrl,
				})
			}

			fmt.Printf("\n  %s\n\n", output.Bold("RepoSwarm Status"))
			fmt.Printf("  %s  %s\n", output.Dim("API URL    "), cfg.APIUrl)
			fmt.Printf("  %s  %s\n", output.Dim("Status     "), output.Green(health.Status))
			fmt.Printf("  %s  %s\n", output.Dim("Version    "), health.Version)
			fmt.Printf("  %s  %dms\n", output.Dim("Latency    "), latency.Milliseconds())

			fmt.Printf("\n  %s\n", output.Bold("Services:"))
			svc := func(name string, connected bool) {
				icon := output.Green("✓")
				if !connected {
					icon = output.Red("✗")
				}
				fmt.Printf("    %s %s\n", icon, name)
			}
			svc("Temporal", health.Temporal.Connected)
			svc("DynamoDB", health.DynamoDB.Connected)
			svc("Worker", health.Worker.Connected)

			if health.Temporal.Connected {
				fmt.Printf("      %s  %s\n", output.Dim("namespace"), health.Temporal.Namespace)
				fmt.Printf("      %s  %s\n", output.Dim("taskQueue"), health.Temporal.TaskQueue)
			}
			if health.Worker.Connected {
				fmt.Printf("      %s  %d\n", output.Dim("workers  "), health.Worker.Count)
			}
			fmt.Println()
			return nil
		},
	}
}
