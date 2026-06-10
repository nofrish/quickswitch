package cmd

import (
	"errors"
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/nofrish/quickswitch/internal/config"
	"github.com/spf13/cobra"
)

var switchCmd = &cobra.Command{
	Use:   "switch [tool] [profile]",
	Short: "Switch the official default config for a tool to a profile",
	Long: `Switch writes a quickswitch profile into the tool's official default
configuration directory. Unlike "start", this changes what the tool uses when
you launch it normally outside quickswitch.`,
	Args: cobra.MaximumNArgs(2),
	RunE: runSwitch,
}

func init() {
	rootCmd.AddCommand(switchCmd)
}

func runSwitch(cmd *cobra.Command, args []string) error {
	var tool string
	var profileName string

	switch len(args) {
	case 0:
		selectedTool, err := selectTool()
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Cancelled.")
			return nil
		}
		if err != nil {
			return err
		}
		tool = selectedTool
	case 1:
		tool = args[0]
	case 2:
		tool = args[0]
		profileName = args[1]
	}

	if !isSupportedTool(tool) {
		return unsupportedToolError(tool)
	}

	if profileName == "" {
		selectedProfile, err := selectProfileInteractive(tool)
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Cancelled.")
			return nil
		}
		if err != nil {
			return err
		}
		profileName = selectedProfile
	}

	toolDir, err := config.ToolDir(tool)
	if err != nil {
		return fmt.Errorf("failed to resolve config directory: %w", err)
	}

	profile, err := resolveProfile(toolDir, profileName)
	if err != nil {
		return err
	}

	switch tool {
	case "claude":
		settingsPath, err := config.ApplyClaudeGlobalProfile(toolDir, profile)
		if err != nil {
			return err
		}
		fmt.Printf("Switched claude default config to profile %q.\n", profileName)
		fmt.Printf("Updated %s\n", settingsPath)
	case "codex":
		configPath, authPath, err := config.ApplyCodexGlobalProfile(toolDir, profileName, profile)
		if err != nil {
			return err
		}
		fmt.Printf("Switched codex default config to profile %q.\n", profileName)
		fmt.Printf("Updated %s\n", configPath)
		fmt.Printf("Updated %s\n", authPath)
	}

	return nil
}
