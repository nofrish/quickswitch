package config

import (
	_ "embed"
	"encoding/json"
	"fmt"
)

//go:embed providers.json
var providersJSON []byte

type Provider struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	BaseURL string            `json:"base_url"`
	Env     map[string]string `json:"env"`
}

type providersFile struct {
	Providers []Provider `json:"providers"`
}

// LoadProviders returns all preset providers embedded in the binary.
func LoadProviders() ([]Provider, error) {
	var f providersFile
	if err := json.Unmarshal(providersJSON, &f); err != nil {
		return nil, fmt.Errorf("failed to parse providers.json: %w", err)
	}
	return f.Providers, nil
}

// FindProvider returns the provider with the given id, or false if not found.
func FindProvider(id string) (Provider, bool) {
	providers, err := LoadProviders()
	if err != nil {
		return Provider{}, false
	}
	for _, p := range providers {
		if p.ID == id {
			return p, true
		}
	}
	return Provider{}, false
}
