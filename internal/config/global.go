package config

import (
	"fmt"
	"os"
	"path/filepath"
)

// ApplyClaudeGlobalProfile writes the selected quickswitch Claude profile to
// Claude Code's official default settings file at ~/.claude/settings.json.
func ApplyClaudeGlobalProfile(claudeDir string, profile EnvProfile) (string, error) {
	settings, err := loadDefaultSettings(claudeDir)
	if err != nil {
		return "", err
	}

	mergeEnvProfile(settings, profile)

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	dstDir := filepath.Join(home, ".claude")
	if err := os.MkdirAll(dstDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create ~/.claude: %w", err)
	}

	if err := writeSettings(dstDir, settings); err != nil {
		return "", err
	}

	return filepath.Join(dstDir, "settings.json"), nil
}

// ApplyCodexGlobalProfile writes the selected quickswitch Codex profile to
// Codex's official default config files at ~/.codex/config.toml and auth.json.
func ApplyCodexGlobalProfile(codexDir, profileName string, profile EnvProfile) (configPath string, authPath string, err error) {
	cfg, err := loadCodexDefaultConfig(codexDir)
	if err != nil {
		return "", "", err
	}

	injectCodexProfile(cfg, profileName, profile)

	home, err := os.UserHomeDir()
	if err != nil {
		return "", "", err
	}

	dstDir := filepath.Join(home, ".codex")
	if err := os.MkdirAll(dstDir, 0700); err != nil {
		return "", "", fmt.Errorf("failed to create ~/.codex: %w", err)
	}

	if err := writeCodexConfig(dstDir, cfg); err != nil {
		return "", "", err
	}
	if err := writeCodexAuth(dstDir, profile); err != nil {
		return "", "", err
	}

	return filepath.Join(dstDir, "config.toml"), filepath.Join(dstDir, "auth.json"), nil
}
