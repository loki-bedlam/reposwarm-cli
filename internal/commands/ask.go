package commands

import (
	"fmt"
	"strings"

	"github.com/reposwarm/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Ask the RepoSwarm AI assistant a question",
		Long: `Ask a question about RepoSwarm and get an AI-powered answer.
Uses the configured LLM provider (Bedrock, Anthropic, or LiteLLM).
The API server handles the inference — works whether local or remote.

Examples:
  reposwarm ask "how do I add a new repo?"
  reposwarm ask "what does doctor --fix do?"
  reposwarm ask "how to switch from Anthropic to Bedrock?"
  reposwarm ask "my worker won't start, what should I check?"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")

			client, err := getClient()
			if err != nil {
				return err
			}

			if !flagJSON && !flagAgent {
				fmt.Printf("  %s Thinking...\r", output.Dim("⏳"))
			}

			var resp struct {
				Success bool   `json:"success"`
				Answer  string `json:"answer"`
				Model   string `json:"model"`
				Latency int    `json:"latencyMs"`
				Error   string `json:"error"`
				Hint    string `json:"hint"`
			}

			err = client.Post(ctx(), "/ask", map[string]string{"question": question}, &resp)
			if err != nil {
				return fmt.Errorf("ask failed: %w", err)
			}

			if !resp.Success {
				if flagJSON {
					return output.JSON(map[string]any{
						"success": false,
						"error":   resp.Error,
						"hint":    resp.Hint,
					})
				}
				msg := resp.Error
				if resp.Hint != "" {
					msg += "\n  💡 " + resp.Hint
				}
				return fmt.Errorf("%s", msg)
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"success":  true,
					"answer":   resp.Answer,
					"model":    resp.Model,
					"latencyMs": resp.Latency,
				})
			}

			if flagAgent {
				fmt.Print(resp.Answer)
				return nil
			}

			// Clear the "Thinking..." line
			fmt.Print("\r\033[K")

			// Print answer with light formatting
			fmt.Println(resp.Answer)
			fmt.Println()
			fmt.Printf("  %s\n", output.Dim(fmt.Sprintf("— %s (%dms)", resp.Model, resp.Latency)))

			return nil
		},
	}

	return cmd
}
