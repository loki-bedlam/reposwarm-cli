package commands

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newConfigWorkerEnvCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "worker-env",
		Aliases: []string{"env"},
		Short:   "Manage worker environment variables",
	}
	cmd.AddCommand(newWorkerEnvListCmd())
	cmd.AddCommand(newWorkerEnvSetCmd())
	cmd.AddCommand(newWorkerEnvUnsetCmd())
	return cmd
}

func newWorkerEnvListCmd() *cobra.Command {
	var reveal bool

	cmd := &cobra.Command{
		Use:   "list",
		Short: "Show all worker environment variables",
		RunE: func(cmd *cobra.Command, args []string) error {
			envFile := workerEnvPath()
			vars := readOrderedEnv(envFile)

			// Known vars we always want to show
			known := []string{
				"ANTHROPIC_API_KEY", "GITHUB_TOKEN", "GITHUB_PAT",
				"AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY", "AWS_DEFAULT_REGION",
				"CLAUDE_MODEL", "MODEL_ID", "MODEL",
				"TEMPORAL_SERVER_URL", "DYNAMODB_TABLE_NAME",
				"API_BEARER_TOKEN",
			}

			// Merge known + found
			seen := map[string]bool{}
			type envEntry struct {
				Key    string `json:"key"`
				Value  string `json:"value"`
				Source string `json:"source"`
				Set    bool   `json:"set"`
			}
			var entries []envEntry

			addEntry := func(key string) {
				if seen[key] {
					return
				}
				seen[key] = true

				val, source := "", "—"
				set := false

				if v, ok := vars[key]; ok {
					val = v
					source = ".env"
					set = true
				} else if v := os.Getenv(key); v != "" {
					val = v
					source = "environment"
					set = true
				}

				if !reveal && set && len(val) > 8 {
					val = val[:4] + "..." + val[len(val)-4:]
				} else if !reveal && set {
					val = "***"
				}
				if !set {
					val = "(not set)"
				}

				entries = append(entries, envEntry{key, val, source, set})
			}

			for _, k := range known {
				addEntry(k)
			}
			for _, kv := range orderedKeys(vars) {
				addEntry(kv)
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"envFile": envFile,
					"entries": entries,
				})
			}

			F := output.F
			F.Section("Worker Environment")
			F.KeyValue("Env file", envFile)
			F.Println()

			headers := []string{"Variable", "Value", "Source"}
			var rows [][]string
			for _, e := range entries {
				valStr := e.Value
				if !e.Set {
					valStr = output.Dim("(not set)")
				}
				rows = append(rows, []string{e.Key, valStr, e.Source})
			}
			output.Table(headers, rows)
			F.Println()
			return nil
		},
	}

	cmd.Flags().BoolVar(&reveal, "reveal", false, "Show full unmasked values")
	return cmd
}

func newWorkerEnvSetCmd() *cobra.Command {
	var restart bool

	cmd := &cobra.Command{
		Use:   "set <KEY> <VALUE>",
		Short: "Set a worker environment variable",
		Args:  friendlyExactArgs(2, "reposwarm config worker-env set <KEY> <VALUE>\n\nExample:\n  reposwarm config worker-env set ANTHROPIC_API_KEY sk-ant-abc123"),
		RunE: func(cmd *cobra.Command, args []string) error {
			key, value := args[0], args[1]
			envFile := workerEnvPath()

			if err := setEnvVar(envFile, key, value); err != nil {
				return err
			}

			maskedVal := value
			if len(value) > 8 {
				maskedVal = value[:4] + "..." + value[len(value)-4:]
			} else {
				maskedVal = "***"
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"key":     key,
					"value":   maskedVal,
					"envFile": envFile,
					"restart": restart,
				})
			}

			output.Successf("Set %s = %s (written to %s)", key, maskedVal, envFile)

			if restart {
				output.F.Info("Restarting worker...")
				if err := restartService("worker"); err != nil {
					output.F.Warning(fmt.Sprintf("Could not restart: %v", err))
					output.F.Info("Restart manually: reposwarm restart worker")
				}
			} else {
				output.F.Warning("Worker restart required. Run: reposwarm restart worker")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&restart, "restart", false, "Automatically restart the worker after setting")
	return cmd
}

func newWorkerEnvUnsetCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "unset <KEY>",
		Short: "Remove a worker environment variable",
		Args:  friendlyExactArgs(1, "reposwarm config worker-env unset <KEY>\n\nExample:\n  reposwarm config worker-env unset SOME_VAR"),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			envFile := workerEnvPath()

			if err := unsetEnvVar(envFile, key); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"key":     key,
					"envFile": envFile,
					"removed": true,
				})
			}

			output.Successf("Removed %s from %s", key, envFile)
			output.F.Warning("Worker restart required. Run: reposwarm restart worker")
			return nil
		},
	}
	return cmd
}

func workerEnvPath() string {
	cfg, _ := config.Load()
	if cfg != nil {
		return filepath.Join(cfg.EffectiveInstallDir(), "worker", ".env")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, "reposwarm", "worker", ".env")
}

func readOrderedEnv(path string) map[string]string {
	m := make(map[string]string)
	f, err := os.Open(path)
	if err != nil {
		return m
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			m[parts[0]] = strings.TrimSpace(parts[1])
		}
	}
	return m
}

func orderedKeys(m map[string]string) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}

func setEnvVar(envFile, key, value string) error {
	// Ensure directory exists
	dir := filepath.Dir(envFile)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory %s: %w", dir, err)
	}

	// Read existing content
	var lines []string
	if data, err := os.ReadFile(envFile); err == nil {
		lines = strings.Split(string(data), "\n")
	}

	// Find and replace existing key, or append
	found := false
	for i, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, key+"=") {
			lines[i] = fmt.Sprintf("%s=%s", key, value)
			found = true
			break
		}
	}

	if !found {
		lines = append(lines, fmt.Sprintf("%s=%s", key, value))
	}

	// Remove trailing empty lines
	for len(lines) > 0 && strings.TrimSpace(lines[len(lines)-1]) == "" {
		lines = lines[:len(lines)-1]
	}

	content := strings.Join(lines, "\n") + "\n"
	return os.WriteFile(envFile, []byte(content), 0600)
}

func unsetEnvVar(envFile, key string) error {
	data, err := os.ReadFile(envFile)
	if err != nil {
		return fmt.Errorf("reading %s: %w", envFile, err)
	}

	lines := strings.Split(string(data), "\n")
	var newLines []string
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if !strings.HasPrefix(trimmed, key+"=") {
			newLines = append(newLines, line)
		}
	}

	// Remove trailing empty lines
	for len(newLines) > 0 && strings.TrimSpace(newLines[len(newLines)-1]) == "" {
		newLines = newLines[:len(newLines)-1]
	}

	content := strings.Join(newLines, "\n") + "\n"
	return os.WriteFile(envFile, []byte(content), 0600)
}
