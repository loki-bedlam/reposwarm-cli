package commands

import (
	"fmt"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newConfigModelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "model",
		Short: "Configure investigation model",
	}
	cmd.AddCommand(newModelSetCmd())
	cmd.AddCommand(newModelShowCmd())
	cmd.AddCommand(newModelListCmd())
	cmd.AddCommand(newModelPinCmd())
	return cmd
}

func newModelSetCmd() *cobra.Command {
	var syncWorker bool

	cmd := &cobra.Command{
		Use:   "set <alias|model-id>",
		Short: "Set the default investigation model",
		Long: `Set the model used for investigations. Accepts aliases or full model IDs.

Aliases: sonnet, opus, haiku, sonnet-3.5
These resolve to the correct model ID based on your configured provider.

Examples:
  reposwarm config model set opus
  reposwarm config model set us.anthropic.claude-opus-4-6-v1
  reposwarm config model set sonnet --sync`,
		Args: friendlyExactArgs(1, "reposwarm config model set <alias|model-id>\n\nAliases: sonnet, opus, haiku\n\nExample:\n  reposwarm config model set opus"),
		RunE: func(cmd *cobra.Command, args []string) error {
			input := args[0]
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			provider := cfg.EffectiveProvider()
			resolved := config.ResolveModel(input, provider, cfg.ProviderConfig.ModelPins)
			cfg.DefaultModel = resolved

			if err := config.Save(cfg); err != nil {
				return err
			}

			// Sync to worker env
			if syncWorker {
				client, err := getClient()
				if err == nil {
					body := map[string]string{"value": resolved}
					var resp any
					client.Put(ctx(), "/workers/worker-1/env/ANTHROPIC_MODEL", body, &resp)
				}
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"input":    input,
					"resolved": resolved,
					"provider": string(provider),
					"synced":   syncWorker,
				})
			}

			if input != resolved {
				output.Successf("Model set: %s → %s (provider: %s)", input, resolved, provider)
			} else {
				output.Successf("Model set: %s (provider: %s)", resolved, provider)
			}
			if syncWorker {
				output.F.Info("Worker env synced. Restart to apply: reposwarm restart worker")
			} else {
				output.F.Info(fmt.Sprintf("To sync to worker: reposwarm config model set %s --sync", input))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&syncWorker, "sync", false, "Also update worker env var")
	return cmd
}

func newModelShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show configured model across CLI, server, and worker",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			provider := cfg.EffectiveProvider()
			cliModel := cfg.EffectiveModel()

			// Get server model
			var serverModel string
			client, clientErr := getClient()
			if clientErr == nil {
				var serverCfg struct {
					DefaultModel string `json:"defaultModel"`
				}
				if err := client.Get(ctx(), "/config", &serverCfg); err == nil {
					serverModel = serverCfg.DefaultModel
				}
			}

			// Get worker model from env
			var workerModel string
			if clientErr == nil {
				var envResp struct {
					Entries []struct {
						Key   string `json:"key"`
						Value string `json:"value"`
						Set   bool   `json:"set"`
					} `json:"entries"`
				}
				if err := client.Get(ctx(), "/workers/worker-1/env?reveal=true", &envResp); err == nil {
					for _, e := range envResp.Entries {
						if e.Key == "ANTHROPIC_MODEL" && e.Set {
							workerModel = e.Value
						}
					}
				}
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"provider":    string(provider),
					"cliModel":    cliModel,
					"serverModel": serverModel,
					"workerModel": workerModel,
					"pins":        cfg.ProviderConfig.ModelPins,
					"smallModel":  cfg.ProviderConfig.SmallModel,
				})
			}

			F := output.F
			F.Section("Model Configuration")
			F.KeyValue("Provider", string(provider))
			F.KeyValue("CLI Model", cliModel)

			if serverModel != "" {
				F.KeyValue("Server Model", serverModel)
			} else {
				F.KeyValue("Server Model", output.Dim("(unavailable)"))
			}

			if workerModel != "" {
				F.KeyValue("Worker Model", workerModel)
			} else {
				F.KeyValue("Worker Model", output.Dim("(unavailable)"))
			}

			// Drift detection
			if serverModel != "" && serverModel != cliModel {
				F.Println()
				F.Warning(fmt.Sprintf("⚠ CLI (%s) ≠ Server (%s)", cliModel, serverModel))
			}
			if workerModel != "" && workerModel != cliModel {
				F.Println()
				F.Warning(fmt.Sprintf("⚠ CLI (%s) ≠ Worker (%s)", cliModel, workerModel))
				F.Info(fmt.Sprintf("Sync: reposwarm config model set %s --sync", reverseResolve(cliModel, provider)))
			}

			if len(cfg.ProviderConfig.ModelPins) > 0 {
				F.Println()
				F.Section("Pinned Versions")
				for alias, id := range cfg.ProviderConfig.ModelPins {
					F.KeyValue(alias, id)
				}
			}

			return nil
		},
	}
	return cmd
}

func newModelListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List available model aliases and their resolved IDs",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			provider := cfg.EffectiveProvider()
			aliases := config.KnownAliases()

			if flagJSON {
				var models []map[string]string
				for _, a := range aliases {
					resolved := config.ResolveModel(a.Alias, provider, cfg.ProviderConfig.ModelPins)
					m := map[string]string{
						"alias":     a.Alias,
						"resolved":  resolved,
						"anthropic": a.Anthropic,
						"bedrock":   a.Bedrock,
					}
					if pin, ok := cfg.ProviderConfig.ModelPins[a.Alias]; ok {
						m["pinned"] = pin
					}
					models = append(models, m)
				}
				return output.JSON(map[string]any{
					"provider": string(provider),
					"models":   models,
				})
			}

			F := output.F
			F.Section(fmt.Sprintf("Available Models (provider: %s)", provider))
			F.Println()

			headers := []string{"Alias", "Resolved ID", "Pinned?"}
			var rows [][]string
			for _, a := range aliases {
				resolved := config.ResolveModel(a.Alias, provider, cfg.ProviderConfig.ModelPins)
				pinned := "—"
				if pin, ok := cfg.ProviderConfig.ModelPins[a.Alias]; ok {
					pinned = output.Green("✓ " + pin)
					if pin != resolved {
						pinned = output.Yellow("✓ " + pin + " (overrides)")
					}
				}

				current := ""
				if resolved == cfg.EffectiveModel() {
					current = " " + output.Green("← current")
				}

				rows = append(rows, []string{a.Alias, resolved + current, pinned})
			}

			output.Table(headers, rows)
			fmt.Println()
			F.Info("Set model: reposwarm config model set <alias>")
			F.Info("Pin all:   reposwarm config model pin")
			return nil
		},
	}
	return cmd
}

func newModelPinCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "pin",
		Short: "Pin all model aliases to current versions (prevents version drift)",
		Long: `Pin model aliases to specific version IDs.

Without pinning, aliases like "sonnet" resolve to the latest version.
When Anthropic releases new versions, your investigations may silently
change behavior. Pinning locks each alias to a specific version.

This is especially important for Bedrock, where model IDs include
version suffixes.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			provider := cfg.EffectiveProvider()
			cfg.ProviderConfig.ModelPins = map[string]string{}

			for _, a := range config.KnownAliases() {
				switch provider {
				case config.ProviderBedrock:
					cfg.ProviderConfig.ModelPins[a.Alias] = a.Bedrock
				default:
					cfg.ProviderConfig.ModelPins[a.Alias] = a.Anthropic
				}
			}

			if err := config.Save(cfg); err != nil {
				return err
			}

			// Sync pins to worker
			client, clientErr := getClient()
			if clientErr == nil {
				workerVars := config.WorkerEnvVars(&cfg.ProviderConfig, cfg.DefaultModel)
				for k, v := range workerVars {
					body := map[string]string{"value": v}
					var resp any
					client.Put(ctx(), "/workers/worker-1/env/"+k, body, &resp)
				}
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"provider": string(provider),
					"pins":     cfg.ProviderConfig.ModelPins,
				})
			}

			output.Successf("All model aliases pinned for %s:", provider)
			for alias, id := range cfg.ProviderConfig.ModelPins {
				output.F.KeyValue(alias, id)
			}
			if clientErr == nil {
				fmt.Println()
				output.F.Warning("Restart worker to apply: reposwarm restart worker")
			}
			return nil
		},
	}
	return cmd
}

// reverseResolve attempts to find the alias for a model ID.
func reverseResolve(modelID string, provider config.Provider) string {
	for _, a := range config.KnownAliases() {
		switch provider {
		case config.ProviderBedrock:
			if a.Bedrock == modelID {
				return a.Alias
			}
		default:
			if a.Anthropic == modelID {
				return a.Alias
			}
		}
	}
	return modelID
}

// isAlias returns true if the input is a known model alias.
func isAlias(input string) bool {
	for _, a := range config.KnownAliases() {
		if a.Alias == input {
			return true
		}
	}
	return false
}

func formatProviderList() string {
	return strings.Join(config.ValidProviders(), ", ")
}
