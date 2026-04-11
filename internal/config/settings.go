package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// BuildRuntimeSettings creates a per-profile settings.json in a dedicated runtime
// directory, then returns the directory path to use as CLAUDE_CONFIG_DIR.
//
// It reads default-settings.json, merges the profile's env vars into the "env"
// section, and writes the result to:
//
//	~/.config/quickswitch/claude/runtime/<profileName>/settings.json
//
// If profileName is empty, "default" is used as the directory name.
func BuildRuntimeSettings(claudeDir string, profileName string, profile EnvProfile) (string, error) {
	if profileName == "" {
		profileName = "default"
	}

	settings, err := loadDefaultSettings(claudeDir)
	if err != nil {
		return "", err
	}

	mergeEnvProfile(settings, profile)

	runtimeDir := filepath.Join(claudeDir, "runtime", profileName)
	if err := os.MkdirAll(runtimeDir, 0700); err != nil {
		return "", fmt.Errorf("failed to create runtime directory: %w", err)
	}

	if err := writeSettings(runtimeDir, settings); err != nil {
		return "", err
	}

	if err := copyPreferences(runtimeDir); err != nil {
		return "", err
	}

	if err := symlinkSharedData(runtimeDir); err != nil {
		return "", err
	}

	return runtimeDir, nil
}

// perProfileFiles are files that quickswitch manages per-profile and must NOT
// be symlinked back to ~/.claude/. Everything else in ~/.claude/ is shared.
var perProfileFiles = map[string]bool{
	"settings.json": true,
}

// symlinkSharedData scans ~/.claude/ and creates symlinks in the runtime
// directory for every entry that is not managed per-profile, so all profiles
// share the same sessions, history, plans, and other data automatically.
func symlinkSharedData(runtimeDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	claudeDir := filepath.Join(home, ".claude")

	entries, err := os.ReadDir(claudeDir)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read ~/.claude: %w", err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if perProfileFiles[name] {
			continue
		}

		src := filepath.Join(claudeDir, name)
		dst := filepath.Join(runtimeDir, name)

		// Skip if the symlink already exists.
		if _, err := os.Lstat(dst); err == nil {
			continue
		}

		if err := os.Symlink(src, dst); err != nil {
			return fmt.Errorf("failed to symlink %s: %w", name, err)
		}
	}

	return nil
}

// ensureExists creates a file or directory at path if it does not exist.
// Files (identified by having an extension) are created empty; everything
// else is created as a directory.
func ensureExists(path string) error {
	if _, err := os.Stat(path); err == nil {
		return nil // already exists
	}
	if filepath.Ext(path) != "" {
		// Looks like a file — create it empty.
		f, err := os.OpenFile(path, os.O_CREATE, 0600)
		if err != nil {
			return err
		}
		return f.Close()
	}
	// Create as directory.
	return os.MkdirAll(path, 0700)
}

// copyPreferences copies ~/.claude.json (user preferences: theme, onboarding state, etc.)
// into the runtime directory so claude does not show the first-run setup wizard.
func copyPreferences(runtimeDir string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	src := filepath.Join(home, ".claude.json")
	data, err := os.ReadFile(src)
	if os.IsNotExist(err) {
		// No preferences file yet, claude will create one on first run.
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to read .claude.json: %w", err)
	}

	dst := filepath.Join(runtimeDir, ".claude.json")
	if err := os.WriteFile(dst, data, 0600); err != nil {
		return fmt.Errorf("failed to write .claude.json: %w", err)
	}

	return nil
}

// loadDefaultSettings reads default-settings.json as a generic map.
// Returns an empty map if the file does not exist.
func loadDefaultSettings(claudeDir string) (map[string]interface{}, error) {
	src := filepath.Join(claudeDir, "default-settings.json")

	var data []byte
	var err error
	data, err = os.ReadFile(src)
	if os.IsNotExist(err) {
		if scaffoldErr := scaffoldClaudeDefaultSettings(src); scaffoldErr != nil {
			fmt.Fprintf(os.Stderr, "quickswitch: warning: could not scaffold default settings: %v\n", scaffoldErr)
			return make(map[string]interface{}), nil
		}
		fmt.Fprintf(os.Stderr, "quickswitch: created default Claude settings: %s\n  Edit this file to customize your settings.\n", src)
		data, err = os.ReadFile(src)
		if err != nil {
			return nil, fmt.Errorf("failed to read default-settings.json: %w", err)
		}
	} else if err != nil {
		return nil, fmt.Errorf("failed to read default-settings.json: %w", err)
	}

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("failed to parse default-settings.json: %w", err)
	}

	return settings, nil
}

// mergeEnvProfile merges the profile's env vars into the settings "env" section.
// Profile values override any existing values with the same key.
func mergeEnvProfile(settings map[string]interface{}, profile EnvProfile) {
	if len(profile) == 0 {
		return
	}

	envSection, _ := settings["env"].(map[string]interface{})
	if envSection == nil {
		envSection = make(map[string]interface{})
	}

	for key, value := range profile {
		envSection[key] = value
	}

	settings["env"] = envSection
}

// writeSettings marshals settings to JSON and writes it to the runtime directory.
func writeSettings(runtimeDir string, settings map[string]interface{}) error {
	data, err := json.MarshalIndent(settings, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal settings: %w", err)
	}

	dst := filepath.Join(runtimeDir, "settings.json")
	if err := os.WriteFile(dst, data, 0600); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}

	return nil
}
