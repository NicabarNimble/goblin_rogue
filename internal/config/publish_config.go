package config

import (
	"encoding/json"
	"fmt"
	"os"

	"github.com/NicabarNimble/go-gittools/internal/errors"
)

// PublishConfig holds configuration for repository publishing
type PublishConfig struct {
	PrivateRepo string `json:"privateRepo"`
	PublicFork  string `json:"publicFork"`
	Branch      string `json:"branch"`
	Token       string `json:"token,omitempty"`
}

// LoadPublishConfig loads configuration from a JSON file
func LoadPublishConfig(path string) (*PublishConfig, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.New("config", fmt.Errorf("failed to read config file: %w", err))
	}

	var config PublishConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, errors.New("config", fmt.Errorf("failed to parse config file: %w", err))
	}

	if err := config.validate(); err != nil {
		return nil, err
	}

	return &config, nil
}

// SavePublishConfig saves configuration to a JSON file
func (c *PublishConfig) SavePublishConfig(path string) error {
	if err := c.validate(); err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return errors.New("config", fmt.Errorf("failed to marshal config: %w", err))
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return errors.New("config", fmt.Errorf("failed to write config file: %w", err))
	}

	return nil
}

func (c *PublishConfig) validate() error {
	if c.PrivateRepo == "" {
		return errors.New("config", fmt.Errorf("private repository is required"))
	}
	if c.PublicFork == "" {
		return errors.New("config", fmt.Errorf("public fork is required"))
	}
	if c.Branch == "" {
		c.Branch = "main" // Set default branch if not specified
	}
	return nil
}

// DefaultConfig returns a PublishConfig with default values
func DefaultPublishConfig() *PublishConfig {
	return &PublishConfig{
		Branch: "main",
	}
}
