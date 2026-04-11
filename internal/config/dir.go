package config

import (
	"os"
	"path/filepath"
)

// SupportedTools is the list of tools quickswitch can manage.
var SupportedTools = []string{"claude", "codex"}

// ToolDir returns the config directory for a given tool.
// e.g. ~/.config/quickswitch/claude
func ToolDir(tool string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "quickswitch", tool), nil
}

// EnsureToolDir creates the config directory for a given tool if it does not exist.
func EnsureToolDir(tool string) (string, error) {
	dir, err := ToolDir(tool)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return "", err
	}
	return dir, nil
}

// EnsureClaudeDir is a convenience wrapper for EnsureToolDir("claude").
func EnsureClaudeDir() (string, error) {
	return EnsureToolDir("claude")
}
