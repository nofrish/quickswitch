package config

import (
	_ "embed"
	"os"
	"path/filepath"
)

//go:embed claude_default_settings.json
var claudeDefaultSettingsJSON []byte

func scaffoldClaudeDefaultSettings(dst string) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0700); err != nil {
		return err
	}
	return os.WriteFile(dst, claudeDefaultSettingsJSON, 0600)
}
