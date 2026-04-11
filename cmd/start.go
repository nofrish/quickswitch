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
	Use:   "start <tool> [profile] [claude-args...]",
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
	if tool != "claude" {
		return fmt.Errorf("unsupported tool %q, currently only 'claude' is supported", tool)
	}

	// If the next arg exists and doesn't look like a flag, treat it as a profile name.
	// Otherwise, show an interactive selector.
	var profileName string
	var claudeArgs []string

	rest := args[1:]
	if len(rest) > 0 && rest[0][0] != '-' {
		profileName = rest[0]
		claudeArgs = rest[1:]
	} else {
		claudeArgs = rest
		selected, err := selectProfileInteractive(tool)
		if err != nil {
			return err
		}
		profileName = selected
	}

	return launchClaude(profileName, claudeArgs)
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

	// Strip any existing auth/URL env vars from the shell so that the values in
	// settings.json (which contain the profile's credentials) take effect.
	env := stripAuthEnvVars(os.Environ())
	env = append(env, "CLAUDE_CONFIG_DIR="+runtimeDir)

	claude := exec.Command("claude", extraArgs...)
	claude.Env = env
	claude.Stdin = os.Stdin
	claude.Stdout = os.Stdout
	claude.Stderr = os.Stderr

	return claude.Run()
}

// resolveProfile loads the env vars for a named profile.
// Returns an empty profile if profileName is empty.
func resolveProfile(claudeDir string, profileName string) (config.EnvProfile, error) {
	if profileName == "" {
		return config.EnvProfile{}, nil
	}

	cfg, err := config.LoadEnvConfig(claudeDir)
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

// authEnvVars is the set of environment variables that control API authentication
// and routing. These are stripped from the inherited shell environment so that
// the profile's settings.json values take effect instead.
var authEnvVars = map[string]bool{
	"ANTHROPIC_API_KEY":       true,
	"ANTHROPIC_AUTH_TOKEN":    true,
	"ANTHROPIC_BASE_URL":      true,
	"CLAUDE_CODE_OAUTH_TOKEN": true,
}

// stripAuthEnvVars returns a copy of env with all auth-related variables removed.
func stripAuthEnvVars(env []string) []string {
	filtered := make([]string, 0, len(env))
	for _, entry := range env {
		key, _, _ := strings.Cut(entry, "=")
		if !authEnvVars[key] {
			filtered = append(filtered, entry)
		}
	}
	return filtered
}
