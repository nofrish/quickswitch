package cmd

import (
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/nofrish/quickswitch/internal/config"
	"github.com/spf13/cobra"
)

var (
	toolStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("135")) // purple
	profileStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15")) // white
	keyStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))             // gray
)

var listCmd = &cobra.Command{
	Use:   "list [tool]",
	Short: "List all profiles. Optionally filter by tool (claude, codex).",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runList,
}

func init() {
	rootCmd.AddCommand(listCmd)
}

func runList(cmd *cobra.Command, args []string) error {
	// Determine which tools to list.
	tools := config.SupportedTools
	if len(args) == 1 {
		tools = []string{args[0]}
	}

	for _, tool := range tools {
		if err := listToolProfiles(tool); err != nil {
			return err
		}
	}

	return nil
}

func listToolProfiles(tool string) error {
	toolDir, err := config.ToolDir(tool)
	if err != nil {
		return fmt.Errorf("failed to resolve config directory: %w", err)
	}

	cfg, err := config.LoadEnvConfig(toolDir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load profiles for %s: %w", tool, err)
	}

	if len(cfg.Profiles) == 0 {
		return nil
	}

	fmt.Println(toolStyle.Render(tool))

	names := profileNames(cfg)
	for i, name := range names {
		isLast := i == len(names)-1

		branch := "├─"
		indent := "│    "
		if isLast {
			branch = "└─"
			indent = "     "
		}

		fmt.Printf("  %s %s\n", branch, profileStyle.Render(name))

		profile := cfg.Profiles[name]
		for key, value := range profile {
			fmt.Printf("  %s %s  %s\n", indent, keyStyle.Render(key), maybeRedact(key, value))
		}

	}

	fmt.Println()
	return nil
}

// profileNames returns profile names in sorted order for consistent display.
func profileNames(cfg *config.EnvConfig) []string {
	names := make([]string, 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

// secretKeys is the set of env variable names whose values should be redacted.
var secretKeys = map[string]bool{
	"ANTHROPIC_API_KEY":    true,
	"ANTHROPIC_AUTH_TOKEN": true,
}

// maybeRedact returns a masked value if the key is a known secret, otherwise returns the value as-is.
func maybeRedact(key, value string) string {
	if secretKeys[key] {
		return maskSecret(value)
	}
	return value
}

// maskSecret hides the middle portion of a secret value.
// If the value starts with "sk-", it preserves "sk-" + first 4 chars + "**" + last 4 chars.
// Otherwise, it preserves first 4 chars + "**" + last 4 chars.
func maskSecret(value string) string {
	const visibleChars = 4
	const mask = "********************"

	if strings.HasPrefix(value, "sk-") {
		body := value[3:] // strip "sk-"
		if len(body) <= visibleChars*2 {
			return value
		}
		return "sk-" + body[:visibleChars] + mask + body[len(body)-visibleChars:]
	}

	if len(value) <= visibleChars*2 {
		return value
	}
	return value[:visibleChars] + mask + value[len(value)-visibleChars:]
}
