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
	var localMode bool

	cmd := &cobra.Command{
		Use:   "new",
		Short: "Set up a new local RepoSwarm installation",
		Long: `Detects your local environment, generates a tailored installation guide,
and optionally hands it to a coding agent (Claude Code, Codex, etc.) for 
interactive setup.

Use --local to automatically set up and start all services locally
(Temporal, API, Worker, UI) via Docker Compose and npm/pip.

Examples:
  reposwarm new                    # Interactive setup in ./reposwarm
  reposwarm new --local            # Automated local setup (start everything)
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

			// --local mode: automated setup
			if localMode {
				if flagJSON {
					printer := &jsonPrinter{}
					result, err := bootstrap.SetupLocal(env, dir, printer)
					if err != nil {
						// Still output what we have
						return output.JSON(result)
					}
					return output.JSON(result)
				}
				printer := &fmtPrinter{}
				_, err := bootstrap.SetupLocal(env, dir, printer)
				return err
			}

			// JSON mode — generate guides
			if flagJSON {
				guideContent := bootstrap.GenerateGuide(env, dir)
				agentGuideContent := bootstrap.GenerateAgentGuide(env, dir)

				if err := writeGuidesSilent(dir, guideContent, agentGuideContent); err != nil {
					return err
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
					output.F.Printf("  %s: missing\n", dep)
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
			fmt.Printf("\n  Or use automated setup:  %s\n", output.Cyan("reposwarm new --local"))

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
	cmd.Flags().BoolVar(&localMode, "local", false, "Automated local setup: start Temporal, API, Worker, and UI")
	return cmd
}

// fmtPrinter implements bootstrap.Printer using the output formatter.
type fmtPrinter struct{}

func (p *fmtPrinter) Section(title string) { output.F.Section(title) }
func (p *fmtPrinter) Info(msg string)      { output.F.Info(msg) }
func (p *fmtPrinter) Success(msg string)   { output.F.Success(msg) }
func (p *fmtPrinter) Warning(msg string)   { output.F.Warning(msg) }
func (p *fmtPrinter) Error(msg string)     { output.F.Error(msg) }
func (p *fmtPrinter) Printf(format string, args ...any) {
	output.F.Printf(format, args...)
}

// jsonPrinter is a no-op printer for JSON mode (output comes from the result struct).
type jsonPrinter struct{}

func (p *jsonPrinter) Section(string)              {}
func (p *jsonPrinter) Info(string)                 {}
func (p *jsonPrinter) Success(string)              {}
func (p *jsonPrinter) Warning(string)              {}
func (p *jsonPrinter) Error(string)                {}
func (p *jsonPrinter) Printf(string, ...any)       {}

func writeGuidesSilent(dir, guide, agentGuide string) error {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}
	installPath := filepath.Join(dir, "INSTALL.md")
	if err := os.WriteFile(installPath, []byte(guide), 0644); err != nil {
		return fmt.Errorf("writing INSTALL.md: %w", err)
	}
	agentPath := filepath.Join(dir, "REPOSWARM_INSTALL.md")
	if err := os.WriteFile(agentPath, []byte(agentGuide), 0644); err != nil {
		return fmt.Errorf("writing REPOSWARM_INSTALL.md: %w", err)
	}
	return nil
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
		"Done!", "reposwarm status")
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
