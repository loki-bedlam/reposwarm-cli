package config

import "testing"

func TestResolveModelAliases(t *testing.T) {
	tests := []struct {
		alias    string
		provider Provider
		pins     map[string]string
		want     string
	}{
		// Anthropic aliases
		{"sonnet", ProviderAnthropic, nil, "claude-sonnet-4-6"},
		{"opus", ProviderAnthropic, nil, "claude-opus-4-6"},
		{"haiku", ProviderAnthropic, nil, "claude-haiku-4-5"},

		// Bedrock aliases
		{"sonnet", ProviderBedrock, nil, "us.anthropic.claude-sonnet-4-6"},
		{"opus", ProviderBedrock, nil, "us.anthropic.claude-opus-4-6-v1"},
		{"haiku", ProviderBedrock, nil, "us.anthropic.claude-haiku-4-5-20251001-v1:0"},

		// LiteLLM uses Anthropic IDs
		{"sonnet", ProviderLiteLLM, nil, "claude-sonnet-4-6"},
		{"opus", ProviderLiteLLM, nil, "claude-opus-4-6"},

		// Raw model ID (not an alias) passes through unchanged
		{"us.anthropic.claude-opus-4-6-v1", ProviderAnthropic, nil, "us.anthropic.claude-opus-4-6-v1"},
		{"custom-model-id", ProviderBedrock, nil, "custom-model-id"},

		// Pinned model overrides alias
		{"sonnet", ProviderBedrock, map[string]string{"sonnet": "us.anthropic.claude-sonnet-4-20250514-v1:0"}, "us.anthropic.claude-sonnet-4-20250514-v1:0"},
		{"opus", ProviderAnthropic, map[string]string{"opus": "claude-opus-4-20250514"}, "claude-opus-4-20250514"},
	}

	for _, tt := range tests {
		got := ResolveModel(tt.alias, tt.provider, tt.pins)
		if got != tt.want {
			t.Errorf("ResolveModel(%q, %q, %v) = %q; want %q", tt.alias, tt.provider, tt.pins, got, tt.want)
		}
	}
}

func TestWorkerEnvVars(t *testing.T) {
	t.Run("Bedrock", func(t *testing.T) {
		pc := &ProviderConfig{
			Provider:  ProviderBedrock,
			AWSRegion: "us-west-2",
		}
		vars := WorkerEnvVars(pc, "opus")
		if vars["CLAUDE_CODE_USE_BEDROCK"] != "1" {
			t.Error("Expected CLAUDE_CODE_USE_BEDROCK=1")
		}
		if vars["AWS_REGION"] != "us-west-2" {
			t.Errorf("Expected AWS_REGION=us-west-2, got %s", vars["AWS_REGION"])
		}
		if vars["ANTHROPIC_MODEL"] != "us.anthropic.claude-opus-4-6-v1" {
			t.Errorf("Expected Bedrock opus model, got %s", vars["ANTHROPIC_MODEL"])
		}
	})

	t.Run("LiteLLM", func(t *testing.T) {
		pc := &ProviderConfig{
			Provider: ProviderLiteLLM,
			ProxyURL: "https://proxy.example.com",
			ProxyKey: "sk-proxy-123",
		}
		vars := WorkerEnvVars(pc, "sonnet")
		if vars["ANTHROPIC_BASE_URL"] != "https://proxy.example.com" {
			t.Errorf("Expected proxy URL, got %s", vars["ANTHROPIC_BASE_URL"])
		}
		if vars["ANTHROPIC_API_KEY"] != "sk-proxy-123" {
			t.Errorf("Expected proxy key, got %s", vars["ANTHROPIC_API_KEY"])
		}
		if vars["ANTHROPIC_MODEL"] != "claude-sonnet-4-6" {
			t.Errorf("Expected Anthropic-style model, got %s", vars["ANTHROPIC_MODEL"])
		}
	})

	t.Run("Anthropic", func(t *testing.T) {
		pc := &ProviderConfig{Provider: ProviderAnthropic}
		vars := WorkerEnvVars(pc, "haiku")
		if _, ok := vars["CLAUDE_CODE_USE_BEDROCK"]; ok {
			t.Error("Should not set CLAUDE_CODE_USE_BEDROCK for Anthropic")
		}
		if vars["ANTHROPIC_MODEL"] != "claude-haiku-4-5" {
			t.Errorf("Expected haiku model, got %s", vars["ANTHROPIC_MODEL"])
		}
	})

	t.Run("Bedrock with pins", func(t *testing.T) {
		pc := &ProviderConfig{
			Provider:  ProviderBedrock,
			AWSRegion: "us-east-1",
			ModelPins: map[string]string{
				"opus":   "us.anthropic.claude-opus-4-6-v1",
				"sonnet": "us.anthropic.claude-sonnet-4-6",
				"haiku":  "us.anthropic.claude-haiku-4-5-20251001-v1:0",
			},
		}
		vars := WorkerEnvVars(pc, "opus")
		if vars["ANTHROPIC_DEFAULT_OPUS_MODEL"] != "us.anthropic.claude-opus-4-6-v1" {
			t.Error("Missing pinned opus model")
		}
		if vars["ANTHROPIC_DEFAULT_SONNET_MODEL"] != "us.anthropic.claude-sonnet-4-6" {
			t.Error("Missing pinned sonnet model")
		}
	})
}

func TestIsValidProvider(t *testing.T) {
	if !IsValidProvider("anthropic") { t.Error("anthropic should be valid") }
	if !IsValidProvider("bedrock") { t.Error("bedrock should be valid") }
	if !IsValidProvider("litellm") { t.Error("litellm should be valid") }
	if IsValidProvider("openai") { t.Error("openai should be invalid") }
	if IsValidProvider("") { t.Error("empty should be invalid") }
}
