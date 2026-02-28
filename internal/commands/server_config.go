package commands

import (
	"fmt"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newServerConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server-config",
		Short: "View or update server-side configuration",
	}
	cmd.AddCommand(newServerConfigShowCmd())
	cmd.AddCommand(newServerConfigSetCmd())
	return cmd
}

func newServerConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Show server configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var cfg api.ConfigResponse
			if err := client.Get(ctx(), "/config", &cfg); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(cfg)
			}

			fmt.Printf("\n  %s\n\n", output.Bold("Server Configuration"))
			fmt.Printf("  %s  %s\n", output.Dim("defaultModel      "), cfg.DefaultModel)
			fmt.Printf("  %s  %d\n", output.Dim("chunkSize         "), cfg.ChunkSize)
			fmt.Printf("  %s  %dms\n", output.Dim("sleepDuration     "), cfg.SleepDuration)
			fmt.Printf("  %s  %d\n", output.Dim("parallelLimit     "), cfg.ParallelLimit)
			fmt.Printf("  %s  %d\n", output.Dim("tokenLimit        "), cfg.TokenLimit)
			fmt.Printf("  %s  %s\n", output.Dim("scheduleExpression"), cfg.ScheduleExpression)
			fmt.Println()
			return nil
		},
	}
}

func newServerConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Update a server configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			body := map[string]any{args[0]: args[1]}
			var result any
			if err := client.Patch(ctx(), "/config", body, &result); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(map[string]any{"key": args[0], "value": args[1]})
			}
			output.Successf("Set server %s = %s", args[0], args[1])
			return nil
		},
	}
}
