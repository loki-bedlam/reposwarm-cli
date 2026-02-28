package commands

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/bootstrap"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newNewCmd() *cobra.Command {
	var dir string
	var agentMode bool
	var guideOnly bool

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Set up a new local RepoSwarm installation",
		Long: `Detects your local environment, generates a tailored installation guide,
and optionally hands it to a coding agent (Claude Code, Codex, etc.) for 
interactive setup.

Examples:
  reposwarm new                    # Interactive setup in ./reposwarm
  reposwarm new --dir ~/projects   # Custom install directory
  reposwarm new --agent            # Auto-launch coding agent
  reposwarm new --guide-only       # Just generate the guide file`,
		RunE: func(cmd *cobra.Command, args []string) error {
			// Detect environment
			env := bootstrap.Detect()

			if dir == "" {
				dir = env.InstallDir()
			}

			missing := env.MissingDeps()

			// JSON mode — clean output, no interactive
			if flagJSON {
				if !guideOnly {
					if err := os.MkdirAll(dir, 0755); err != nil {
						return fmt.Errorf("creating directory: %w", err)
					}
					guideContent := bootstrap.GenerateGuide(env, dir)
					agentGuideContent := bootstrap.GenerateAgentGuide(env, dir)
					os.WriteFile(filepath.Join(dir, "INSTALL.md"), []byte(guideContent), 0644)
					os.WriteFile(filepath.Join(dir, "REPOSWARM_INSTALL.md"), []byte(agentGuideContent), 0644)
				}
				return output.JSON(map[string]any{
					"environment":    env,
					"installDir":     dir,
					"missing":        missing,
					"agentAvailable": env.AgentName() != "",
					"agent":          env.AgentName(),
					"guidePath":      filepath.Join(dir, "INSTALL.md"),
					"agentGuidePath": filepath.Join(dir, "REPOSWARM_INSTALL.md"),
				})
			}

			// Interactive mode
			fmt.Printf("\n%s\n\n", output.Bold("🚀 RepoSwarm New Installation"))
			fmt.Println(output.Dim("  Scanning environment..."))
			fmt.Println()
			fmt.Println(env.Summary())

			if len(missing) > 0 {
				fmt.Printf("\n  %s Missing dependencies:\n", output.Yellow("⚠"))
				for _, dep := range missing {
					fmt.Printf("    %s %s\n", output.Red("✗"), dep)
				}
				fmt.Println()
			}

			// Generate guides
			guideContent := bootstrap.GenerateGuide(env, dir)
			agentGuideContent := bootstrap.GenerateAgentGuide(env, dir)

			if guideOnly {
				return writeGuides(dir, guideContent, agentGuideContent)
			}

			if err := writeGuides(dir, guideContent, agentGuideContent); err != nil {
				return err
			}

			// Check for coding agent
			agent := env.AgentName()
			if agent != "" && !agentMode {
				fmt.Printf("\n  %s detected! Use it for interactive installation? [Y/n] ",
					output.Bold(agentDisplayName(agent)))
				reader := bufio.NewReader(os.Stdin)
				line, _ := reader.ReadString('\n')
				line = strings.TrimSpace(strings.ToLower(line))
				if line == "" || line == "y" || line == "yes" {
					agentMode = true
				}
			}

			if agentMode && agent != "" {
				return launchAgent(agent, dir)
			}

			// No agent — show manual instructions
			fmt.Printf("\n  %s\n\n", output.Bold("Next steps:"))
			fmt.Printf("  1. Review the guide:     %s\n", output.Cyan(filepath.Join(dir, "INSTALL.md")))
			fmt.Printf("  2. Follow the steps to start each service\n")
			fmt.Printf("  3. Configure the CLI:    %s\n", output.Cyan("reposwarm config set apiUrl http://localhost:3000/v1"))
			fmt.Printf("  4. Verify:               %s\n", output.Cyan("reposwarm status"))

			if agent != "" {
				fmt.Printf("\n  Or let %s do it:\n", output.Bold(agentDisplayName(agent)))
				switch agent {
				case "claude":
					fmt.Printf("    %s\n", output.Cyan(fmt.Sprintf("cd %s && claude \"Follow REPOSWARM_INSTALL.md step by step\"", dir)))
				case "codex":
					fmt.Printf("    %s\n", output.Cyan(fmt.Sprintf("cd %s && codex \"Follow REPOSWARM_INSTALL.md step by step\"", dir)))
				case "aider":
					fmt.Printf("    %s\n", output.Cyan(fmt.Sprintf("cd %s && aider --read REPOSWARM_INSTALL.md", dir)))
				}
			}

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().StringVar(&dir, "dir", "", "Installation directory (default: ./reposwarm)")
	cmd.Flags().BoolVar(&agentMode, "agent", false, "Auto-launch coding agent for installation")
	cmd.Flags().BoolVar(&guideOnly, "guide-only", false, "Only generate guide files, don't prompt")
	return cmd
}

func writeGuides(dir, guide, agentGuide string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	installPath := filepath.Join(dir, "INSTALL.md")
	if err := os.WriteFile(installPath, []byte(guide), 0644); err != nil {
		return fmt.Errorf("writing INSTALL.md: %w", err)
	}
	output.Successf("Generated %s", installPath)

	agentPath := filepath.Join(dir, "REPOSWARM_INSTALL.md")
	if err := os.WriteFile(agentPath, []byte(agentGuide), 0644); err != nil {
		return fmt.Errorf("writing REPOSWARM_INSTALL.md: %w", err)
	}
	output.Successf("Generated %s (agent-friendly)", agentPath)

	return nil
}

func launchAgent(agent, dir string) error {
	guidePath := filepath.Join(dir, "REPOSWARM_INSTALL.md")

	fmt.Printf("\n  %s Launching %s...\n\n",
		output.Bold("🤖"), output.Bold(agentDisplayName(agent)))

	var cmd *exec.Cmd
	switch agent {
	case "claude":
		cmd = exec.Command("claude",
			"--print",
			fmt.Sprintf("Read %s and follow every step. Install RepoSwarm in %s. Verify each step before moving to the next.", guidePath, dir))
		cmd.Dir = dir
	case "codex":
		cmd = exec.Command("codex",
			fmt.Sprintf("Follow the instructions in REPOSWARM_INSTALL.md step by step to install RepoSwarm locally in %s", dir))
		cmd.Dir = dir
	case "aider":
		cmd = exec.Command("aider", "--read", guidePath)
		cmd.Dir = dir
	default:
		return fmt.Errorf("unsupported agent: %s", agent)
	}

	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("agent exited with error: %w", err)
	}

	fmt.Printf("\n  %s Agent finished. Verify with: %s\n\n",
		output.Bold("✅"), output.Cyan("reposwarm status"))
	return nil
}

func agentDisplayName(agent string) string {
	names := map[string]string{
		"claude": "Claude Code",
		"codex":  "Codex",
		"cursor": "Cursor",
		"aider":  "Aider",
	}
	if n, ok := names[agent]; ok {
		return n
	}
	return agent
}
