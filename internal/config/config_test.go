package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadAndRoute(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := `
server:
  addr: ":9090"
providers:
  - name: openai
    base_url: https://api.openai.com/v1
    api_key: ${TEST_OPENAI_KEY}
    models:
      - gpt-4o-mini
  - name: ollama
    base_url: http://127.0.0.1:11434/v1
    models:
      - llama3
routing:
  - model: gpt-4o-mini
    provider: openai
    fallback: ollama
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TEST_OPENAI_KEY", "sk-test")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if cfg.Server.Addr != ":9090" {
		t.Fatalf("addr = %q, want :9090", cfg.Server.Addr)
	}

	p, fallback, err := cfg.ProviderForModel("gpt-4o-mini")
	if err != nil {
		t.Fatalf("ProviderForModel() error = %v", err)
	}
	if p.Name != "openai" {
		t.Fatalf("provider = %q, want openai", p.Name)
	}
	if fallback != "ollama" {
		t.Fatalf("fallback = %q, want ollama", fallback)
	}

	p, _, err = cfg.ProviderForModel("llama3")
	if err != nil {
		t.Fatalf("ProviderForModel(llama3) error = %v", err)
	}
	if p.Name != "ollama" {
		t.Fatalf("provider = %q, want ollama", p.Name)
	}
}
