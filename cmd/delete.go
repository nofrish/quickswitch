package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
	"github.com/nofrish/quickswitch/internal/config"
	"github.com/spf13/cobra"
)

var deleteCmd = &cobra.Command{
	Use:   "delete",
	Short: "Delete a profile",
	RunE:  runDelete,
}

func init() {
	rootCmd.AddCommand(deleteCmd)
}

func runDelete(cmd *cobra.Command, args []string) error {
	// Step 1: select tool
	tool, err := selectTool()
	if errors.Is(err, huh.ErrUserAborted) {
		fmt.Println("Cancelled.")
		return nil
	}
	if err != nil {
		return err
	}

	toolDir, err := config.ToolDir(tool)
	if err != nil {
		return fmt.Errorf("failed to resolve config directory: %w", err)
	}

	// Step 2: load profiles
	cfg, err := config.LoadEnvConfig(toolDir)
	if os.IsNotExist(err) {
		fmt.Printf("No profiles found for %s.\n", tool)
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load profiles: %w", err)
	}

	if len(cfg.Profiles) == 0 {
		fmt.Printf("No profiles found for %s.\n", tool)
		return nil
	}

	// Step 3: select profile to delete
	profileName, err := selectProfile(cfg)
	if errors.Is(err, huh.ErrUserAborted) {
		fmt.Println("Cancelled.")
		return nil
	}
	if err != nil {
		return err
	}

	// Step 4: confirm deletion
	confirmed, err := confirmDeletion(profileName)
	if errors.Is(err, huh.ErrUserAborted) {
		fmt.Println("Cancelled.")
		return nil
	}
	if err != nil {
		return err
	}
	if !confirmed {
		fmt.Println("Cancelled.")
		return nil
	}

	// Step 5: delete and save
	delete(cfg.Profiles, profileName)

	if err := config.SaveEnvConfig(toolDir, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Profile %q deleted.\n", profileName)
	return nil
}

func selectTool() (string, error) {
	var tool string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select tool").
				Options(
					huh.NewOption("claude", "claude"),
					huh.NewOption("codex", "codex"),
				).
				Value(&tool),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return tool, nil
}

func selectProfile(cfg *config.EnvConfig) (string, error) {
	var profileName string

	options := make([]huh.Option[string], 0, len(cfg.Profiles))
	for name := range cfg.Profiles {
		options = append(options, huh.NewOption(name, name))
	}

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select profile to delete").
				Options(options...).
				Value(&profileName),
		),
	)
	if err := form.Run(); err != nil {
		return "", err
	}
	return profileName, nil
}

func confirmDeletion(profileName string) (bool, error) {
	var confirmed bool
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title(fmt.Sprintf("Delete profile %q?", profileName)).
				Value(&confirmed),
		),
	)
	if err := form.Run(); err != nil {
		return false, err
	}
	return confirmed, nil
}
