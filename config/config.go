package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Config represents user preferences for Ganymede CLI
type Config struct {
	ColorMode string `json:"color_mode"` // "hex", "rgb", "cmyk"
	Hotkey    string `json:"hotkey"`     // "win+alt+p", "ctrl+alt+c", "alt+shift+x"
	Startup   bool   `json:"startup"`    // true or false
	Theme     string `json:"theme"`      // "ganymede", "rainbow", "cyberpunk", etc.
	Sound     bool   `json:"sound"`      // audio chime on copy
}

// DefaultConfig returns standard recommended defaults
func DefaultConfig() *Config {
	return &Config{
		ColorMode: "hex",
		Hotkey:    "win+alt+p",
		Startup:   true,
		Theme:     "ganymede",
		Sound:     true,
	}
}

// GetConfigDir returns the directory where ganymede configuration is stored
func GetConfigDir() (string, error) {
	appData := os.Getenv("APPDATA")
	if appData != "" {
		return filepath.Join(appData, "ganymede"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".ganymede"), nil
}

// GetConfigPath returns the full path to config.json
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Exists checks if configuration file exists
func Exists() bool {
	p, err := GetConfigPath()
	if err != nil {
		return false
	}
	info, err := os.Stat(p)
	return err == nil && !info.IsDir()
}

// Load reads and parses config.json, returning DefaultConfig if not found
func Load() (*Config, error) {
	p, err := GetConfigPath()
	if err != nil {
		return DefaultConfig(), err
	}

	data, err := os.ReadFile(p)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config: %w", err)
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return DefaultConfig(), fmt.Errorf("failed to parse config JSON: %w", err)
	}

	// Normalize
	cfg.ColorMode = strings.ToLower(strings.TrimSpace(cfg.ColorMode))
	if cfg.ColorMode != "hex" && cfg.ColorMode != "rgb" && cfg.ColorMode != "cmyk" {
		cfg.ColorMode = "hex"
	}
	cfg.Hotkey = strings.ToLower(strings.TrimSpace(cfg.Hotkey))
	if cfg.Hotkey == "" {
		cfg.Hotkey = "win+alt+p"
	}
	if cfg.Theme == "" {
		cfg.Theme = "ganymede"
	}

	return cfg, nil
}

// Save writes config to disk
func Save(cfg *Config) error {
	dir, err := GetConfigDir()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config dir: %w", err)
	}

	p := filepath.Join(dir, "config.json")
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to serialize config: %w", err)
	}

	if err := os.WriteFile(p, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}
	return nil
}
