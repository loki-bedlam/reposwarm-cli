package commands

import (
	"fmt"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newReposSyncCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Discover and add CodeCommit repositories to tracking",
		Long: `Discovers all CodeCommit repositories in the AWS account and adds
any new ones to tracking. Does NOT remove existing repos.

To remove repos, use: reposwarm repos remove <name>`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var discoverResult api.DiscoverResult
			if err := client.Post(ctx(), "/repos/discover", nil, &discoverResult); err != nil {
				return fmt.Errorf("discover failed: %w", err)
			}

			if flagJSON {
				return output.JSON(discoverResult)
			}

			fmt.Println()
			output.Successf("Discovered %s CodeCommit repos", output.Bold(fmt.Sprint(discoverResult.Discovered)))
			if discoverResult.Added > 0 {
				output.Successf("Added %s new repos", output.Bold(fmt.Sprint(discoverResult.Added)))
			} else {
				output.Infof("All repos already tracked (%d skipped)", discoverResult.Skipped)
			}
			fmt.Println()
			return nil
		},
	}

	return cmd
}
