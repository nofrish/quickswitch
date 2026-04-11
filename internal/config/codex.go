package config

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

// BuildCodexRuntime creates a per-profile runtime directory for Codex and returns its path.
//
// It reads ~/.config/quickswitch/codex/default-config.toml (shared settings: tui, projects, model, etc.),
// injects the profile's provider config into [model_providers.<profileName>], writes the merged
// result as config.toml, writes auth.json with the OPENAI_API_KEY, and symlinks all shared
// data directories back to ~/.codex/ so history and sessions are shared across profiles.
func BuildCodexRuntime(codexDir, profileName string, profile EnvProfile) (string, error) {
	if profileName == "" {
		profileName = "default"
	}

	cfg, err := loadCodexDefaultConfig(codexDir)
	if err != nil {
		return "", err
	}

	injectCodexProfile(cfg, profileName, profile)

	runtimeDir := filepath.Join(codexDir, "runtime", profileName)
	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create runtime directory: %w", err)
	}

	if err := writeCodexConfig(runtimeDir, cfg); err != nil {
		return "", err
	}

	if err := writeCodexAuth(runtimeDir, profile); err != nil {
		return "", err
	}

	if err := symlinkCodexSharedData(runtimeDir); err != nil {
		return "", err
	}

	return runtimeDir, nil
}

// perProfileCodexFiles are files quickswitch manages per-profile. Everything else
// in ~/.codex/ is symlinked into the runtime directory so data is shared.
var perProfileCodexFiles = map[string]bool{
	"config.toml": true,
	"auth.json":   true,
}

func symlinkCodexSharedData(runtimeDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	codexHome := filepath.Join(home, ".codex")

	entries, err := os.ReadDir(codexHome)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read ~/.codex: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if perProfileCodexFiles[name] {
			continue
		}

		src := filepath.Join(codexHome, name)
		dst := filepath.Join(runtimeDir, name)

		if _, err := os.Lstat(dst); err == nil {
			continue
		}

		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("failed to symlink %s: %w", name, err)
		}
	}

	return nil
}

func loadCodexDefaultConfig(codexDir string) (map[string]interface{}, error) {
	src := filepath.Join(codexDir, "default-config.toml")
	data, err := os.ReadFile(src)
	if os.IsNotExist(err) {
		return make(map[string]interface{}), nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to read default-config.toml: %w", err)
	}

	var cfg map[string]interface{}
	if _, err := toml.Decode(string(data), &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse default-config.toml: %w", err)
	}
	return cfg, nil
}

// injectCodexProfile sets model_provider and populates [model_providers.<profileName>]
// with the API endpoint config from the profile's stored values.
func injectCodexProfile(cfg map[string]interface{}, profileName string, profile EnvProfile) {
	cfg["model_provider"] = profileName

	providers, _ := cfg["model_providers"].(map[string]interface{})
	if providers == nil {
		providers = make(map[string]interface{})
	}

	providerCfg := map[string]interface{}{
		"name":                 profileName,
		"requires_openai_auth": true,
		"wire_api":             "responses",
	}
	if baseURL, ok := profile["base_url"]; ok {
		providerCfg["base_url"] = baseURL
	}
	if wireAPI, ok := profile["wire_api"]; ok {
		providerCfg["wire_api"] = wireAPI
	}

	providers[profileName] = providerCfg
	cfg["model_providers"] = providers
}

func writeCodexConfig(runtimeDir string, cfg map[string]interface{}) error {
	var buf bytes.Buffer
	if err := toml.NewEncoder(&buf).Encode(cfg); err != nil {
		return fmt.Errorf("failed to encode config.toml: %w", err)
	}
	dst := filepath.Join(runtimeDir, "config.toml")
	if err := os.WriteFile(dst, buf.Bytes(), 0600); err != nil {
		return fmt.Errorf("failed to write config.toml: %w", err)
	}
	return nil
}

func writeCodexAuth(runtimeDir string, profile EnvProfile) error {
	auth := map[string]string{
		"OPENAI_API_KEY": profile["OPENAI_API_KEY"],
	}
	data, err := json.MarshalIndent(auth, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal auth.json: %w", err)
	}
	dst := filepath.Join(runtimeDir, "auth.json")
	if err := os.WriteFile(dst, data, 0600); err != nil {
		return fmt.Errorf("failed to write auth.json: %w", err)
	}
	return nil
}
