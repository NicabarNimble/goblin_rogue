package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/NicabarNimble/go-gittools/internal/token"
	"github.com/NicabarNimble/go-gittools/internal/github"
	"github.com/NicabarNimble/go-gittools/internal/gitlab"
)

var (
	value         string
	expires       string
	tokenFile     string
	nonInteractive bool
	osExit        = os.Exit // For testing purposes
)

// detectedProvider stores the auto-detected provider
var detectedProvider token.Provider

// Environment variable names for non-interactive mode
const (
	EnvProvider    = "GIT_PROVIDER"
	EnvTokenValue  = "GIT_TOKEN_VALUE"
	EnvTokenExpiry = "GIT_TOKEN_EXPIRY"
)

func main() {
	rootCmd := &cobra.Command{
		Use:   "go-gittoken",
		Short: "Manage Git authentication tokens",
		Long: `A tool for managing Git authentication tokens.
Supports token setup, validation, and storage configuration.`,
	}

	// Setup command
	setupCmd := &cobra.Command{
		Use:   "setup",
		Short: "Set up a new Git authentication token",
		Long: `Interactive guide for setting up a new Git authentication token.
Validates the token and configures storage.`,
		Run: setupToken,
	}

	setupCmd.Flags().StringVarP(&value, "token", "t", "", "Token value")
	setupCmd.Flags().StringVarP(&expires, "expires", "e", "", "Token expiration (e.g., 30d, 1y)")
	setupCmd.Flags().StringVarP(&tokenFile, "token-file", "f", "", "File containing the token value")
	setupCmd.Flags().BoolVarP(&nonInteractive, "non-interactive", "n", false, "Run in non-interactive mode")

	rootCmd.AddCommand(setupCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		osExit(1)
	}
}

func setupToken(cmd *cobra.Command, args []string) {
	ctx := context.Background()

	// Load from environment variables in non-interactive mode
	if nonInteractive {
		loadFromEnv()
	}

	// Load token from file if specified
	if tokenFile != "" {
		if err := loadTokenFromFile(); err != nil {
			fmt.Printf("Error loading token from file: %v\n", err)
			osExit(1)
		}
	}

	// Check file permissions if token file is used
	if tokenFile != "" {
		if err := checkFilePermissions(tokenFile); err != nil {
			fmt.Printf("Warning: %v\n", err)
		}
	}

	if value == "" && !nonInteractive {
		fmt.Print("\nPlease enter your Git token: ")

		// Create a context with 30-second timeout
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// Channel to receive the token input
		tokenCh := make(chan string)

		// Start a goroutine to read the token
		go func() {
			var input string
			fmt.Scanln(&input)
			tokenCh <- input
		}()

		// Wait for either token input or timeout
		select {
		case input := <-tokenCh:
			value = input
		case <-ctx.Done():
			fmt.Println("\nTimeout: No token provided within 30 seconds")
			osExit(1)
		}
	}

	// Auto-detect provider from token format
	detectedProvider = token.DetectProvider(value)
	if detectedProvider == "" {
		fmt.Println("Error: Unable to detect token provider. Please ensure you're using a valid GitHub or GitLab token.")
		osExit(1)
	}

	fmt.Printf("Detected %s token\n", detectedProvider)

	var expiresAt time.Time
	if expires != "" {
		duration, err := parseDuration(expires)
		if err != nil {
			fmt.Printf("Error parsing expiration: %v\n", err)
			osExit(1)
		}
		expiresAt = time.Now().Add(duration)
	}

	// Create token with required scopes based on provider
	var requiredScopes string
	switch detectedProvider {
	case token.ProviderGitHub:
		requiredScopes = "repo workflow admin:repo"
	case token.ProviderGitLab:
		requiredScopes = "api" // GitLab's equivalent scopes
	}

	// Create and validate token
	newToken, err := token.NewToken(value, expiresAt, requiredScopes)
	if err != nil {
		if errors.Is(err, token.ErrTokenInvalid) {
			fmt.Printf("Error: Invalid token format\n")
		} else {
			fmt.Printf("Error creating token: %v\n", err)
		}
		osExit(1)
	}

	// Validate token with provider's API
	var tokenInfo string
	switch detectedProvider {
	case token.ProviderGitHub:
		validator := github.NewTokenValidator()
		if err := validator.Validate(ctx, newToken); err != nil {
			var scopeErr *token.ScopeError
			if errors.As(err, &scopeErr) {
				fmt.Println("\nRequired GitHub token scopes:")
				for scope, present := range scopeErr.Status {
					status := "✓"
					if !present {
						status = "✗"
					}
					fmt.Printf("%s %s\n", status, scope)
				}
				fmt.Printf("\nError: Token is missing required scopes. Please add the missing scopes marked with ✗\n")
			} else if errors.Is(err, token.ErrTokenExpired) {
				fmt.Printf("Error: GitHub token has expired. Please provide a new token\n")
			} else {
				fmt.Printf("Error validating GitHub token: %v\n", err)
			}
			osExit(1)
		}
		tokenInfo = fmt.Sprintf("Scopes: %s", newToken.Scope)
	case token.ProviderGitLab:
		validator := gitlab.NewTokenValidator()
		if err := validator.Validate(ctx, newToken); err != nil {
			if strings.Contains(err.Error(), "missing required scopes") {
				fmt.Printf("Error: GitLab token is missing required scopes (api). Please check token permissions\n")
			} else if errors.Is(err, token.ErrTokenExpired) {
				fmt.Printf("Error: GitLab token has expired. Please provide a new token\n")
			} else {
				fmt.Printf("Error validating GitLab token: %v\n", err)
			}
			osExit(1)
		}
		tokenInfo = fmt.Sprintf("Scopes: %s", newToken.Scope)
	}

	// Store validated token in environment
	envStorage := token.NewEnvStorage()
	if err := envStorage.Store(ctx, string(detectedProvider), *newToken); err != nil {
		if errors.Is(err, token.ErrStorageUnavailable) {
			fmt.Printf("Error: Unable to access token storage. Please check environment permissions\n")
		} else {
			fmt.Printf("Error storing token: %v\n", err)
		}
		osExit(1)
	}

	fmt.Printf("\nSuccessfully configured %s token!\n", detectedProvider)
	fmt.Println("\nToken details:")
	fmt.Printf("Provider: %s\n", detectedProvider)
	fmt.Println(tokenInfo)
	if !newToken.ExpiresAt.IsZero() {
		fmt.Printf("Expires: %s\n", newToken.ExpiresAt.Format("January 2, 2006 at 3:04 PM MST"))
		daysUntilExpiry := time.Until(newToken.ExpiresAt).Hours() / 24
		if daysUntilExpiry < 7 {
			fmt.Printf("\nWarning: Token will expire in %.0f days\n", daysUntilExpiry)
		}
	} else {
		fmt.Println("Expires: Never")
	}

	fmt.Printf("\nEnvironment variable set: GIT_TOKEN_%s\n", detectedProvider)
}

// loadFromEnv loads token configuration from environment variables
func loadFromEnv() {
	if envToken := os.Getenv(EnvTokenValue); envToken != "" && value == "" {
		value = envToken
	}
	if envExpiry := os.Getenv(EnvTokenExpiry); envExpiry != "" && expires == "" {
		expires = envExpiry
	}
}

// loadTokenFromFile loads the token value from a file
func loadTokenFromFile() error {
	data, err := os.ReadFile(tokenFile)
	if err != nil {
		return fmt.Errorf("failed to read token file: %w", err)
	}
	value = strings.TrimSpace(string(data))
	return nil
}

// checkFilePermissions verifies that the token file has secure permissions
func checkFilePermissions(filepath string) error {
	info, err := os.Stat(filepath)
	if err != nil {
		return fmt.Errorf("failed to check file permissions: %w", err)
	}

	mode := info.Mode()
	if mode&0077 != 0 {
		return fmt.Errorf("token file has insecure permissions. Please run: chmod 600 %s", filepath)
	}

	return nil
}

func parseDuration(s string) (time.Duration, error) {
	// Handle year notation (e.g., "1y")
	if strings.HasSuffix(s, "y") {
		yearsStr := strings.TrimSuffix(s, "y")
		numYears, err := time.ParseDuration(yearsStr + "h")
		if err != nil {
			return 0, err
		}
		hours := numYears.Hours()
		return time.Duration(hours * float64(365*24)) * time.Hour, nil
	}

	// Handle day notation (e.g., "30d")
	if strings.HasSuffix(s, "d") {
		daysStr := strings.TrimSuffix(s, "d")
		numDays, err := time.ParseDuration(daysStr + "h")
		if err != nil {
			return 0, err
		}
		hours := numDays.Hours()
		return time.Duration(hours * 24) * time.Hour, nil
	}

	return time.ParseDuration(s)
}
