package config

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"
)

// ErrorConfig defines error handling configuration
type ErrorConfig struct {
	RetryAttempts int    `json:"retry_attempts"`
	RetryDelay    string `json:"retry_delay"`
	Notify        bool   `json:"notify"`
	NotifyEmail   string `json:"notify_email,omitempty"`
}

// DefaultConfig provides default configuration values
func DefaultConfig() *SyncConfig {
	return &SyncConfig{
		Schedule: "0 */6 * * *", // Every 6 hours
		BranchMappings: map[string]string{
			"main": "main",
		},
		ErrorHandling: ErrorConfig{
			RetryAttempts: 3,
			RetryDelay:    "5m",
			Notify:        false,
		},
	}
}

// SyncConfig represents the configuration for repository synchronization
type SyncConfig struct {
	SourceRepo     string            `json:"source_repo"`
	TargetRepo     string            `json:"target_repo"`
	Schedule       string            `json:"schedule,omitempty"`
	BranchMappings map[string]string `json:"branch_mappings,omitempty"`
	ErrorHandling  ErrorConfig       `json:"error_handling"`
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*SyncConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	cfg := &SyncConfig{}
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	cfg.MergeDefaults()
	return cfg, nil
}

// SaveConfig saves configuration to a file
func SaveConfig(cfg *SyncConfig, path string) error {
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// MergeDefaults merges default values for unset fields
func (c *SyncConfig) MergeDefaults() {
	if c.Schedule == "" {
		c.Schedule = DefaultConfig().Schedule
	}
	if c.BranchMappings == nil {
		c.BranchMappings = make(map[string]string)
	}
	if len(c.BranchMappings) == 0 {
		for k, v := range DefaultConfig().BranchMappings {
			c.BranchMappings[k] = v
		}
	}
	if c.ErrorHandling.RetryAttempts == 0 {
		c.ErrorHandling.RetryAttempts = DefaultConfig().ErrorHandling.RetryAttempts
	}
	if c.ErrorHandling.RetryDelay == "" {
		c.ErrorHandling.RetryDelay = DefaultConfig().ErrorHandling.RetryDelay
	}
}

// Validate checks if the configuration is valid
func (c *SyncConfig) Validate() error {
	if err := ValidateRepoFormat(c.SourceRepo); err != nil {
		return fmt.Errorf("invalid source repository: %w", err)
	}
	if err := ValidateRepoFormat(c.TargetRepo); err != nil {
		return fmt.Errorf("invalid target repository: %w", err)
	}
	if c.Schedule != "" {
		if err := ValidateSchedule(c.Schedule); err != nil {
			return fmt.Errorf("invalid schedule: %w", err)
		}
	}
	if c.ErrorHandling.RetryAttempts < 0 {
		return fmt.Errorf("retry attempts cannot be negative")
	}
	if c.ErrorHandling.Notify && c.ErrorHandling.NotifyEmail == "" {
		return fmt.Errorf("notify email is required when notifications are enabled")
	}
	return nil
}

// ValidateRepoFormat validates the owner/repo format
func ValidateRepoFormat(repo string) error {
	if repo == "" {
		return fmt.Errorf("repository cannot be empty")
	}

	parts := strings.Split(repo, "/")
	if len(parts) != 2 {
		return fmt.Errorf("invalid repository format, expected 'owner/repo'")
	}

	if parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("both owner and repo must be non-empty")
	}

	return nil
}

// ValidateSchedule validates a cron schedule expression
func ValidateSchedule(schedule string) error {
	if schedule == "" {
		return fmt.Errorf("schedule cannot be empty")
	}

	// Basic cron format validation (minute hour day-of-month month day-of-week)
	parts := strings.Fields(schedule)
	if len(parts) != 5 {
		return fmt.Errorf("invalid cron format, expected 5 fields (minute hour day-of-month month day-of-week)")
	}

	// Validate each field
	for i, part := range parts {
		if err := validateCronField(part, i); err != nil {
			return fmt.Errorf("invalid cron field %d: %w", i+1, err)
		}
	}

	return nil
}

// ParseBranchMapping parses a branch mapping string in the format "source:target"
func ParseBranchMapping(mapping string) (source, target string, err error) {
	parts := strings.Split(mapping, ":")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid branch mapping format, expected 'source:target'")
	}

	source = strings.TrimSpace(parts[0])
	target = strings.TrimSpace(parts[1])

	if source == "" || target == "" {
		return "", "", fmt.Errorf("both source and target branches must be non-empty")
	}

	return source, target, nil
}

// validateCronField validates a single field in a cron expression
func validateCronField(field string, position int) error {
	// Define patterns for each field
	patterns := []struct {
		pattern string
		ranges  []int
	}{
		{`^(\*|[0-9]|[1-5][0-9])$`, []int{0, 59}},                        // Minutes
		{`^(\*|[0-9]|1[0-9]|2[0-3])$`, []int{0, 23}},                    // Hours
		{`^(\*|[1-9]|[12][0-9]|3[01])$`, []int{1, 31}},                  // Day of month
		{`^(\*|[1-9]|1[0-2])$`, []int{1, 12}},                           // Month
		{`^(\*|[0-6])$`, []int{0, 6}},                                    // Day of week
	}

	if position < 0 || position >= len(patterns) {
		return fmt.Errorf("invalid field position")
	}

	pattern := patterns[position]
	if field == "*" {
		return nil
	}

	// Handle lists and ranges
	for _, part := range strings.Split(field, ",") {
		if strings.Contains(part, "-") {
			rangeParts := strings.Split(part, "-")
			if len(rangeParts) != 2 {
				return fmt.Errorf("invalid range format")
			}
			if !regexp.MustCompile(pattern.pattern).MatchString(rangeParts[0]) ||
				!regexp.MustCompile(pattern.pattern).MatchString(rangeParts[1]) {
				return fmt.Errorf("invalid range values")
			}
		} else if !regexp.MustCompile(pattern.pattern).MatchString(part) {
			return fmt.Errorf("invalid value")
		}
	}

	return nil
}
