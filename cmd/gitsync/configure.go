package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/NicabarNimble/go-gittools/internal/config"
	"github.com/spf13/cobra"
)

type configureOptions struct {
	sourceRepo     string
	targetRepo     string
	schedule       string
	branchMappings []string
	errorNotify    bool
	notifyEmail    string
	retryAttempts  int
	retryDelay     string
	configFile     string
}

func newConfigureCmd() *cobra.Command {
	opts := &configureOptions{}

	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Update sync settings",
		Long: `Update configuration settings for repository synchronization.
Settings include source/target repositories, branch mappings, schedule, and error handling.`,
		Example: `  gitsync configure --source owner/repo --target fork/repo
  gitsync configure --branch main:master,dev:development
  gitsync configure --schedule "0 0 * * *"
  gitsync configure --error-notify --notify-email user@example.com
  gitsync configure --retry-attempts 5 --retry-delay 10m`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return updateConfig(opts)
		},
	}

	cmd.Flags().StringVar(&opts.sourceRepo, "source", "", "Source repository (owner/repo)")
	cmd.Flags().StringVar(&opts.targetRepo, "target", "", "Target repository (owner/repo)")
	cmd.Flags().StringVar(&opts.schedule, "schedule", "", "Cron schedule for automated syncs")
	cmd.Flags().StringSliceVar(&opts.branchMappings, "branch", nil, "Branch mappings (source:target)")
	cmd.Flags().BoolVar(&opts.errorNotify, "error-notify", false, "Enable error notifications")
	cmd.Flags().StringVar(&opts.notifyEmail, "notify-email", "", "Email address for error notifications")
	cmd.Flags().IntVar(&opts.retryAttempts, "retry-attempts", 0, "Number of retry attempts (0-10)")
	cmd.Flags().StringVar(&opts.retryDelay, "retry-delay", "", "Delay between retries (e.g. 5m, 1h)")
	cmd.Flags().StringVar(&opts.configFile, "config", ".gitsync.json", "Configuration file path")

	return cmd
}

func updateConfig(opts *configureOptions) error {
	// Load existing config if it exists
	cfg := &config.SyncConfig{}
	if _, err := os.Stat(opts.configFile); err == nil {
		data, err := os.ReadFile(opts.configFile)
		if err != nil {
			return fmt.Errorf("failed to read config file: %w", err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return fmt.Errorf("failed to parse config file: %w", err)
		}
	}

	// Update config with new values
	if opts.sourceRepo != "" {
		if err := config.ValidateRepoFormat(opts.sourceRepo); err != nil {
			return fmt.Errorf("invalid source repository: %w", err)
		}
		cfg.SourceRepo = opts.sourceRepo
	}

	if opts.targetRepo != "" {
		if err := config.ValidateRepoFormat(opts.targetRepo); err != nil {
			return fmt.Errorf("invalid target repository: %w", err)
		}
		cfg.TargetRepo = opts.targetRepo
	}

	if opts.schedule != "" {
		if err := config.ValidateSchedule(opts.schedule); err != nil {
			return fmt.Errorf("invalid schedule: %w", err)
		}
		cfg.Schedule = opts.schedule
	}

	if len(opts.branchMappings) > 0 {
		if cfg.BranchMappings == nil {
			cfg.BranchMappings = make(map[string]string)
		}
		for _, mapping := range opts.branchMappings {
			source, target, err := config.ParseBranchMapping(mapping)
			if err != nil {
				return fmt.Errorf("invalid branch mapping: %w", err)
			}
			cfg.BranchMappings[source] = target
		}
	}

	// Update error handling configuration
	if opts.errorNotify {
		cfg.ErrorHandling.Notify = true
		if opts.notifyEmail != "" {
			cfg.ErrorHandling.NotifyEmail = opts.notifyEmail
		}
	}
	if opts.retryAttempts > 0 {
		if opts.retryAttempts > 10 {
			return fmt.Errorf("retry attempts cannot exceed 10")
		}
		cfg.ErrorHandling.RetryAttempts = opts.retryAttempts
	}
	if opts.retryDelay != "" {
		cfg.ErrorHandling.RetryDelay = opts.retryDelay
	}

	// Create config directory if it doesn't exist
	configDir := filepath.Dir(opts.configFile)
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	// Save updated config
	if err := config.SaveConfig(cfg, opts.configFile); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Configuration updated successfully:\n")
	fmt.Printf("Source repository: %s\n", cfg.SourceRepo)
	fmt.Printf("Target repository: %s\n", cfg.TargetRepo)
	if cfg.Schedule != "" {
		fmt.Printf("Schedule: %s\n", cfg.Schedule)
	}
	if len(cfg.BranchMappings) > 0 {
		fmt.Printf("Branch mappings:\n")
		for source, target := range cfg.BranchMappings {
			fmt.Printf("  %s -> %s\n", source, target)
		}
	}
	fmt.Printf("Error handling:\n")
	fmt.Printf("  Notifications: %v\n", cfg.ErrorHandling.Notify)
	if cfg.ErrorHandling.NotifyEmail != "" {
		fmt.Printf("  Notify email: %s\n", cfg.ErrorHandling.NotifyEmail)
	}
	fmt.Printf("  Retry attempts: %d\n", cfg.ErrorHandling.RetryAttempts)
	fmt.Printf("  Retry delay: %s\n", cfg.ErrorHandling.RetryDelay)

	return nil
}
