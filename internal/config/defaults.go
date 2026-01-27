package config

const (
	// DefaultAutoPruneTokenLimit is the default token count before pruning triggers.
	// Gemini 1.5 Pro/Flash has a large context, but we want to be economical.
	// 1,000,000 is generous.
	DefaultAutoPruneTokenLimit = 1000000

	// DefaultAutoCompact is disabled by default.
	DefaultAutoCompact = false
)
