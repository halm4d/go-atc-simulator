package config

import (
	"atc-sim/internal/logger"
	"encoding/json"
	"os"
	"path/filepath"
)

type OllamaConfig struct {
	Endpoint string `json:"endpoint"`
	Model    string `json:"model"`
}

type Config struct {
	InputMode string       `json:"inputMode"`
	Ollama    OllamaConfig `json:"ollama"`
}

func DefaultConfig() Config {
	return Config{
		InputMode: "keyboard",
		Ollama: OllamaConfig{
			Endpoint: "http://localhost:11434",
			Model:    "qwen2.5:0.5b",
		},
	}
}

// configDir returns the path to the atc-sim config directory.
func configDir() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		// Fallback to working directory
		return ".", nil
	}
	return filepath.Join(dir, "atc-sim"), nil
}

// configPath returns the full path to config.json.
func configPath() (string, error) {
	dir, err := configDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// Load reads config from disk, returning defaults if file doesn't exist.
func Load() Config {
	path, err := configPath()
	if err != nil {
		logger.Warn("config path error, using defaults", "error", err)
		return DefaultConfig()
	}
	data, err := os.ReadFile(path)
	if err != nil {
		logger.Info("no config file found, using defaults", "path", path)
		return DefaultConfig()
	}
	cfg := DefaultConfig()
	if err := json.Unmarshal(data, &cfg); err != nil {
		logger.Error("failed to parse config, using defaults", "path", path, "error", err)
		return DefaultConfig()
	}
	logger.Info("config loaded from file", "path", path)
	return cfg
}

// Save writes config to disk.
func Save(cfg Config) error {
	path, err := configPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}
