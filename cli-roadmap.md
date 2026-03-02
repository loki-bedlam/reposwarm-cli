
---

## Provider & Model Configuration (Planned)

### Context
The RepoSwarm worker uses Claude (via Anthropic API or Bedrock) for LLM calls during investigations. Currently only supports Anthropic direct API (`ANTHROPIC_API_KEY`). Need to add Bedrock as an alternative provider.

### Three Layers
1. **Provider** — Anthropic direct API vs Amazon Bedrock
2. **Model** — which Claude model (opus, sonnet, haiku, or full model ID)
3. **Credentials** — API key (Anthropic) or AWS credentials (Bedrock)

### Bedrock Required Env Vars
- `CLAUDE_CODE_USE_BEDROCK=1` — enables Bedrock mode
- `AWS_REGION` — required (Bedrock doesn't read .aws config)
- `ANTHROPIC_MODEL` — model ID (e.g. `us.anthropic.claude-sonnet-4-6`)
- `ANTHROPIC_SMALL_FAST_MODEL` — optional fast/small model
- `ANTHROPIC_DEFAULT_OPUS_MODEL` / `ANTHROPIC_DEFAULT_SONNET_MODEL` / `ANTHROPIC_DEFAULT_HAIKU_MODEL` — pin versions
- AWS creds via standard chain (env vars, instance profile, SSO)

### Commands

#### `reposwarm config provider`
```bash
reposwarm config provider setup           # Interactive: provider, model, creds
reposwarm config provider set <provider>   # Quick switch (bedrock|anthropic)
reposwarm config provider show             # Current config + validation
```

`provider setup` interactive flow:
- Asks: provider → region (Bedrock) → model alias → pin versions?
- Writes both config.json AND worker .env
- Non-interactive: `--provider bedrock --region us-east-1 --model opus --pin`

#### `reposwarm config model`
```bash
reposwarm config model set <alias|id>      # Set model (auto-resolves per provider)
reposwarm config model pin                 # Pin all 3 aliases to current versions
reposwarm config model show                # Config across CLI/server/worker
reposwarm config model list                # Available models for provider
```

### Model Alias Resolution

| Alias   | Anthropic API         | Bedrock                                          |
|---------|-----------------------|--------------------------------------------------|
| sonnet  | claude-sonnet-4-6     | us.anthropic.claude-sonnet-4-6                   |
| opus    | claude-opus-4-6       | us.anthropic.claude-opus-4-6-v1                  |
| haiku   | claude-haiku-4-5      | us.anthropic.claude-haiku-4-5-20251001-v1:0      |

### Config Changes

config.json adds:
```json
{
  "provider": "bedrock",
  "awsRegion": "us-east-1",
  "smallModel": "us.anthropic.claude-haiku-4-5-20251001-v1:0",
  "modelPins": {
    "opus": "us.anthropic.claude-opus-4-6-v1",
    "sonnet": "us.anthropic.claude-sonnet-4-6",
    "haiku": "us.anthropic.claude-haiku-4-5-20251001-v1:0"
  }
}
```

### Doctor/Preflight Enhancements
- Check provider-specific creds (Bedrock → AWS; Anthropic → API key)
- Validate model ID format matches provider
- Test Bedrock access with `aws bedrock list-inference-profiles`
- Warn on unpinned aliases with Bedrock
