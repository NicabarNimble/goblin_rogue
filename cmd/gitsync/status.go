package main

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/github"
	"github.com/NicabarNimble/go-gittools/internal/progress"
	"github.com/NicabarNimble/go-gittools/internal/token"
	"github.com/spf13/cobra"
)

type statusOptions struct {
	repo   string
	runID  string
	watch  bool
	format string
}

func newStatusCmd() *cobra.Command {
	opts := &statusOptions{}

	cmd := &cobra.Command{
		Use:   "status",
		Short: "Check workflow status",
		Long: `Check the status of a running or completed sync workflow.
Optionally watch the workflow progress in real-time.`,
		Example: `  gitsync status --repo owner/repo --run-id 123456
  gitsync status --repo owner/repo --run-id 123456 --watch
  gitsync status --repo owner/repo --run-id 123456 --format json`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return checkStatus(opts)
		},
	}

	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository to check (owner/repo)")
	cmd.Flags().StringVar(&opts.runID, "run-id", "", "Workflow run ID")
	cmd.Flags().BoolVar(&opts.watch, "watch", false, "Watch workflow progress")
	cmd.Flags().StringVar(&opts.format, "format", "text", "Output format (text or json)")
	cmd.MarkFlagRequired("repo")
	cmd.MarkFlagRequired("run-id")

	return cmd
}

func checkStatus(opts *statusOptions) error {
	// Parse run ID
	runID, err := strconv.ParseInt(opts.runID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid run ID: %w", err)
	}

	// Create context
	ctx := context.Background()

	// Initialize progress tracker if watching
	var tracker *progress.WorkflowTracker
	if opts.watch {
		tracker = progress.NewWorkflowTracker()
	}

	// Get and validate GitHub token
	storage := token.NewEnvStorage()
	t, err := storage.Retrieve(ctx, "GITHUB")
	if err != nil {
		if errors.Is(err, token.ErrTokenNotFound) {
			return fmt.Errorf("GitHub token not found in environment. Set GIT_TOKEN_GITHUB environment variable")
		}
		if errors.Is(err, token.ErrTokenExpired) {
			return fmt.Errorf("GitHub token has expired. Please refresh or provide a new token")
		}
		if errors.Is(err, token.ErrTokenInvalid) {
			return fmt.Errorf("GitHub token is invalid. Check token format in GIT_TOKEN_GITHUB environment variable")
		}
		return fmt.Errorf("failed to get GitHub token: %w", err)
	}

	// Pre-validate token before creating client
	validator := github.NewTokenValidator()
	if err := validator.Validate(ctx, &t); err != nil {
		if strings.Contains(err.Error(), "missing required scopes") {
			return fmt.Errorf("GitHub token is missing required scopes (repo, workflow, admin:repo). Please check token permissions")
		}
		return fmt.Errorf("GitHub token validation failed: %w", err)
	}

	// Parse owner and repo
	owner, repo, err := github.ParseRepo(opts.repo)
	if err != nil {
		return fmt.Errorf("failed to parse repository: %w", err)
	}

	// Create GitHub client
	client, err := github.NewClient(ctx, &t)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Get workflow run
	run, err := client.GetWorkflowRun(ctx, owner, repo, runID)
	if err != nil {
		return fmt.Errorf("failed to get workflow run: %w", err)
	}

	if !opts.watch {
		// Single status check
		if opts.format == "json" {
			fmt.Printf(`{"id":%d,"status":"%s","conclusion":"%s","created_at":"%s","updated_at":"%s"}`,
				run.ID, run.Status, run.Conclusion, run.CreatedAt.Format(time.RFC3339),
				run.UpdatedAt.Format(time.RFC3339))
		} else {
			fmt.Printf("Workflow run #%d\n", run.ID)
			fmt.Printf("Status: %s\n", run.Status)
			if run.Conclusion != "" {
				fmt.Printf("Conclusion: %s\n", run.Conclusion)
			}
			fmt.Printf("Created: %s\n", run.CreatedAt.Format(time.RFC3339))
			fmt.Printf("Updated: %s\n", run.UpdatedAt.Format(time.RFC3339))
		}
		return nil
	}

	// Watch mode
	workflow := tracker.StartWorkflow("Repository Sync", run.ID, run.ID)

	for {
		run, err := client.GetWorkflowRun(ctx, owner, repo, runID)
		if err != nil {
			return fmt.Errorf("failed to get workflow status: %w", err)
		}

		switch run.Status {
		case "completed":
			if run.Conclusion == "success" {
				workflow.Status = progress.WorkflowCompleted
				tracker.UpdateWorkflowStatus(progress.WorkflowCompleted)
				return nil
			}
			workflow.Status = progress.WorkflowFailed
			tracker.UpdateWorkflowStatus(progress.WorkflowFailed)
			return fmt.Errorf("workflow failed with conclusion: %s", run.Conclusion)
		case "queued":
			workflow.Status = progress.WorkflowQueued
			tracker.UpdateWorkflowStatus(progress.WorkflowQueued)
		default:
			workflow.Status = progress.WorkflowInProgress
			tracker.UpdateWorkflowStatus(progress.WorkflowInProgress)
		}

		time.Sleep(5 * time.Second)
	}
}
