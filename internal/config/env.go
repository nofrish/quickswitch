package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// EnvProfile holds the environment variables for a single profile.
type EnvProfile map[string]string

// EnvConfig holds all profiles for a tool (e.g. Claude).
type EnvConfig struct {
	Profiles map[string]EnvProfile `json:"profiles"`
}

func envConfigPath(claudeDir string) string {
	return filepath.Join(claudeDir, "env.json")
}

// LoadEnvConfig reads env.json from the given Claude config directory.
func LoadEnvConfig(claudeDir string) (*EnvConfig, error) {
	data, err := os.ReadFile(envConfigPath(claudeDir))
	if err != nil {
		return nil, err
	}
	var cfg EnvConfig
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

// SaveEnvConfig writes env.json to the given Claude config directory.
func SaveEnvConfig(claudeDir string, cfg *EnvConfig) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(envConfigPath(claudeDir), data, 0600)
}
