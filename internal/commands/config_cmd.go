package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "config",
		Short: "Manage CLI configuration",
	}
	cmd.AddCommand(newConfigInitCmd())
	cmd.AddCommand(newConfigShowCmd())
	cmd.AddCommand(newConfigSetCmd())
	return cmd
}

func newConfigInitCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "init",
		Short: "Interactive setup wizard",
		Long:  "Set up API URL and token interactively. Tests the connection before saving.",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg := config.DefaultConfig()
			reader := bufio.NewReader(os.Stdin)

			fmt.Printf("\n%s\n\n", output.Bold("RepoSwarm CLI Setup"))

			fmt.Printf("  API URL [%s]: ", cfg.APIUrl)
			if line, _ := reader.ReadString('\n'); strings.TrimSpace(line) != "" {
				cfg.APIUrl = strings.TrimSpace(line)
			}

			fmt.Print("  API Token: ")
			if line, _ := reader.ReadString('\n'); strings.TrimSpace(line) != "" {
				cfg.APIToken = strings.TrimSpace(line)
			}

			if cfg.APIToken == "" {
				return fmt.Errorf("API token is required")
			}

			// Test connection
			output.Infof("Testing connection to %s...", cfg.APIUrl)
			client := api.New(cfg.APIUrl, cfg.APIToken)
			health, err := client.Health(ctx())
			if err != nil {
				return fmt.Errorf("connection test failed: %w", err)
			}

			output.Successf("Connected to RepoSwarm API %s (%s)", health.Version, health.Status)

			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			path, _ := config.ConfigPath()
			output.Successf("Config saved to %s", path)
			fmt.Println()
			return nil
		},
	}
}

func newConfigShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show",
		Short: "Display current configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if flagJSON {
				display := map[string]any{
					"apiUrl":       cfg.APIUrl,
					"apiToken":     config.MaskedToken(cfg.APIToken),
					"region":       cfg.Region,
					"defaultModel": cfg.DefaultModel,
					"chunkSize":    cfg.ChunkSize,
					"outputFormat": cfg.OutputFormat,
				}
				return output.JSON(display)
			}

			fmt.Printf("\n%s\n\n", output.Bold("RepoSwarm CLI Configuration"))
			fmt.Printf("  %s  %s\n", output.Dim("apiUrl       "), cfg.APIUrl)
			fmt.Printf("  %s  %s\n", output.Dim("apiToken     "), config.MaskedToken(cfg.APIToken))
			fmt.Printf("  %s  %s\n", output.Dim("region       "), cfg.Region)
			fmt.Printf("  %s  %s\n", output.Dim("defaultModel "), cfg.DefaultModel)
			fmt.Printf("  %s  %d\n", output.Dim("chunkSize    "), cfg.ChunkSize)
			fmt.Printf("  %s  %s\n", output.Dim("outputFormat "), cfg.OutputFormat)
			fmt.Println()
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			if err := config.Set(cfg, args[0], args[1]); err != nil {
				return err
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			output.Successf("Set %s = %s", args[0], args[1])
			return nil
		},
	}
}
