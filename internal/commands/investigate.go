package commands

import (
	"fmt"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newInvestigateCmd() *cobra.Command {
	var model string
	var chunkSize, parallel int
	var all bool

	cmd := &cobra.Command{
		Use:   "investigate [repo]",
		Short: "Trigger architecture investigation",
		Long: `Trigger an AI-powered architecture investigation for one or all repos.

Examples:
  reposwarm investigate is-odd              # Single repo
  reposwarm investigate --all               # All enabled repos
  reposwarm investigate is-odd --model us.anthropic.claude-opus-4-6`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			cfg, _ := config.Load()
			if model == "" {
				model = cfg.DefaultModel
			}
			if chunkSize == 0 {
				chunkSize = cfg.ChunkSize
			}

			if len(args) > 0 {
				// Single repo
				req := api.InvestigateRequest{
					RepoName:  args[0],
					Model:     model,
					ChunkSize: chunkSize,
				}
				var result any
				if err := client.Post(ctx(), "/investigate/single", req, &result); err != nil {
					return err
				}
				if flagJSON {
					return output.JSON(result)
				}
				output.Successf("Investigation started for %s", output.Bold(args[0]))
				return nil
			}

			if all {
				req := api.InvestigateDailyRequest{
					Model:         model,
					ChunkSize:     chunkSize,
					ParallelLimit: parallel,
				}
				var result any
				if err := client.Post(ctx(), "/investigate/daily", req, &result); err != nil {
					return err
				}
				if flagJSON {
					return output.JSON(result)
				}
				output.Successf("Daily investigation started for all enabled repos")
				return nil
			}

			return fmt.Errorf("specify a repo name or use --all\n\nExamples:\n  reposwarm investigate my-repo\n  reposwarm investigate --all")
		},
	}

	cmd.Flags().BoolVar(&all, "all", false, "Investigate all enabled repos")
	cmd.Flags().StringVar(&model, "model", "", "Model ID (default from config)")
	cmd.Flags().IntVar(&chunkSize, "chunk-size", 0, "Files per chunk (default from config)")
	cmd.Flags().IntVar(&parallel, "parallel", 3, "Parallel limit (daily only)")
	return cmd
}
