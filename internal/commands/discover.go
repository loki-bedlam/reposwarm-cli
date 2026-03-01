package commands

import (
	"fmt"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newDiscoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Auto-discover repositories from CodeCommit",
		Long:  "Triggers server-side discovery of CodeCommit repositories and adds new ones to tracking.",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var result api.DiscoverResult
			if err := client.Post(ctx(), "/repos/discover", nil, &result); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(result)
			}

			F := output.F
			F.Success(fmt.Sprintf("Discovered %d CodeCommit repos", result.Discovered))
			if result.Added > 0 {
				F.Success(fmt.Sprintf("Added %d new repos", result.Added))
			} else {
				F.Info(fmt.Sprintf("All repos already tracked (%d skipped)", result.Skipped))
			}
			return nil
		},
	}
}
