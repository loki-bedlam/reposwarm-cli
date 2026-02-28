// Package commands defines all CLI commands.
package commands

import (
	"context"
	"fmt"
	"os"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

var (
	flagJSON     bool
	flagAPIUrl   string
	flagAPIToken string
	flagNoColor  bool
	flagVerbose  bool
)

// NewRootCmd creates the root cobra command with all subcommands.
func NewRootCmd(version string) *cobra.Command {
	root := &cobra.Command{
		Use:   "reposwarm",
		Short: "CLI for RepoSwarm — AI-powered multi-repo architecture discovery",
		Long: `RepoSwarm CLI provides command-line access to the RepoSwarm platform.
Discover repositories, trigger investigations, browse results, and manage prompts.

Get started:
  reposwarm config init        Set up API connection
  reposwarm status             Check connection and services
  reposwarm repos list         List tracked repositories
  reposwarm results list       Browse investigation results
  reposwarm prompts list       View investigation prompts`,
		Version: version,
		PersistentPreRun: func(cmd *cobra.Command, args []string) {
			if flagNoColor {
				output.Bold = fmt.Sprint
				output.Green = fmt.Sprint
				output.Red = fmt.Sprint
				output.Yellow = fmt.Sprint
				output.Cyan = fmt.Sprint
				output.Dim = fmt.Sprint
			}
		},
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	root.PersistentFlags().BoolVar(&flagJSON, "json", false, "Output as JSON (agent-friendly)")
	root.PersistentFlags().StringVar(&flagAPIUrl, "api-url", "", "API server URL (overrides config)")
	root.PersistentFlags().StringVar(&flagAPIToken, "api-token", "", "API bearer token (overrides config)")
	root.PersistentFlags().BoolVar(&flagNoColor, "no-color", false, "Disable colored output")
	root.PersistentFlags().BoolVar(&flagVerbose, "verbose", false, "Show debug info")

	// Core
	root.AddCommand(newStatusCmd())
	root.AddCommand(newConfigCmd())

	// Repos
	root.AddCommand(newReposCmd())
	root.AddCommand(newDiscoverCmd())

	// Workflows
	root.AddCommand(newWorkflowsCmd())
	root.AddCommand(newInvestigateCmd())
	root.AddCommand(newWatchCmd())

	// Results
	root.AddCommand(newResultsCmd())
	root.AddCommand(newDiffCmd())

	// Prompts
	root.AddCommand(newPromptsCmd())

	// Bootstrap
	root.AddCommand(newNewCmd())

	// Server
	root.AddCommand(newServerConfigCmd())

	return root
}

// getClient creates an API client from config + flag overrides.
func getClient() (*api.Client, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}

	url := cfg.APIUrl
	token := cfg.APIToken
	if flagAPIUrl != "" {
		url = flagAPIUrl
	}
	if flagAPIToken != "" {
		token = flagAPIToken
	}

	if url == "" {
		return nil, fmt.Errorf("no API URL configured: run 'reposwarm config init' or pass --api-url")
	}
	if token == "" {
		return nil, fmt.Errorf("no API token configured: run 'reposwarm config init' or pass --api-token")
	}

	return api.New(url, token), nil
}

// ctx returns a background context.
func ctx() context.Context {
	return context.Background()
}

// Execute runs the root command.
func Execute(version string) {
	root := NewRootCmd(version)
	if err := root.Execute(); err != nil {
		output.Errorf("%s", err)
		os.Exit(1)
	}
}
