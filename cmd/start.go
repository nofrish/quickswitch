package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/charmbracelet/huh"
	"github.com/nofrish/quickswitch/internal/config"
	"github.com/spf13/cobra"
)

var startCmd = &cobra.Command{
	Use:   "start <tool> [profile] [tool-args...]",
	Short: "Launch a tool with a profile. Interactively selects profile if not specified.",
	// DisableFlagParsing lets all arguments pass through to the launched tool as-is.
	DisableFlagParsing: true,
	RunE:               runStart,
}

func init() {
	rootCmd.AddCommand(startCmd)
}

func runStart(cmd *cobra.Command, args []string) error {
	if len(args) == 0 || args[0] == "-h" || args[0] == "--help" {
		return cmd.Help()
	}

	tool := args[0]

	// If the next arg exists and doesn't look like a flag, treat it as a profile name.
	// Otherwise, show an interactive selector.
	var profileName string
	var toolArgs []string

	rest := args[1:]
	if len(rest) > 0 && rest[0][0] != '-' {
		profileName = rest[0]
		toolArgs = rest[1:]
	} else {
		toolArgs = rest
		selected, err := selectProfileInteractive(tool)
		if err != nil {
			return err
		}
		profileName = selected
	}

	switch tool {
	case "claude":
		return launchClaude(profileName, toolArgs)
	case "codex":
		return launchCodex(profileName, toolArgs)
	default:
		return fmt.Errorf("unsupported tool %q, supported tools: claude, codex", tool)
	}
}

func selectProfileInteractive(tool string) (string, error) {
	toolDir, err := config.ToolDir(tool)
	if err != nil {
		return "", err
	}

	cfg, err := config.LoadEnvConfig(toolDir)
	if os.IsNotExist(err) {
		return "", fmt.Errorf("no profiles found, use 'qs add' to create one")
	}
	if err != nil {
		return "", fmt.Errorf("failed to load profiles: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		return "", fmt.Errorf("no profiles found, use 'qs add' to create one")
	}

	options := make([]huh.Option[string], 0, len(cfg.Profiles))
	for _, name := range profileNames(cfg) {
		options = append(options, huh.NewOption(name, name))
	}

	var profileName string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a profile").
				Options(options...).
				Value(&profileName),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}

	return profileName, nil
}

// launchClaude builds a per-profile runtime directory and launches claude.
// profileName may be empty (uses default-settings.json only, no credentials).
// extraArgs are passed through to the claude process as-is.
func launchClaude(profileName string, extraArgs []string) error {
	claudeDir, err := config.ToolDir("claude")
	if err != nil {
		return err
	}

	profile, err := resolveProfile(claudeDir, profileName)
	if err != nil {
		return err
	}

	runtimeDir, err := config.BuildRuntimeSettings(claudeDir, profileName, profile)
	if err != nil {
		return err
	}

	env := stripEnvVars(os.Environ(), claudeAuthEnvVars)
	env = append(env, "CLAUDE_CONFIG_DIR="+runtimeDir)

	claude := exec.Command("claude", extraArgs...)
	claude.Env = env
	claude.Stdin = os.Stdin
	claude.Stdout = os.Stdout
	claude.Stderr = os.Stderr

	return claude.Run()
}

// launchCodex builds a per-profile runtime directory and launches codex.
// profileName may be empty (uses default-config.toml only, no credentials).
// extraArgs are passed through to the codex process as-is.
func launchCodex(profileName string, extraArgs []string) error {
	codexDir, err := config.ToolDir("codex")
	if err != nil {
		return err
	}

	profile, err := resolveProfile(codexDir, profileName)
	if err != nil {
		return err
	}

	runtimeDir, err := config.BuildCodexRuntime(codexDir, profileName, profile)
	if err != nil {
		return err
	}

	env := stripEnvVars(os.Environ(), codexAuthEnvVars)
	env = append(env, "CODEX_HOME="+runtimeDir)

	codex := exec.Command("codex", extraArgs...)
	codex.Env = env
	codex.Stdin = os.Stdin
	codex.Stdout = os.Stdout
	codex.Stderr = os.Stderr

	return codex.Run()
}

// resolveProfile loads the env vars for a named profile from the tool's config directory.
// Returns an empty profile if profileName is empty.
func resolveProfile(toolDir string, profileName string) (config.EnvProfile, error) {
	if profileName == "" {
		return config.EnvProfile{}, nil
	}

	cfg, err := config.LoadEnvConfig(toolDir)
	if os.IsNotExist(err) {
		return nil, fmt.Errorf("no profiles found, use 'qs add' to create one")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load profiles: %w", err)
	}

	profile, ok := cfg.Profiles[profileName]
	if !ok {
		return nil, fmt.Errorf("profile %q not found, use 'qs list' to see available profiles", profileName)
	}

	return profile, nil
}

// claudeAuthEnvVars are stripped from the shell so that the profile's settings.json values take effect.
var claudeAuthEnvVars = map[string]bool{
	"ANTHROPIC_API_KEY":       true,
	"ANTHROPIC_AUTH_TOKEN":    true,
	"ANTHROPIC_BASE_URL":      true,
	"CLAUDE_CODE_OAUTH_TOKEN": true,
}

// codexAuthEnvVars are stripped from the shell so that the profile's auth.json values take effect.
var codexAuthEnvVars = map[string]bool{
	"OPENAI_API_KEY": true,
}

// stripEnvVars returns a copy of env with all keys in the given set removed.
func stripEnvVars(env []string, keys map[string]bool) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if !keys[key] {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
