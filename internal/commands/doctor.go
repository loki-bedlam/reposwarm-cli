package commands

import (
	"context"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

// checkResult holds a single health check.
type checkResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "ok", "warn", "fail"
	Message string `json:"message"`
}

func newDoctorCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "doctor",
		Short: "Diagnose RepoSwarm installation health",
		Long: `Runs a series of checks to verify your RepoSwarm setup is working:
  - CLI configuration (API URL, token)
  - API server connectivity and health
  - Temporal server connectivity
  - DynamoDB connectivity
  - Worker status
  - Local dependencies (Docker, Node, Python, Git)
  - Network connectivity`,
		RunE: func(cmd *cobra.Command, args []string) error {
			var checks []checkResult

			if !flagJSON {
				fmt.Printf("\n%s\n\n", output.Bold("🩺 RepoSwarm Doctor"))
			}

			// 1. Config file
			checks = append(checks, checkConfig()...)

			// 2. API connectivity
			checks = append(checks, checkAPI()...)

			// 3. Local tools
			checks = append(checks, checkLocalTools()...)

			// 4. Network
			checks = append(checks, checkNetwork()...)

			if flagJSON {
				summary := map[string]any{
					"checks": checks,
					"ok":     countStatus(checks, "ok"),
					"warn":   countStatus(checks, "warn"),
					"fail":   countStatus(checks, "fail"),
				}
				return output.JSON(summary)
			}

			// Summary
			ok := countStatus(checks, "ok")
			warn := countStatus(checks, "warn")
			fail := countStatus(checks, "fail")
			fmt.Println()
			if fail == 0 && warn == 0 {
				fmt.Printf("  %s All %d checks passed\n\n", output.Green("✅"), ok)
			} else if fail == 0 {
				fmt.Printf("  %s %d passed, %d warnings\n\n", output.Yellow("⚠️"), ok, warn)
			} else {
				fmt.Printf("  %s %d passed, %d warnings, %d failed\n\n", output.Red("❌"), ok, warn, fail)
			}
			return nil
		},
	}
}

func printCheck(c checkResult) {
	if flagJSON {
		return
	}
	icon := output.Green("✓")
	if c.Status == "warn" {
		icon = output.Yellow("⚠")
	} else if c.Status == "fail" {
		icon = output.Red("✗")
	}
	fmt.Printf("  %s %s — %s\n", icon, c.Name, c.Message)
}

func checkConfig() []checkResult {
	var results []checkResult

	cfg, err := config.Load()
	if err != nil {
		c := checkResult{"Config file", "fail", fmt.Sprintf("error loading: %s", err)}
		printCheck(c)
		return append(results, c)
	}

	// Config path
	path, _ := config.ConfigPath()
	if _, err := os.Stat(path); err != nil {
		c := checkResult{"Config file", "warn", "no config file — using defaults. Run 'reposwarm config init'"}
		printCheck(c)
		results = append(results, c)
	} else {
		c := checkResult{"Config file", "ok", path}
		printCheck(c)
		results = append(results, c)
	}

	// API URL
	if cfg.APIUrl == "" {
		c := checkResult{"API URL", "fail", "not configured"}
		printCheck(c)
		results = append(results, c)
	} else {
		c := checkResult{"API URL", "ok", cfg.APIUrl}
		printCheck(c)
		results = append(results, c)
	}

	// API Token
	if cfg.APIToken == "" {
		c := checkResult{"API token", "fail", "not configured — run 'reposwarm config init'"}
		printCheck(c)
		results = append(results, c)
	} else {
		c := checkResult{"API token", "ok", config.MaskedToken(cfg.APIToken)}
		printCheck(c)
		results = append(results, c)
	}

	return results
}

func checkAPI() []checkResult {
	var results []checkResult

	client, err := getClient()
	if err != nil {
		c := checkResult{"API connection", "fail", fmt.Sprintf("cannot create client: %s", err)}
		printCheck(c)
		return append(results, c)
	}

	start := time.Now()
	health, err := client.Health(context.Background())
	latency := time.Since(start)

	if err != nil {
		c := checkResult{"API connection", "fail", fmt.Sprintf("unreachable: %s", err)}
		printCheck(c)
		results = append(results, c)
		return results
	}

	c := checkResult{"API connection", "ok", fmt.Sprintf("%s (%dms)", health.Status, latency.Milliseconds())}
	printCheck(c)
	results = append(results, c)

	// Temporal
	if health.Temporal.Connected {
		c = checkResult{"Temporal", "ok", fmt.Sprintf("connected (ns: %s, queue: %s)", health.Temporal.Namespace, health.Temporal.TaskQueue)}
	} else {
		c = checkResult{"Temporal", "fail", "not connected"}
	}
	printCheck(c)
	results = append(results, c)

	// DynamoDB
	if health.DynamoDB.Connected {
		c = checkResult{"DynamoDB", "ok", "connected"}
	} else {
		c = checkResult{"DynamoDB", "fail", "not connected"}
	}
	printCheck(c)
	results = append(results, c)

	// Worker
	if health.Worker.Connected {
		c = checkResult{"Worker", "ok", fmt.Sprintf("connected (%d active)", health.Worker.Count)}
	} else {
		c = checkResult{"Worker", "warn", "no worker connected — investigations will queue but not run"}
	}
	printCheck(c)
	results = append(results, c)

	return results
}

func checkLocalTools() []checkResult {
	var results []checkResult

	tools := []struct {
		name    string
		cmd     string
		args    []string
		level   string // "fail" or "warn" if missing
	}{
		{"Git", "git", []string{"--version"}, "warn"},
		{"Docker", "docker", []string{"--version"}, "warn"},
		{"Node.js", "node", []string{"--version"}, "warn"},
		{"Python", "python3", []string{"--version"}, "warn"},
		{"AWS CLI", "aws", []string{"--version"}, "warn"},
	}

	for _, t := range tools {
		out, err := exec.Command(t.cmd, t.args...).Output()
		if err != nil {
			c := checkResult{t.name, t.level, "not found"}
			printCheck(c)
			results = append(results, c)
		} else {
			ver := strings.TrimSpace(string(out))
			if len(ver) > 60 {
				ver = ver[:60] + "..."
			}
			c := checkResult{t.name, "ok", ver}
			printCheck(c)
			results = append(results, c)
		}
	}

	return results
}

func checkNetwork() []checkResult {
	var results []checkResult

	// DNS resolution
	_, err := net.LookupHost("github.com")
	if err != nil {
		c := checkResult{"DNS", "fail", "cannot resolve github.com"}
		printCheck(c)
		results = append(results, c)
	} else {
		c := checkResult{"DNS", "ok", "resolving"}
		printCheck(c)
		results = append(results, c)
	}

	// GitHub connectivity
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get("https://api.github.com")
	if err != nil {
		c := checkResult{"GitHub API", "warn", fmt.Sprintf("unreachable: %s", err)}
		printCheck(c)
		results = append(results, c)
	} else {
		resp.Body.Close()
		c := checkResult{"GitHub API", "ok", fmt.Sprintf("HTTP %d", resp.StatusCode)}
		printCheck(c)
		results = append(results, c)
	}

	return results
}

func countStatus(checks []checkResult, status string) int {
	n := 0
	for _, c := range checks {
		if c.Status == status {
			n++
		}
	}
	return n
}
