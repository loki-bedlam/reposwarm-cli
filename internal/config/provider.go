package config

// Provider represents an LLM provider backend.
type Provider string

const (
	ProviderAnthropic Provider = "anthropic"
	ProviderBedrock   Provider = "bedrock"
	ProviderLiteLLM   Provider = "litellm"
)

// ValidProviders returns all supported provider names.
func ValidProviders() []string {
	return []string{string(ProviderAnthropic), string(ProviderBedrock), string(ProviderLiteLLM)}
}

// IsValidProvider returns true if the provider name is known.
func IsValidProvider(p string) bool {
	switch Provider(p) {
	case ProviderAnthropic, ProviderBedrock, ProviderLiteLLM:
		return true
	}
	return false
}

// ProviderConfig holds provider-specific configuration.
type ProviderConfig struct {
	Provider   Provider          `json:"provider,omitempty"`
	AWSRegion  string            `json:"awsRegion,omitempty"`
	ProxyURL   string            `json:"proxyUrl,omitempty"`    // LiteLLM proxy URL
	ProxyKey   string            `json:"proxyKey,omitempty"`    // LiteLLM proxy API key
	SmallModel string            `json:"smallModel,omitempty"`  // Fast/cheap model for triage
	ModelPins  map[string]string `json:"modelPins,omitempty"`   // alias → pinned model ID
}

// ModelAlias maps human-friendly names to provider-specific model IDs.
type ModelAlias struct {
	Alias     string
	Anthropic string
	Bedrock   string
}

// KnownAliases returns the standard model alias table.
func KnownAliases() []ModelAlias {
	return []ModelAlias{
		{"sonnet", "claude-sonnet-4-6", "us.anthropic.claude-sonnet-4-6"},
		{"opus", "claude-opus-4-6", "us.anthropic.claude-opus-4-6-v1"},
		{"haiku", "claude-haiku-4-5", "us.anthropic.claude-haiku-4-5-20251001-v1:0"},
		{"sonnet-3.5", "claude-3-5-sonnet-20241022", "us.anthropic.claude-3-5-sonnet-20241022-v2:0"},
	}
}

// ResolveModel takes an alias or raw model ID and returns the provider-specific model ID.
// If modelPins has a pin for this alias, use it. Otherwise resolve from the alias table.
func ResolveModel(alias string, provider Provider, pins map[string]string) string {
	// Check pins first
	if pins != nil {
		if pinned, ok := pins[alias]; ok {
			return pinned
		}
	}

	// Check alias table
	for _, a := range KnownAliases() {
		if a.Alias == alias {
			switch provider {
			case ProviderBedrock:
				return a.Bedrock
			case ProviderAnthropic, ProviderLiteLLM:
				return a.Anthropic
			}
		}
	}

	// Not an alias — return as-is (raw model ID)
	return alias
}

// DefaultSmallModel returns the default small/fast model for a provider.
func DefaultSmallModel(provider Provider) string {
	switch provider {
	case ProviderBedrock:
		return "us.anthropic.claude-haiku-4-5-20251001-v1:0"
	default:
		return "claude-haiku-4-5"
	}
}

// WorkerEnvVars returns the env vars the worker needs for a given provider config.
func WorkerEnvVars(pc *ProviderConfig, model string) map[string]string {
	vars := map[string]string{}

	resolved := ResolveModel(model, pc.Provider, pc.ModelPins)
	smallResolved := pc.SmallModel
	if smallResolved == "" {
		smallResolved = DefaultSmallModel(pc.Provider)
	}

	switch pc.Provider {
	case ProviderBedrock:
		vars["CLAUDE_CODE_USE_BEDROCK"] = "1"
		vars["AWS_REGION"] = pc.AWSRegion
		if pc.AWSRegion == "" {
			vars["AWS_REGION"] = "us-east-1"
		}
		vars["ANTHROPIC_MODEL"] = resolved
		vars["ANTHROPIC_SMALL_FAST_MODEL"] = smallResolved

		// Set version pins if available
		if pc.ModelPins != nil {
			if v, ok := pc.ModelPins["opus"]; ok {
				vars["ANTHROPIC_DEFAULT_OPUS_MODEL"] = v
			}
			if v, ok := pc.ModelPins["sonnet"]; ok {
				vars["ANTHROPIC_DEFAULT_SONNET_MODEL"] = v
			}
			if v, ok := pc.ModelPins["haiku"]; ok {
				vars["ANTHROPIC_DEFAULT_HAIKU_MODEL"] = v
			}
		}

	case ProviderLiteLLM:
		// LiteLLM proxy uses standard Anthropic SDK format but routes through proxy
		vars["ANTHROPIC_API_KEY"] = pc.ProxyKey
		vars["ANTHROPIC_BASE_URL"] = pc.ProxyURL
		vars["ANTHROPIC_MODEL"] = resolved

	case ProviderAnthropic:
		// Standard Anthropic — API key should already be set
		vars["ANTHROPIC_MODEL"] = resolved
		vars["ANTHROPIC_SMALL_FAST_MODEL"] = smallResolved
	}

	return vars
}
