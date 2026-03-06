package commands

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/reposwarm/reposwarm-cli/internal/api"
	"github.com/reposwarm/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newAskCmd() *cobra.Command {
	var archFlag bool
	var reposFlag string
	var adapterFlag string
	var noWaitFlag bool

	cmd := &cobra.Command{
		Use:   "ask <question>",
		Short: "Ask the RepoSwarm AI assistant a question",
		Long: `Ask a question about RepoSwarm or your architecture.

Without --arch: asks about RepoSwarm CLI usage (fast, simple Q&A).
With --arch:    queries your architecture docs using the askbox agent (slower, thorough).

The askbox reads your .arch.md files and reasons across repos to answer
complex architecture questions.

Flags:
  --arch              Use the askbox agent for architecture analysis
  --repos <list>      Comma-separated repos to scope the question to
  --adapter <name>    Agent adapter: claude-agent-sdk (default) or strands
  --no-wait           Return ask-id immediately without waiting for answer

Output modes:
  (default)           Human-friendly with progress indicators
  --for-agent         Plain text answer only, no formatting
  --json              Structured JSON output

Examples:
  reposwarm ask "how do I add a new repo?"
  reposwarm ask --arch "how does auth work across all services?"
  reposwarm ask --arch --repos my-api,billing "how do they communicate?"
  reposwarm ask --arch --adapter strands "what databases are used?"
  reposwarm ask --arch --no-wait --json "what patterns do repos share?"
  reposwarm ask --arch --for-agent "summarize the test strategies"`,
		Args: cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			question := strings.Join(args, " ")

			client, err := getClient()
			if err != nil {
				return err
			}

			if archFlag {
				return runArchAsk(client, question, reposFlag, adapterFlag, noWaitFlag)
			}

			return runSimpleAsk(client, question)
		},
	}

	cmd.Flags().BoolVar(&archFlag, "arch", false, "Query architecture docs using the askbox agent")
	cmd.Flags().StringVar(&reposFlag, "repos", "", "Comma-separated list of repos to scope the question to")
	cmd.Flags().StringVar(&adapterFlag, "adapter", "", "Agent adapter: claude-agent-sdk (default) or strands")
	cmd.Flags().BoolVar(&noWaitFlag, "no-wait", false, "Submit and return ask-id without waiting (for agents)")

	return cmd
}

func runSimpleAsk(client *api.Client, question string) error {
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

	err := client.Post(ctx(), "/ask", map[string]string{"question": question}, &resp)
	if err != nil {
		if flagJSON {
			return output.JSON(map[string]any{"success": false, "error": err.Error()})
		}
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
			"success":   true,
			"answer":    resp.Answer,
			"model":     resp.Model,
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
}

func runArchAsk(client *api.Client, question, repos, adapter string, noWait bool) error {

	body := map[string]string{"question": question}
	if repos != "" {
		body["repos"] = repos
	}
	if adapter != "" {
		body["adapter"] = adapter
	}

	// Submit the question
	var submitResp struct {
		Success bool   `json:"success"`
		AskID   string `json:"askId"`
		Status  string `json:"status"`
		Error   string `json:"error"`
	}

	if !flagJSON && !flagAgent {
		fmt.Printf("  %s Submitting question to askbox...\r", output.Dim("⏳"))
	}

	err := client.Post(ctx(), "/ask/arch", body, &submitResp)
	if err != nil {
		if flagJSON {
			return output.JSON(map[string]any{"success": false, "error": err.Error()})
		}
		return fmt.Errorf("ask failed: %w", err)
	}

	if !submitResp.Success {
		if flagJSON {
			return output.JSON(map[string]any{"success": false, "error": submitResp.Error})
		}
		return fmt.Errorf("ask failed: %s", submitResp.Error)
	}

	askID := submitResp.AskID

	// --no-wait: return immediately with the ask-id
	if noWait {
		if flagJSON {
			return output.JSON(map[string]any{
				"success": true,
				"askId":   askID,
				"status":  "pending",
			})
		}
		// --for-agent: plain text ask-id
		if flagAgent {
			fmt.Printf("ask-id: %s\nstatus: pending\n", askID)
			return nil
		}
		fmt.Printf("  %s Submitted — ask-id: %s (use reposwarm ask status %s to check)\n",
			output.Green("✓"), askID, askID)
		return nil
	}

	if !flagJSON && !flagAgent {
		fmt.Printf("\r\033[K  %s Submitted — ask-id: %s\n", output.Green("✓"), askID)
	}
	if flagAgent {
		fmt.Fprintf(os.Stderr, "ask-id: %s\nstatus: polling\n", askID)
	}

	// Poll for completion
	for {
		var pollResp struct {
			Success     bool   `json:"success"`
			AskID       string `json:"askId"`
			Status      string `json:"status"`
			Detail      string `json:"detail"`
			Answer      string `json:"answer"`
			DownloadURL string `json:"downloadUrl"`
			Error       string `json:"error"`
			Chars       int    `json:"chars"`
		}

		err := client.Get(ctx(), fmt.Sprintf("/ask/arch/%s", askID), &pollResp)
		if err != nil {
			if flagJSON {
				return output.JSON(map[string]any{
					"success": false,
					"askId":   askID,
					"error":   err.Error(),
				})
			}
			return fmt.Errorf("polling failed: %w", err)
		}

		switch pollResp.Status {
		case "completed":
			if flagJSON {
				return output.JSON(map[string]any{
					"success": true,
					"askId":   askID,
					"answer":  pollResp.Answer,
					"chars":   pollResp.Chars,
				})
			}
			if flagAgent {
				fmt.Print(pollResp.Answer)
				return nil
			}
			fmt.Printf("\r\033[K  %s Answer ready (%d chars)\n\n", output.Green("✓"), pollResp.Chars)
			fmt.Println(pollResp.Answer)
			return nil

		case "failed":
			if flagJSON {
				return output.JSON(map[string]any{
					"success": false,
					"askId":   askID,
					"status":  "failed",
					"error":   pollResp.Error,
				})
			}
			return fmt.Errorf("ask failed: %s", pollResp.Error)

		default:
			if !flagJSON && !flagAgent {
				detail := pollResp.Detail
				if detail == "" {
					detail = pollResp.Status
				}
				fmt.Printf("\r\033[K  %s %s", output.Dim("⠋"), detail)
			}
			time.Sleep(3 * time.Second)
		}
	}
}
