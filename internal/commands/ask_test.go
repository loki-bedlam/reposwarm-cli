package commands

import (
	"strings"
	"testing"
)

func TestAskCmdSuccess(t *testing.T) {
	_, cleanup := testServer(t, map[string]any{
		"/health": map[string]any{"status": "healthy", "version": "1.2.0"},
		"POST /ask": map[string]any{
			"success":  true,
			"answer":   "Use `reposwarm repos add <name> --url <url>` to add a repository.",
			"model":    "us.anthropic.claude-haiku-4-5",
			"latencyMs": 450,
		},
	})
	defer cleanup()

	out, err := runCmd(t, "ask", "how do I add a repo?", "--for-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "repos add") {
		t.Errorf("expected answer about repos add, got: %s", out)
	}
}

func TestAskCmdJSON(t *testing.T) {
	_, cleanup := testServer(t, map[string]any{
		"/health": map[string]any{"status": "healthy", "version": "1.2.0"},
		"POST /ask": map[string]any{
			"success":  true,
			"answer":   "Run `reposwarm doctor --fix` to auto-remediate issues.",
			"model":    "us.anthropic.claude-haiku-4-5",
			"latencyMs": 300,
		},
	})
	defer cleanup()

	out, err := runCmd(t, "ask", "what does doctor fix do?", "--json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, `"success":true`) && !strings.Contains(out, `"success": true`) {
		t.Errorf("expected success:true in JSON, got: %s", out)
	}
	if !strings.Contains(out, "doctor") {
		t.Errorf("expected 'doctor' in answer, got: %s", out)
	}
}

func TestAskCmdError(t *testing.T) {
	_, cleanup := testServer(t, map[string]any{
		"/health": map[string]any{"status": "healthy", "version": "1.2.0"},
		"POST /ask": map[string]any{
			"success": false,
			"error":   "No model configured",
			"hint":    "Run: reposwarm config provider setup",
		},
	})
	defer cleanup()

	_, err := runCmd(t, "ask", "test question")
	if err == nil {
		t.Fatal("expected error for failed inference")
	}
	if !strings.Contains(err.Error(), "No model configured") {
		t.Errorf("expected 'No model configured' error, got: %v", err)
	}
}

func TestAskCmdMultipleWords(t *testing.T) {
	_, cleanup := testServer(t, map[string]any{
		"/health": map[string]any{"status": "healthy", "version": "1.2.0"},
		"POST /ask": map[string]any{
			"success":  true,
			"answer":   "You can switch providers with config provider set.",
			"model":    "claude-haiku-4-5",
			"latencyMs": 200,
		},
	})
	defer cleanup()

	// Multiple args get joined
	out, err := runCmd(t, "ask", "how", "to", "switch", "providers", "--for-agent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "switch providers") {
		t.Errorf("expected answer about switching providers, got: %s", out)
	}
}

func TestAskCmdNoArgs(t *testing.T) {
	_, cleanup := testServer(t, map[string]any{
		"/health": map[string]any{"status": "healthy", "version": "1.2.0"},
	})
	defer cleanup()

	_, err := runCmd(t, "ask")
	if err == nil {
		t.Fatal("expected error with no arguments")
	}
}
