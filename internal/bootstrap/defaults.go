// Package bootstrap — defaults.go centralizes all environment-specific defaults.
// These should be the only place repo URLs, table names, and model IDs are defined.
// Override via environment variables or CLI flags, never hardcode elsewhere.
package bootstrap

const (
	// Git repositories
	DefaultWorkerRepoURL = "https://github.com/royosherove/repo-swarm.git"
	DefaultAPIRepoURL    = "https://github.com/loki-bedlam/reposwarm-api.git"
	DefaultUIRepoURL     = "https://github.com/loki-bedlam/reposwarm-ui.git"

	// DynamoDB
	DefaultDynamoDBTable = "reposwarm-cache"

	// AI model
	DefaultModel = "us.anthropic.claude-sonnet-4-6"

	// Local ports
	DefaultTemporalPort   = "7233"
	DefaultTemporalUIPort = "8233"
	DefaultAPIPort        = "3000"
	DefaultUIPort         = "3001"
)
