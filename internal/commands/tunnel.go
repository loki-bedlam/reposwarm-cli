package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/reposwarm/reposwarm-cli/internal/config"
	"github.com/reposwarm/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newTunnelCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "tunnel",
		Short: "Show SSH tunnel instructions to access RepoSwarm UI remotely",
		Long: `Shows how to set up an SSH tunnel to access the RepoSwarm UI,
API server, and Temporal dashboard from your local browser.

Useful when RepoSwarm is installed on a remote server (EC2, VPS, etc.)
and you want to access the web UIs without exposing ports publicly.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			cfg, _ := config.Load()

			uiPort := "3001"
			apiPort := "3000"
			temporalPort := "8233"
			if cfg != nil {
				uiPort = cfg.EffectiveUIPort()
				apiPort = cfg.EffectiveAPIPort()
				temporalPort = cfg.EffectiveTemporalUIPort()
			}

			// Detect hostname / IP
			hostname := detectHostname()
			user := detectSSHUser()
			keyHint := detectSSHKey()

			if flagJSON {
				return output.JSON(map[string]any{
					"hostname":     hostname,
					"user":         user,
					"uiPort":       uiPort,
					"apiPort":      apiPort,
					"temporalPort": temporalPort,
					"tunnelCmd":    buildTunnelCmd(user, hostname, keyHint, uiPort, apiPort, temporalPort),
				})
			}

			if flagAgent {
				fmt.Println(buildTunnelCmd(user, hostname, keyHint, uiPort, apiPort, temporalPort))
				fmt.Printf("Then open: http://localhost:%s\n", uiPort)
				return nil
			}

			output.F.Section("SSH Tunnel Setup")
			fmt.Println()
			output.F.Info("Access RepoSwarm from your local browser via SSH tunnel.")
			fmt.Println()

			// Single tunnel command for all services
			fmt.Printf("  %s Run this on your %s machine:\n\n", output.Bold("1."), output.Bold("local"))
			tunnelCmd := buildTunnelCmd(user, hostname, keyHint, uiPort, apiPort, temporalPort)
			fmt.Printf("    %s\n\n", output.Cyan(tunnelCmd))

			fmt.Printf("  %s Open in your browser:\n\n", output.Bold("2."))
			fmt.Printf("    %-20s %s\n", output.Bold("RepoSwarm UI:"), output.Cyan(fmt.Sprintf("http://localhost:%s", uiPort)))
			fmt.Printf("    %-20s %s\n", output.Bold("API Server:"), output.Cyan(fmt.Sprintf("http://localhost:%s", apiPort)))
			fmt.Printf("    %-20s %s\n", output.Bold("Temporal UI:"), output.Cyan(fmt.Sprintf("http://localhost:%s", temporalPort)))
			fmt.Println()

			// Tips
			output.F.Section("Tips")
			fmt.Println()
			fmt.Printf("  • Add %s to keep tunnel alive\n", output.Cyan("-N"))
			fmt.Printf("  • Add %s to run in background\n", output.Cyan("-f"))
			fmt.Printf("  • Background + keep-alive:\n")
			bgCmd := buildTunnelCmd(user, hostname, keyHint, uiPort, apiPort, temporalPort)
			bgCmd = strings.Replace(bgCmd, "ssh ", "ssh -fN ", 1)
			fmt.Printf("    %s\n\n", output.Cyan(bgCmd))

			// SSH config shortcut
			fmt.Printf("  • Or add to %s:\n\n", output.Cyan("~/.ssh/config"))
			fmt.Printf("    %s\n", output.Dim("Host reposwarm"))
			fmt.Printf("    %s\n", output.Dim(fmt.Sprintf("  HostName %s", hostname)))
			fmt.Printf("    %s\n", output.Dim(fmt.Sprintf("  User %s", user)))
			if keyHint != "" {
				fmt.Printf("    %s\n", output.Dim(fmt.Sprintf("  IdentityFile %s", keyHint)))
			}
			fmt.Printf("    %s\n", output.Dim(fmt.Sprintf("  LocalForward %s localhost:%s", uiPort, uiPort)))
			fmt.Printf("    %s\n", output.Dim(fmt.Sprintf("  LocalForward %s localhost:%s", apiPort, apiPort)))
			fmt.Printf("    %s\n\n", output.Dim(fmt.Sprintf("  LocalForward %s localhost:%s", temporalPort, temporalPort)))
			fmt.Printf("    Then just: %s\n\n", output.Cyan("ssh -fN reposwarm"))

			return nil
		},
	}

	return cmd
}

func buildTunnelCmd(user, host, key, uiPort, apiPort, temporalPort string) string {
	parts := []string{"ssh"}

	if key != "" {
		parts = append(parts, fmt.Sprintf("-i %s", key))
	}

	parts = append(parts,
		fmt.Sprintf("-L %s:localhost:%s", uiPort, uiPort),
		fmt.Sprintf("-L %s:localhost:%s", apiPort, apiPort),
		fmt.Sprintf("-L %s:localhost:%s", temporalPort, temporalPort),
		fmt.Sprintf("%s@%s", user, host),
	)

	return strings.Join(parts, " ")
}

func detectHostname() string {
	// Try public IP first (EC2 metadata)
	if out, err := exec.Command("curl", "-s", "--connect-timeout", "1",
		"http://169.254.169.254/latest/meta-data/public-ipv4").Output(); err == nil {
		ip := strings.TrimSpace(string(out))
		if ip != "" && !strings.Contains(ip, "<!") {
			return ip
		}
	}

	// Try public hostname
	if out, err := exec.Command("curl", "-s", "--connect-timeout", "1",
		"http://169.254.169.254/latest/meta-data/public-hostname").Output(); err == nil {
		h := strings.TrimSpace(string(out))
		if h != "" && !strings.Contains(h, "<!") {
			return h
		}
	}

	// Fallback to hostname
	if h, err := os.Hostname(); err == nil {
		return h
	}

	return "<your-server-ip>"
}

func detectSSHUser() string {
	if u := os.Getenv("USER"); u != "" {
		return u
	}
	return "ubuntu"
}

func detectSSHKey() string {
	// Check common key locations
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}

	// Check for EC2-style key pairs in ~/.ssh/
	candidates := []string{
		"reposwarm.pem",
		"reposwarm-key.pem",
		"id_ed25519",
		"id_rsa",
	}

	for _, k := range candidates {
		path := fmt.Sprintf("%s/.ssh/%s", home, k)
		if fileExists(path) {
			return path
		}
	}

	return ""
}
