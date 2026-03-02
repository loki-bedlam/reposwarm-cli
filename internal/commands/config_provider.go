package commands

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newConfigProviderCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "provider",
		Short: "Configure LLM provider (Anthropic, Bedrock, LiteLLM)",
	}
	cmd.AddCommand(newProviderSetupCmd())
	cmd.AddCommand(newProviderSetCmd())
	cmd.AddCommand(newProviderShowCmd())
	return cmd
}

func newProviderSetupCmd() *cobra.Command {
	var (
		providerFlag string
		regionFlag   string
		modelFlag    string
		proxyURLFlag string
		proxyKeyFlag string
		pinFlag      bool
		nonInterFlag bool
	)

	cmd := &cobra.Command{
		Use:   "setup",
		Short: "Interactive provider setup wizard",
		Long: `Set up the LLM provider for RepoSwarm investigations.

Supported providers:
  anthropic  — Direct Anthropic API (needs ANTHROPIC_API_KEY)
  bedrock    — Amazon Bedrock (needs AWS credentials)
  litellm    — LiteLLM proxy (needs proxy URL and optional key)

Interactive mode walks you through each step.
Non-interactive mode requires --provider and provider-specific flags.

Examples:
  reposwarm config provider setup
  reposwarm config provider setup --provider bedrock --region us-east-1 --model opus --pin
  reposwarm config provider setup --provider litellm --proxy-url https://my-proxy.example.com --model sonnet`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			provider := providerFlag
			region := regionFlag
			model := modelFlag
			proxyURL := proxyURLFlag
			proxyKey := proxyKeyFlag
			pin := pinFlag

			if !nonInterFlag && provider == "" {
				// Interactive mode
				reader := bufio.NewReader(os.Stdin)

				fmt.Println()
				output.F.Section("Provider Setup")
				fmt.Println()
				fmt.Println("  Which LLM provider should RepoSwarm use?")
				fmt.Println()
				fmt.Println("  1) anthropic  — Direct Anthropic API (API key)")
				fmt.Println("  2) bedrock    — Amazon Bedrock (AWS credentials)")
				fmt.Println("  3) litellm    — LiteLLM proxy (custom endpoint)")
				fmt.Println()

				provider = promptChoice(reader, "Provider [1/2/3]", map[string]string{
					"1": "anthropic", "2": "bedrock", "3": "litellm",
					"anthropic": "anthropic", "bedrock": "bedrock", "litellm": "litellm",
				}, "anthropic")

				switch config.Provider(provider) {
				case config.ProviderBedrock:
					if region == "" {
						region = promptString(reader, "AWS Region", "us-east-1")
					}
				case config.ProviderLiteLLM:
					if proxyURL == "" {
						proxyURL = promptString(reader, "LiteLLM proxy URL", "http://localhost:4000")
					}
					if proxyKey == "" {
						proxyKey = promptString(reader, "LiteLLM proxy API key (blank if none)", "")
					}
				}

				if model == "" {
					fmt.Println()
					fmt.Println("  Model aliases: sonnet, opus, haiku")
					fmt.Println("  Or specify a full model ID.")
					model = promptString(reader, "Model", "sonnet")
				}

				if !pin {
					fmt.Println()
					pinStr := promptString(reader, "Pin model versions? (recommended for stability) [y/N]", "n")
					pin = strings.ToLower(pinStr) == "y" || strings.ToLower(pinStr) == "yes"
				}
			}

			if provider == "" {
				return fmt.Errorf("--provider is required in non-interactive mode")
			}
			if !config.IsValidProvider(provider) {
				return fmt.Errorf("unknown provider: %s (valid: %s)", provider, strings.Join(config.ValidProviders(), ", "))
			}

			// Apply to config
			cfg.ProviderConfig.Provider = config.Provider(provider)

			switch config.Provider(provider) {
			case config.ProviderBedrock:
				if region == "" {
					region = "us-east-1"
				}
				cfg.ProviderConfig.AWSRegion = region
			case config.ProviderLiteLLM:
				cfg.ProviderConfig.ProxyURL = proxyURL
				cfg.ProviderConfig.ProxyKey = proxyKey
			}

			// Resolve model
			if model == "" {
				model = "sonnet"
			}
			resolved := config.ResolveModel(model, config.Provider(provider), nil)
			cfg.DefaultModel = resolved

			// Clear old pins from different provider, then optionally re-pin
			if !pin {
				cfg.ProviderConfig.ModelPins = nil
			}
			if pin {
				cfg.ProviderConfig.ModelPins = map[string]string{}
				for _, a := range config.KnownAliases() {
					switch config.Provider(provider) {
					case config.ProviderBedrock:
						cfg.ProviderConfig.ModelPins[a.Alias] = a.Bedrock
					default:
						cfg.ProviderConfig.ModelPins[a.Alias] = a.Anthropic
					}
				}
			}

			// Save config
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("saving config: %w", err)
			}

			// Push env vars to worker via API
			workerVars := config.WorkerEnvVars(&cfg.ProviderConfig, model)
			client, clientErr := getClient()
			if clientErr == nil {
				for k, v := range workerVars {
					body := map[string]string{"value": v}
					var resp any
					if err := client.Put(ctx(), "/workers/worker-1/env/"+k, body, &resp); err != nil {
						if !flagJSON {
							output.F.Warning(fmt.Sprintf("Could not set worker env %s: %v", k, err))
						}
					}
				}
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"provider":   provider,
					"model":      resolved,
					"region":     region,
					"proxyUrl":   proxyURL,
					"pinned":     pin,
					"workerVars": workerVars,
				})
			}

			fmt.Println()
			output.Successf("Provider configured: %s", provider)
			output.F.KeyValue("Model", resolved)
			if region != "" {
				output.F.KeyValue("Region", region)
			}
			if proxyURL != "" {
				output.F.KeyValue("Proxy URL", proxyURL)
			}
			if pin {
				output.F.KeyValue("Pinned", "yes")
			}
			fmt.Println()

			if clientErr == nil {
				output.Successf("Worker env vars synced (%d vars)", len(workerVars))
				output.F.Warning("Worker restart required: reposwarm restart worker")
			} else {
				output.F.Warning("Could not sync to worker API — set env vars manually")
				fmt.Println()
				for k, v := range workerVars {
					fmt.Printf("  %s=%s\n", k, v)
				}
			}

			return nil
		},
	}

	cmd.Flags().StringVar(&providerFlag, "provider", "", "Provider (anthropic|bedrock|litellm)")
	cmd.Flags().StringVar(&regionFlag, "region", "", "AWS region (Bedrock)")
	cmd.Flags().StringVar(&modelFlag, "model", "", "Model alias or ID")
	cmd.Flags().StringVar(&proxyURLFlag, "proxy-url", "", "LiteLLM proxy URL")
	cmd.Flags().StringVar(&proxyKeyFlag, "proxy-key", "", "LiteLLM proxy API key")
	cmd.Flags().BoolVar(&pinFlag, "pin", false, "Pin model versions")
	cmd.Flags().BoolVar(&nonInterFlag, "non-interactive", false, "Skip prompts")
	return cmd
}

func newProviderSetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "set <provider>",
		Short: "Quick-switch provider (preserves other settings)",
		Args:  friendlyExactArgs(1, "reposwarm config provider set <provider>\n\nProviders: anthropic, bedrock, litellm\n\nExample:\n  reposwarm config provider set bedrock"),
		RunE: func(cmd *cobra.Command, args []string) error {
			provider := args[0]
			if !config.IsValidProvider(provider) {
				return fmt.Errorf("unknown provider: %s (valid: %s)", provider, strings.Join(config.ValidProviders(), ", "))
			}

			cfg, err := config.Load()
			if err != nil {
				return err
			}

			oldProvider := cfg.EffectiveProvider()
			cfg.ProviderConfig.Provider = config.Provider(provider)

			// Re-resolve default model for new provider
			for _, a := range config.KnownAliases() {
				switch oldProvider {
				case config.ProviderBedrock:
					if cfg.DefaultModel == a.Bedrock {
						cfg.DefaultModel = config.ResolveModel(a.Alias, config.Provider(provider), cfg.ProviderConfig.ModelPins)
					}
				default:
					if cfg.DefaultModel == a.Anthropic {
						cfg.DefaultModel = config.ResolveModel(a.Alias, config.Provider(provider), cfg.ProviderConfig.ModelPins)
					}
				}
			}

			// Clear pins from old provider (they have wrong format)
			cfg.ProviderConfig.ModelPins = nil

			if err := config.Save(cfg); err != nil {
				return err
			}

			// Sync worker env
			workerVars := config.WorkerEnvVars(&cfg.ProviderConfig, cfg.DefaultModel)
			client, clientErr := getClient()
			if clientErr == nil {
				for k, v := range workerVars {
					body := map[string]string{"value": v}
					var resp any
					client.Put(ctx(), "/workers/worker-1/env/"+k, body, &resp)
				}
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"provider": provider,
					"model":    cfg.DefaultModel,
				})
			}

			output.Successf("Switched to %s (model: %s)", provider, cfg.DefaultModel)
			if clientErr == nil {
				output.F.Warning("Restart worker to apply: reposwarm restart worker")
			}
			return nil
		},
	}
	return cmd
}

func newProviderShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show",
		Short: "Show current provider configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, err := config.Load()
			if err != nil {
				return err
			}

			pc := cfg.ProviderConfig
			provider := cfg.EffectiveProvider()
			model := cfg.EffectiveModel()

			// Check worker env via API for validation
			var workerProvider string
			client, clientErr := getClient()
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
						if e.Key == "CLAUDE_CODE_USE_BEDROCK" && e.Set && e.Value == "1" {
							workerProvider = "bedrock"
						}
						if e.Key == "ANTHROPIC_BASE_URL" && e.Set {
							workerProvider = "litellm"
						}
					}
					if workerProvider == "" {
						workerProvider = "anthropic"
					}
				}
			}

			if flagJSON {
				result := map[string]any{
					"provider":    string(provider),
					"model":       model,
					"awsRegion":   pc.AWSRegion,
					"proxyUrl":    pc.ProxyURL,
					"smallModel":  pc.SmallModel,
					"modelPins":   pc.ModelPins,
				}
				if workerProvider != "" {
					result["workerProvider"] = workerProvider
				}
				return output.JSON(result)
			}

			F := output.F
			F.Section("Provider Configuration")
			F.KeyValue("Provider", string(provider))
			F.KeyValue("Model", model)

			switch provider {
			case config.ProviderBedrock:
				F.KeyValue("AWS Region", orDefault(pc.AWSRegion, "us-east-1"))
			case config.ProviderLiteLLM:
				F.KeyValue("Proxy URL", orDefault(pc.ProxyURL, "(not set)"))
				if pc.ProxyKey != "" {
					F.KeyValue("Proxy Key", config.MaskedToken(pc.ProxyKey))
				}
			}

			if pc.SmallModel != "" {
				F.KeyValue("Small Model", pc.SmallModel)
			}

			if len(pc.ModelPins) > 0 {
				F.Println()
				F.Section("Model Pins")
				for alias, id := range pc.ModelPins {
					F.KeyValue(alias, id)
				}
			}

			// Drift check
			if workerProvider != "" && workerProvider != string(provider) {
				F.Println()
				F.Warning(fmt.Sprintf("⚠ Config drift: CLI says '%s' but worker is running '%s'", provider, workerProvider))
				F.Info("Run: reposwarm restart worker")
			}

			return nil
		},
	}
	return cmd
}

// ── Prompt helpers ──

func promptString(reader *bufio.Reader, label, defaultVal string) string {
	if defaultVal != "" {
		fmt.Printf("  %s [%s]: ", label, defaultVal)
	} else {
		fmt.Printf("  %s: ", label)
	}
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return defaultVal
	}
	return input
}

func promptChoice(reader *bufio.Reader, label string, choices map[string]string, defaultVal string) string {
	for {
		fmt.Printf("  %s: ", label)
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)
		if input == "" {
			return defaultVal
		}
		if val, ok := choices[strings.ToLower(input)]; ok {
			return val
		}
		fmt.Printf("    Invalid choice. Try: %s\n", strings.Join(mapKeys(choices), ", "))
	}
}

func mapKeys(m map[string]string) []string {
	seen := map[string]bool{}
	var keys []string
	for _, v := range m {
		if !seen[v] {
			keys = append(keys, v)
			seen[v] = true
		}
	}
	return keys
}

func orDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
