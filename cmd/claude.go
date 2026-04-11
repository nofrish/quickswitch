package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/nofrish/quickswitch/internal/config"
	"github.com/spf13/cobra"
)

var claudeCmd = &cobra.Command{
	Use:   "claude [profile] [claude-args...]",
	Short: "Launch Claude with an optional profile",
	// DisableFlagParsing lets all arguments pass through to claude as-is.
	DisableFlagParsing: true,
	RunE:               runClaude,
}

func init() {
	rootCmd.AddCommand(claudeCmd)
}

func runClaude(cmd *cobra.Command, args []string) error {
	// Handle -h/--help manually since DisableFlagParsing is active.
	if len(args) > 0 && (args[0] == "-h" || args[0] == "--help") {
		return cmd.Help()
	}

	// The first argument is the profile name if it doesn't look like a flag.
	// Everything else is passed through to claude.
	var profileName string
	claudeArgs := args

	if len(args) > 0 && args[0][0] != '-' {
		profileName = args[0]
		claudeArgs = args[1:]
	}

	claudeDir, err := config.ToolDir("claude")
	if err != nil {
		return err
	}

	// Resolve the profile's env vars (empty if no profile specified).
	profile, err := resolveProfile(claudeDir, profileName)
	if err != nil {
		return err
	}

	// Build a per-profile settings.json in an isolated runtime directory.
	runtimeDir, err := config.BuildRuntimeSettings(claudeDir, profileName, profile)
	if err != nil {
		return err
	}

	// Launch claude with CLAUDE_CONFIG_DIR pointing to the isolated runtime directory.
	// We strip any existing auth/URL env vars from the shell so that the values in
	// settings.json (which contain the profile's credentials) take effect.
	env := stripAuthEnvVars(os.Environ())
	env = append(env, "CLAUDE_CONFIG_DIR="+runtimeDir)

	claude := exec.Command("claude", claudeArgs...)
	claude.Env = env
	claude.Stdin = os.Stdin
	claude.Stdout = os.Stdout
	claude.Stderr = os.Stderr

	return claude.Run()
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
