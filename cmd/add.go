package cmd

import (
	"errors"
	"fmt"
	"os"

	"github.com/charmbracelet/huh"
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
	// Step 1: select tool
	var tool string
	toolForm := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("选择工具").
				Options(
					huh.NewOption("claude", "claude"),
					huh.NewOption("codex", "codex"),
				).
				Value(&tool),
		),
	)
	if err := toolForm.Run(); err != nil {
		if errors.Is(err, huh.ErrUserAborted) {
			fmt.Println("Cancelled.")
			return nil
		}
		return err
	}

	// Step 2: select provider + enter profile name (options depend on tool)
	var providerID, profileName string
	var err error

	switch tool {
	case "claude":
		providerID, profileName, err = selectClaudeProviderAndName()
	case "codex":
		providerID, profileName, err = selectCodexProviderAndName()
	}
	if errors.Is(err, huh.ErrUserAborted) {
		fmt.Println("Cancelled.")
		return nil
	}
	if err != nil {
		return err
	}

	toolDir, err := config.EnsureToolDir(tool)
	if err != nil {
		return fmt.Errorf("failed to initialize config directory: %w", err)
	}

	// Step 3: check for duplicate profile
	cfg, err := loadOrInitEnvConfig(toolDir)
	if err != nil {
		return err
	}

	if _, exists := cfg.Profiles[profileName]; exists {
		var overwrite bool
		confirm := huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Profile %q already exists. Overwrite?", profileName)).
					Value(&overwrite),
			),
		)
		if err := confirm.Run(); err != nil {
			if errors.Is(err, huh.ErrUserAborted) {
				fmt.Println("Cancelled.")
				return nil
			}
			return err
		}
		if !overwrite {
			fmt.Println("Cancelled.")
			return nil
		}
	}

	// Step 4: collect credentials
	var profile config.EnvProfile
	switch tool {
	case "claude":
		profile, err = buildClaudeProfile(providerID)
	case "codex":
		profile, err = buildCodexProfile(providerID)
	}
	if errors.Is(err, huh.ErrUserAborted) {
		fmt.Println("Cancelled.")
		return nil
	}
	if err != nil {
		return err
	}

	cfg.Profiles[profileName] = profile

	if err := config.SaveEnvConfig(toolDir, cfg); err != nil {
		return fmt.Errorf("failed to save profile: %w", err)
	}

	fmt.Printf("\nProfile %q saved for %s.\n", profileName, tool)
	return nil
}

// --- Claude ---

func selectClaudeProviderAndName() (providerID, profileName string, err error) {
	providers, err := config.LoadProviders()
	if err != nil {
		return "", "", err
	}

	providerOptions := make([]huh.Option[string], 0, len(providers)+1)
	for _, p := range providers {
		providerOptions = append(providerOptions, huh.NewOption(p.Name, p.ID))
	}
	providerOptions = append(providerOptions, huh.NewOption("自定义", "custom"))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("选择供应商").
				Options(providerOptions...).
				Value(&providerID),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Profile 名称").
				Value(&profileName),
		),
	)
	return providerID, profileName, form.Run()
}

// buildClaudeProfile collects Anthropic credentials based on the selected provider.
func buildClaudeProfile(providerID string) (config.EnvProfile, error) {
	if providerID == "custom" {
		return buildClaudeCustomProfile()
	}

	provider, ok := config.FindProvider(providerID)
	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerID)
	}

	var authToken string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("API Key").
				Value(&authToken),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}

	profile := config.EnvProfile{
		"ANTHROPIC_AUTH_TOKEN": authToken,
		"ANTHROPIC_BASE_URL":   provider.BaseURL,
	}
	for k, v := range provider.Env {
		profile[k] = v
	}
	return profile, nil
}

func buildClaudeCustomProfile() (config.EnvProfile, error) {
	var authToken, baseURL string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("ANTHROPIC_AUTH_TOKEN").
				Value(&authToken),
			huh.NewInput().
				Title("ANTHROPIC_BASE_URL").
				Value(&baseURL),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}

	profile := config.EnvProfile{}
	if authToken != "" {
		profile["ANTHROPIC_AUTH_TOKEN"] = authToken
	}
	if baseURL != "" {
		profile["ANTHROPIC_BASE_URL"] = baseURL
	}
	return profile, nil
}

// --- Codex ---

func selectCodexProviderAndName() (providerID, profileName string, err error) {
	providers, err := config.LoadCodexProviders()
	if err != nil {
		return "", "", err
	}

	providerOptions := make([]huh.Option[string], 0, len(providers)+1)
	for _, p := range providers {
		providerOptions = append(providerOptions, huh.NewOption(p.Name, p.ID))
	}
	providerOptions = append(providerOptions, huh.NewOption("自定义", "custom"))

	form := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("选择供应商").
				Options(providerOptions...).
				Value(&providerID),
		),
		huh.NewGroup(
			huh.NewInput().
				Title("Profile 名称").
				Value(&profileName),
		),
	)
	return providerID, profileName, form.Run()
}

// buildCodexProfile collects OpenAI-compatible credentials based on the selected provider.
func buildCodexProfile(providerID string) (config.EnvProfile, error) {
	if providerID == "custom" {
		return buildCodexCustomProfile()
	}

	provider, ok := config.FindCodexProvider(providerID)
	if !ok {
		return nil, fmt.Errorf("provider %q not found", providerID)
	}

	var apiKey string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("API Key").
				Value(&apiKey),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}

	return config.EnvProfile{
		"OPENAI_API_KEY": apiKey,
		"base_url":       provider.BaseURL,
		"wire_api":       provider.WireAPI,
	}, nil
}

func buildCodexCustomProfile() (config.EnvProfile, error) {
	var apiKey, baseURL string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("OPENAI_API_KEY").
				Value(&apiKey),
			huh.NewInput().
				Title("Base URL").
				Value(&baseURL),
		),
	)
	if err := form.Run(); err != nil {
		return nil, err
	}

	profile := config.EnvProfile{}
	if apiKey != "" {
		profile["OPENAI_API_KEY"] = apiKey
	}
	if baseURL != "" {
		profile["base_url"] = baseURL
		profile["wire_api"] = "responses"
	}
	return profile, nil
}

// loadOrInitEnvConfig loads the existing env.json, or returns an empty config if it doesn't exist yet.
func loadOrInitEnvConfig(toolDir string) (*config.EnvConfig, error) {
	cfg, err := config.LoadEnvConfig(toolDir)
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
