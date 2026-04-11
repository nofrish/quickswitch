package cmd

import (
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "qs",
	Short: "quickswitch - manage and launch AI CLI tools with different profiles",
	Long: `quickswitch (qs) lets you manage multiple configuration profiles
for AI CLI tools like Claude Code, and launch them with a single command.`,
	CompletionOptions: cobra.CompletionOptions{
		HiddenDefaultCmd: true,
	},
}

// Execute is the entry point called from main.go.
func Execute() error {
	return rootCmd.Execute()
}
