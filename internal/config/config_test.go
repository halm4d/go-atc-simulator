package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.InputMode != "keyboard" {
		t.Errorf("expected default InputMode 'keyboard', got %q", cfg.InputMode)
	}
	if cfg.Ollama.Enabled != false {
		t.Error("expected Ollama disabled by default")
	}
	if cfg.Ollama.Endpoint != "http://localhost:11434" {
		t.Errorf("unexpected default Ollama endpoint: %s", cfg.Ollama.Endpoint)
	}
	if cfg.Ollama.Model != "llama3.2:1b" {
		t.Errorf("unexpected default Ollama model: %s", cfg.Ollama.Model)
	}
}

func TestSaveLoadRoundtrip(t *testing.T) {
	// Use a temp dir to avoid polluting real config
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "config.json")

	cfg := DefaultConfig()
	cfg.InputMode = "chat"
	cfg.Ollama.Enabled = true

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}

	loaded := DefaultConfig()
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(raw, &loaded); err != nil {
		t.Fatal(err)
	}

	if loaded.InputMode != "chat" {
		t.Errorf("expected InputMode 'chat', got %q", loaded.InputMode)
	}
	if !loaded.Ollama.Enabled {
		t.Error("expected Ollama enabled after load")
	}
}
