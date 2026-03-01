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
				output.F.Error(fmt.Sprintf("Connection failed: %s", err))
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

			F := output.F
			F.Section("RepoSwarm Status")
			F.KeyValue("API URL", cfg.APIUrl)
			F.KeyValue("Status", health.Status)
			F.KeyValue("Version", health.Version)
			F.KeyValue("Latency", fmt.Sprintf("%dms", latency.Milliseconds()))

			svcStatus := func(name string, connected bool) string {
				if connected {
					return "ok"
				}
				return "DISCONNECTED"
			}

			F.Println()
			F.KeyValue("Temporal", svcStatus("Temporal", health.Temporal.Connected))
			F.KeyValue("DynamoDB", svcStatus("DynamoDB", health.DynamoDB.Connected))
			F.KeyValue("Worker", svcStatus("Worker", health.Worker.Connected))

			if health.Temporal.Connected {
				F.KeyValue("  namespace", health.Temporal.Namespace)
				F.KeyValue("  taskQueue", health.Temporal.TaskQueue)
			}
			if health.Worker.Connected {
				F.KeyValue("  workers", fmt.Sprint(health.Worker.Count))
			}
			F.Println()
			return nil
		},
	}
}
