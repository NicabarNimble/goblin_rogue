// Package gitclone provides a CLI tool for cloning public repositories to private ones
package main

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/NicabarNimble/go-gittools/internal/gitutils"
)

var (
	customName string
	token     string
	// cloneFunc allows for mocking in tests
	cloneFunc = gitutils.CloneRepository
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "go-gitclone [source-repo-url]",
		Short: "Clone public repositories to private repositories",
		Long: `A tool for cloning public repositories to private repositories while maintaining
proper remote configuration and authentication.

Example usage:
  go-gitclone https://github.com/owner/repo.git
  go-gitclone https://github.com/owner/repo.git --name custom-name`,
		Args: cobra.ExactArgs(1),
		Run: func(cmd *cobra.Command, args []string) {
			// The CloneRepository function will handle the exit codes directly
			// Exit code 2 indicates repository already exists
			// Exit code 1 indicates other errors
			if err := cloneRepository(args[0]); err != nil {
				fmt.Printf("Error: %v\n", err)
				os.Exit(1)
			}
		},
	}

	rootCmd.Flags().StringVar(&customName, "name", "", "Custom name for the target repository")
	// Token flag is now optional as we'll try to get it automatically
	rootCmd.Flags().StringVar(&token, "token", "", "GitHub token for authentication (optional)")

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func cloneRepository(sourceURL string) error {
	opts := gitutils.CloneOptions{
		SourceURL:  sourceURL,
		WorkingDir: "",
		Verbose:    true,
		Token:      token,
		CustomName: customName,
	}

	// CloneRepository will handle exit codes directly for repository exists case
	if err := cloneFunc(opts); err != nil {
		// If we get here, it's an error other than "repository exists"
		return fmt.Errorf("clone operation failed: %w", err)
	}

	return nil
}
