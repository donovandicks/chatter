package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Config holds the application configuration.
type Config struct {
	// Threshold of tokens before automatic pruning triggers
	AutoPruneTokenLimit int `json:"auto_prune_token_limit"`
	// Whether to auto-compact the chat when a threshold is reached (future)
	AutoCompact bool `json:"auto_compact"`
}

// Load reads the configuration from the default path or creates it if it doesn't exist.
func Load() (*Config, error) {
	configDir, err := getConfigDir()
	if err != nil {
		return nil, err
	}

	configPath := filepath.Join(configDir, "settings.json")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return createDefaultConfig(configPath)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply overrides or defaults for missing fields if necessary
	if cfg.AutoPruneTokenLimit == 0 {
		cfg.AutoPruneTokenLimit = DefaultAutoPruneTokenLimit
	}

	return &cfg, nil
}

// Save writes the configuration to the default path.
func (c *Config) Save() error {
	configDir, err := getConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(configDir, 0o755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	configPath := filepath.Join(configDir, "settings.json")
	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

func getConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user home directory: %w", err)
	}
	return filepath.Join(home, ".chatter"), nil
}

func createDefaultConfig(path string) (*Config, error) {
	cfg := &Config{
		AutoPruneTokenLimit: DefaultAutoPruneTokenLimit,
		AutoCompact:         DefaultAutoCompact,
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal default config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil {
		return nil, fmt.Errorf("failed to write default config file: %w", err)
	}

	return cfg, nil
}
