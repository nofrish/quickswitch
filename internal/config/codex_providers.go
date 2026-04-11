package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed codex_providers.json
var codexProvidersJSON []byte

type CodexProvider struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	BaseURL string `json:"base_url"`
	WireAPI string `json:"wire_api"`
}

type codexProvidersFile struct {
	Providers []CodexProvider `json:"providers"`
}

// LoadCodexProviders returns all preset Codex providers embedded in the binary.
func LoadCodexProviders() ([]CodexProvider, error) {
	var f codexProvidersFile
	if err := json.Unmarshal(codexProvidersJSON, &f); err != nil {
		return nil, fmt.Errorf("failed to parse codex_providers.json: %w", err)
	}
	return f.Providers, nil
}

// FindCodexProvider returns the provider with the given id, or false if not found.
func FindCodexProvider(id string) (CodexProvider, bool) {
	providers, err := LoadCodexProviders()
	if err != nil {
		return CodexProvider{}, false
	}
	for _, p := range providers {
		if p.ID == id {
			return p, true
		}
	}
	return CodexProvider{}, false
}
