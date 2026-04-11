package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/nofrish/quickswitch/internal/config"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add",
	Short: "Add a new profile",
	RunE:  runAdd,
}

func init() {
	rootCmd.AddCommand(addCmd)
}

func runAdd(cmd *cobra.Command, args []string) error {
	reader := bufio.NewReader(os.Stdin)

	// Step 1: select tool
	tool, err := promptTool(reader)
	if err != nil {
		return err
	}

	claudeDir, err := config.EnsureToolDir(tool)
	if err != nil {
		return fmt.Errorf("failed to initialize config directory: %w", err)
	}

	// Step 2: ask for profile name
	name, err := prompt(reader, "Profile name")
	if err != nil {
		return err
	}

	// Step 3: load existing config and check for duplicates
	cfg, err := loadOrInitEnvConfig(claudeDir)
	if err != nil {
		return err
	}

	if _, exists := cfg.Profiles[name]; exists {
		overwrite, err := promptConfirm(reader, fmt.Sprintf("Profile %q already exists. Overwrite?", name))
		if err != nil {
			return err
		}
		if !overwrite {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Step 4: ask for env variables
	authToken, err := prompt(reader, "ANTHROPIC_AUTH_TOKEN")
	if err != nil {
		return err
	}

	baseURL, err := prompt(reader, "ANTHROPIC_BASE_URL")
	if err != nil {
		return err
	}

	// Step 5: build and save the profile
	profile := config.EnvProfile{}
	if authToken != "" {
		profile["ANTHROPIC_AUTH_TOKEN"] = authToken
	}
	if baseURL != "" {
		profile["ANTHROPIC_BASE_URL"] = baseURL
	}

	cfg.Profiles[name] = profile

	if err := config.SaveEnvConfig(claudeDir, cfg); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("\nProfile %q saved for %s.\n", name, tool)
	return nil
}

// promptTool asks the user to select a tool, defaulting to claude.
func promptTool(reader *bufio.Reader) (string, error) {
	fmt.Println("Tool:")
	fmt.Println("  a) claude (default)")
	fmt.Println("  b) codex")

	answer, err := prompt(reader, "Choice [a]")
	if err != nil {
		return "", err
	}

	switch strings.ToLower(answer) {
	case "", "a":
		return "claude", nil
	case "b":
		return "codex", nil
	default:
		return "", fmt.Errorf("invalid choice %q", answer)
	}
}

// loadOrInitEnvConfig loads the existing env.json, or returns an empty config if it doesn't exist yet.
func loadOrInitEnvConfig(claudeDir string) (*config.EnvConfig, error) {
	cfg, err := config.LoadEnvConfig(claudeDir)
	if os.IsNotExist(err) {
		return &config.EnvConfig{
			Profiles: make(map[string]config.EnvProfile),
		}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to load env config: %w", err)
	}
	return cfg, nil
}

// prompt prints a label and reads a line of input from the user.
func prompt(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", fmt.Errorf("failed to read input: %w", err)
	}
	return strings.TrimSpace(line), nil
}

// promptConfirm asks a yes/no question and returns true if the user answers "y".
func promptConfirm(reader *bufio.Reader, question string) (bool, error) {
	answer, err := prompt(reader, question+" (y/n)")
	if err != nil {
		return false, err
	}
	return strings.ToLower(answer) == "y", nil
}
