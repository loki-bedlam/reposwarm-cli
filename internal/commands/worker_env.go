package commands

import (
	"fmt"

	"github.com/reposwarm/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newConfigWorkerEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "worker-env",
		Aliases: []string{"env"},
		Short:   "Manage worker environment variables",
	}
	cmd.AddCommand(newWorkerEnvListCmd())
	cmd.AddCommand(newWorkerEnvSetCmd())
	cmd.AddCommand(newWorkerEnvUnsetCmd())
	return cmd
}

func newWorkerEnvListCmd() *cobra.Command {
	var reveal bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show all worker environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			path := "/workers/worker-1/env"
			if reveal {
				path += "?reveal=true"
			}

			var resp struct {
				EnvFile string `json:"envFile"`
				Entries []struct {
					Key    string `json:"key"`
					Value  string `json:"value"`
					Source string `json:"source"`
					Set    bool   `json:"set"`
				} `json:"entries"`
			}
			if err := client.Get(ctx(), path, &resp); err != nil {
				return fmt.Errorf("failed to list worker env: %w", err)
			}

			if flagJSON {
				return output.JSON(resp)
			}

			F := output.F
			F.Section("Worker Environment")
			F.KeyValue("Env file", resp.EnvFile)
			F.Println()

			headers := []string{"Variable", "Value", "Source"}
			var rows [][]string
			for _, e := range resp.Entries {
				valStr := e.Value
				if !e.Set {
					valStr = output.Dim("(not set)")
				}
				rows = append(rows, []string{e.Key, valStr, e.Source})
			}
			output.Table(headers, rows)
			F.Println()
			return nil
		},
	}

	cmd.Flags().BoolVar(&reveal, "reveal", false, "Show full unmasked values")
	return cmd
}

func newWorkerEnvSetCmd() *cobra.Command {
	var restart bool

	cmd := &cobra.Command{
		Use:   "set <KEY> <VALUE>",
		Short: "Set a worker environment variable",
		Args:  friendlyExactArgs(2, "reposwarm config worker-env set <KEY> <VALUE>\n\nExample:\n  reposwarm config worker-env set ANTHROPIC_API_KEY sk-ant-abc123"),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			client, err := getClient()
			if err != nil {
				return err
			}

			body := map[string]string{"value": value}
			var resp struct {
				Key     string `json:"key"`
				Value   string `json:"value"`
				EnvFile string `json:"envFile"`
			}
			if err := client.Put(ctx(), "/workers/worker-1/env/"+key, body, &resp); err != nil {
				return fmt.Errorf("failed to set %s: %w", key, err)
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"key":     resp.Key,
					"value":   resp.Value,
					"envFile": resp.EnvFile,
					"restart": restart,
				})
			}

			output.Successf("Set %s = %s (written to %s)", key, resp.Value, resp.EnvFile)

			if restart {
				output.F.Info("Restarting worker...")
				var restartResp any
				if err := client.Post(ctx(), "/workers/worker-1/restart", nil, &restartResp); err != nil {
					output.F.Warning(fmt.Sprintf("Could not restart: %v", err))
					output.F.Info("Restart manually: reposwarm restart worker")
				} else {
					output.Successf("Worker restarted")
				}
			} else {
				output.F.Warning("Worker restart required. Run: reposwarm restart worker")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&restart, "restart", false, "Automatically restart the worker after setting")
	return cmd
}

func newWorkerEnvUnsetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset <KEY>",
		Short: "Remove a worker environment variable",
		Args:  friendlyExactArgs(1, "reposwarm config worker-env unset <KEY>\n\nExample:\n  reposwarm config worker-env unset SOME_VAR"),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			client, err := getClient()
			if err != nil {
				return err
			}

			var resp any
			if err := client.Delete(ctx(), "/workers/worker-1/env/"+key, &resp); err != nil {
				return fmt.Errorf("failed to unset %s: %w", key, err)
			}

			if flagJSON {
				return output.JSON(map[string]any{"key": key, "removed": true})
			}

			output.Successf("Removed %s", key)
			output.F.Warning("Worker restart required. Run: reposwarm restart worker")
			return nil
		},
	}
	return cmd
}
