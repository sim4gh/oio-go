package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

// Version is set at build time
var Version = "2.3.0"

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
	addRecCommand()
	addTrustYouCommand()
	addTrustMeCommand()
	addShortcutCommands()

	// Custom root help with tree structure and inline aliases
	defaultHelp := rootCmd.HelpFunc()
	rootCmd.SetHelpFunc(func(cmd *cobra.Command, args []string) {
		if cmd == rootCmd {
			printRootHelp()
		} else {
			defaultHelp(cmd, args)
		}
	})
}

func printRootHelp() {
	fmt.Print(`OIO CLI - Ephemeral content management

A fast CLI tool for managing ephemeral content with automatic TTL-based deletion.
Upload text, files, and screenshots with optional sharing capabilities.

Usage:
  oio [command]

Commands:
  a, add [input]              Add from clipboard, screenshot, file, or text
    ├ c                       Quick clipboard shortcut
    └ sc                      Quick screenshot shortcut
  auth                        Authentication commands
  config [subcommand]         Manage configuration
  d, delete <id>              Delete item by ID
  extend <id>                 Extend TTL or make item permanent
  g, get <id>                 Get/download item by ID
  health                      Check system health status
  ls, list                    List all items
  rec                         Record screen to GIF, MP4, or MOV
  sh, share <id>              Share item (Pro only)
    └ p <id>                  Quick public share shortcut
  trustme <token> <file>      Upload file using trust token
  trustyou                    Create trust token for unauthenticated uploads

Flags:
  -h, --help      help for oio
  -v, --version   version for oio

Use "oio [command] --help" for more information about a command.
`)
}

// exitWithError prints an error message and exits
func exitWithError(msg string) {
	fmt.Fprintln(os.Stderr, "Error:", msg)
	os.Exit(1)
}
