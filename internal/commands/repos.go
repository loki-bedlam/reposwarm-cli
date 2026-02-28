package commands

import (
	"fmt"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newReposCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "repos",
		Short: "Manage tracked repositories",
	}
	cmd.AddCommand(newReposListCmd())
	cmd.AddCommand(newReposShowCmd())
	cmd.AddCommand(newReposAddCmd())
	cmd.AddCommand(newReposRemoveCmd())
	cmd.AddCommand(newReposEnableCmd())
	cmd.AddCommand(newReposDisableCmd())
	return cmd
}

func newReposListCmd() *cobra.Command {
	var source, filter string
	var enabled, disabled bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all tracked repositories",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var repos []api.Repository
			if err := client.Get(ctx(), "/repos", &repos); err != nil {
				return err
			}

			// Apply filters
			var filtered []api.Repository
			for _, r := range repos {
				if source != "" && !strings.EqualFold(r.Source, source) {
					continue
				}
				if filter != "" && !strings.Contains(strings.ToLower(r.Name), strings.ToLower(filter)) {
					continue
				}
				if enabled && !r.Enabled {
					continue
				}
				if disabled && r.Enabled {
					continue
				}
				filtered = append(filtered, r)
			}

			if flagJSON {
				return output.JSON(filtered)
			}

			fmt.Printf("\n  %s (%d repos)\n\n", output.Bold("Repositories"), len(filtered))
			headers := []string{"Name", "Source", "Enabled", "Docs", "Status"}
			var rows [][]string
			for _, r := range filtered {
				en := output.Green("✓")
				if !r.Enabled {
					en = output.Dim("✗")
				}
				docs := ""
				if r.HasDocs {
					docs = output.Green("✓")
				}
				rows = append(rows, []string{r.Name, r.Source, en, docs, r.Status})
			}
			output.Table(headers, rows)
			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&source, "source", "", "Filter by source (CodeCommit, GitHub)")
	cmd.Flags().StringVar(&filter, "filter", "", "Filter by name (case-insensitive)")
	cmd.Flags().BoolVar(&enabled, "enabled", false, "Show only enabled repos")
	cmd.Flags().BoolVar(&disabled, "disabled", false, "Show only disabled repos")
	return cmd
}

func newReposAddCmd() *cobra.Command {
	var url, source string

	cmd := &cobra.Command{
		Use:   "add <name>",
		Short: "Add a repository to track",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			body := map[string]any{
				"name":   args[0],
				"url":    url,
				"source": source,
			}

			var result any
			if err := client.Post(ctx(), "/repos", body, &result); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(result)
			}
			output.Successf("Added repository %s", output.Bold(args[0]))
			return nil
		},
	}

	cmd.Flags().StringVar(&url, "url", "", "Repository URL")
	cmd.Flags().StringVar(&source, "source", "CodeCommit", "Source (CodeCommit, GitHub)")
	return cmd
}

func newReposRemoveCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "remove <name>",
		Short: "Remove a tracked repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if !yes {
				fmt.Printf("  Remove %s? [y/N] ", output.Bold(args[0]))
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" {
					output.Infof("Cancelled")
					return nil
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var result any
			if err := client.Delete(ctx(), "/repos/"+args[0], &result); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(result)
			}
			output.Successf("Removed repository %s", args[0])
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}

func newReposEnableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "enable <name>",
		Short: "Enable a repository for investigation",
		Args:  cobra.ExactArgs(1),
		RunE:  repoToggle(true),
	}
}

func newReposDisableCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "disable <name>",
		Short: "Disable a repository from investigation",
		Args:  cobra.ExactArgs(1),
		RunE:  repoToggle(false),
	}
}

func repoToggle(enable bool) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		client, err := getClient()
		if err != nil {
			return err
		}

		body := map[string]any{"enabled": enable}
		var result any
		if err := client.Patch(ctx(), "/repos/"+args[0], body, &result); err != nil {
			return err
		}

		action := "Enabled"
		if !enable {
			action = "Disabled"
		}
		if flagJSON {
			return output.JSON(map[string]any{"name": args[0], "enabled": enable})
		}
		output.Successf("%s repository %s", action, args[0])
		return nil
	}
}

func newReposShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <name>",
		Short: "Show detailed info for a single repository",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var repo api.Repository
			if err := client.Get(ctx(), "/repos/"+args[0], &repo); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(repo)
			}

			fmt.Printf("\n  %s\n\n", output.Bold("Repository: "+repo.Name))
			fmt.Printf("  %s  %s\n", output.Dim("Source     "), repo.Source)
			fmt.Printf("  %s  %s\n", output.Dim("URL        "), repo.URL)
			fmt.Printf("  %s  %v\n", output.Dim("Enabled    "), repo.Enabled)
			fmt.Printf("  %s  %v\n", output.Dim("Has Docs   "), repo.HasDocs)
			fmt.Printf("  %s  %s\n", output.Dim("Status     "), repo.Status)
			if repo.Description != "" {
				fmt.Printf("  %s  %s\n", output.Dim("Description"), repo.Description)
			}
			fmt.Println()
			return nil
		},
	}
}
