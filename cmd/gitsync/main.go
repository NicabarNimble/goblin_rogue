package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

func newRootCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "gitsync",
		Short: "GitHub repository synchronization tool",
		Long: `A CLI tool for managing GitHub repository synchronization through GitHub Actions.
Supports initializing workflows, triggering syncs, checking status, and viewing logs.`,
	}

	// Add subcommands
	cmd.AddCommand(
		newInitCmd(),
		newRunCmd(),
		newStatusCmd(),
		newLogsCmd(),
		newConfigureCmd(),
	)

	return cmd
}

func main() {
	rootCmd := newRootCmd()
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
