package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time
var Version = "2.0.0"

// rootCmd represents the base command
var rootCmd = &cobra.Command{
	Use:   "oio",
	Short: "OIO CLI - Ephemeral content management",
	Long: `OIO CLI - Ephemeral content management

A fast CLI tool for managing ephemeral content with automatic TTL-based deletion.
Upload text, files, and screenshots with optional sharing capabilities.`,
	SilenceUsage:  true,
	SilenceErrors: true,
}

// Execute runs the root command
func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.Version = Version

	// Add all subcommands
	addAuthCommands()
	addHealthCommand()
	addConfigCommand()
	addAddCommand()
	addGetCommand()
	addListCommand()
	addDeleteCommand()
	addExtendCommand()
	addShareCommand()
	addShortcutCommands()
}

// exitWithError prints an error message and exits
func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
