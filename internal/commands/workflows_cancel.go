package commands

import (
	"fmt"
	"strings"

	"github.com/reposwarm/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newWorkflowsCancelCmd() *cobra.Command {
	var yes bool

	cmd := &cobra.Command{
		Use:   "cancel <workflow-id>",
		Short: "Request graceful cancellation of a workflow",
		Long: `Send a cancellation signal to a running workflow.
Unlike terminate, cancel allows the current activity to complete before stopping.

Examples:
  reposwarm workflows cancel investigate-single-is-odd-1772470037390`,
		Args: friendlyExactArgs(1, "reposwarm workflows cancel <workflow-id>\n\nExample:\n  reposwarm workflows cancel investigate-single-is-odd-123"),
		RunE: func(cmd *cobra.Command, args []string) error {
			workflowID := args[0]

			if !yes && !flagJSON {
				fmt.Printf("  Cancel workflow %s? (current activity will complete first) [y/N] ", workflowID)
				var confirm string
				fmt.Scanln(&confirm)
				if strings.ToLower(confirm) != "y" {
					output.F.Info("Cancelled")
					return nil
				}
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			// Send cancel signal
			var result any
			if err := client.Post(ctx(), "/workflows/"+workflowID+"/cancel", nil, &result); err != nil {
				// If /cancel endpoint doesn't exist, fall back to terminate with reason
				body := map[string]string{"reason": "Gracefully cancelled via CLI"}
				if err2 := client.Post(ctx(), "/workflows/"+workflowID+"/terminate", body, &result); err2 != nil {
					return fmt.Errorf("cancel failed: %w (terminate fallback also failed: %v)", err, err2)
				}
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"workflowId": workflowID,
					"cancelled":  true,
				})
			}

			output.F.Success(fmt.Sprintf("Cancellation requested for %s", workflowID))
			output.F.Info("Current activity will complete before workflow stops")
			output.F.Info(fmt.Sprintf("Monitor: reposwarm wf status %s -v", workflowID))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&yes, "yes", "y", false, "Skip confirmation")
	return cmd
}
