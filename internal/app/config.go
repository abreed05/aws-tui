package app

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Config holds application configuration
type Config struct {
	// AWS settings
	DefaultProfile string `yaml:"default_profile"`
	DefaultRegion  string `yaml:"default_region"`

	// UI settings
	Theme          string `yaml:"theme"`
	ShowHelp       bool   `yaml:"show_help"`
	RefreshSeconds int    `yaml:"refresh_seconds"`

	// Paths
	ConfigDir string `yaml:"-"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	configDir := filepath.Join(homeDir, ".config", "aws-tui")

	// Check for AWS_PROFILE environment variable
	defaultProfile := os.Getenv("AWS_PROFILE")
	if defaultProfile == "" {
		defaultProfile = "default"
	}

	// Check for AWS_REGION or AWS_DEFAULT_REGION environment variable
	defaultRegion := os.Getenv("AWS_REGION")
	if defaultRegion == "" {
		defaultRegion = os.Getenv("AWS_DEFAULT_REGION")
	}
	if defaultRegion == "" {
		defaultRegion = "us-east-1"
	}

	return &Config{
		DefaultProfile: defaultProfile,
		DefaultRegion:  defaultRegion,
		Theme:          "default",
		ShowHelp:       true,
		RefreshSeconds: 30,
		ConfigDir:      configDir,
	}
}

// LoadConfig loads configuration from file or returns defaults
func LoadConfig() (*Config, error) {
	cfg := DefaultConfig()

	// Ensure config directory exists
	if err := os.MkdirAll(cfg.ConfigDir, 0755); err != nil {
		return nil, err
	}

	configPath := filepath.Join(cfg.ConfigDir, "config.yaml")

	// Try to read existing config
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			// No config file yet, use defaults and save them
			if saveErr := cfg.Save(); saveErr != nil {
				// Non-fatal, just continue with defaults
			}
			return cfg, nil
		}
		return nil, err
	}

	// Parse config file
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save persists the configuration to file
func (c *Config) Save() error {
	if err := os.MkdirAll(c.ConfigDir, 0755); err != nil {
		return err
	}

	configPath := filepath.Join(c.ConfigDir, "config.yaml")

	data, err := yaml.Marshal(c)
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}

// HistoryPath returns the path to the command history file
func (c *Config) HistoryPath() string {
	return filepath.Join(c.ConfigDir, "history")
}

// BookmarksPath returns the path to the bookmarks file
func (c *Config) BookmarksPath() string {
	return filepath.Join(c.ConfigDir, "bookmarks.yaml")
}

// ThemesPath returns the path to the themes directory
func (c *Config) ThemesPath() string {
	return filepath.Join(c.ConfigDir, "themes")
}
