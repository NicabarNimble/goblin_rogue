package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/NicabarNimble/go-gittools/internal/github"
	"github.com/NicabarNimble/go-gittools/internal/progress"
	"github.com/NicabarNimble/go-gittools/internal/token"
	"github.com/spf13/cobra"
)

type logsOptions struct {
	repo     string
	runID    string
	output   string
	follow   bool
	tailNum  int
}

func newLogsCmd() *cobra.Command {
	opts := &logsOptions{}

	cmd := &cobra.Command{
		Use:   "logs",
		Short: "View workflow logs",
		Long: `View logs from a sync workflow run.
Logs can be displayed in the terminal or saved to a file.`,
		Example: `  gitsync logs --repo owner/repo --run-id 123456
  gitsync logs --repo owner/repo --run-id 123456 --output workflow.log
  gitsync logs --repo owner/repo --run-id 123456 --follow
  gitsync logs --repo owner/repo --run-id 123456 --tail 100`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return fetchLogs(opts)
		},
	}

	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository to fetch logs from (owner/repo)")
	cmd.Flags().StringVar(&opts.runID, "run-id", "", "Workflow run ID")
	cmd.Flags().StringVar(&opts.output, "output", "", "Output file (default: stdout)")
	cmd.Flags().BoolVar(&opts.follow, "follow", false, "Follow log output")
	cmd.Flags().IntVar(&opts.tailNum, "tail", 0, "Number of lines to show from the end (0 for all)")
	cmd.MarkFlagRequired("repo")
	cmd.MarkFlagRequired("run-id")

	return cmd
}

func fetchLogs(opts *logsOptions) error {
	// Parse run ID
	runID, err := strconv.ParseInt(opts.runID, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid run ID: %w", err)
	}

	// Create context
	ctx := context.Background()

	// Initialize progress tracker if following logs
	var tracker *progress.WorkflowTracker
	if opts.follow {
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

	// Get workflow run to check status
	run, err := client.GetWorkflowRun(ctx, owner, repo, runID)
	if err != nil {
		return fmt.Errorf("failed to get workflow run: %w", err)
	}

	// Prepare output writer
	var out io.Writer = os.Stdout
	if opts.output != "" {
		file, err := os.Create(opts.output)
		if err != nil {
			return fmt.Errorf("failed to create output file: %w", err)
		}
		defer file.Close()
		out = file
	}

	// Start tracking if following
	if opts.follow {
		workflow := tracker.StartWorkflow("Repository Sync", run.ID, run.ID)
		defer tracker.UpdateWorkflowStatus(progress.WorkflowCompleted)

		switch run.Status {
		case "completed":
			workflow.Status = progress.WorkflowCompleted
		case "queued":
			workflow.Status = progress.WorkflowQueued
		default:
			workflow.Status = progress.WorkflowInProgress
		}
		tracker.UpdateWorkflowStatus(workflow.Status)
	}

	// Get logs
	logs, err := client.GetWorkflowLogs(ctx, owner, repo, runID)
	if err != nil {
		return fmt.Errorf("failed to get workflow logs: %w", err)
	}

	// Write logs to output
	if _, err := out.Write(logs); err != nil {
		return fmt.Errorf("failed to write logs: %w", err)
	}

	// If following, continue to poll for new logs while the workflow is running
	if opts.follow && run.Status != "completed" {
		lastSize := len(logs)
		for {
			run, err := client.GetWorkflowRun(ctx, owner, repo, runID)
			if err != nil {
				return fmt.Errorf("failed to get workflow status: %w", err)
			}

			logs, err := client.GetWorkflowLogs(ctx, owner, repo, runID)
			if err != nil {
				return fmt.Errorf("failed to get workflow logs: %w", err)
			}

			if len(logs) > lastSize {
				if _, err := out.Write(logs[lastSize:]); err != nil {
					return fmt.Errorf("failed to write logs: %w", err)
				}
				lastSize = len(logs)
			}

			if run.Status == "completed" {
				tracker.UpdateWorkflowStatus(progress.WorkflowCompleted)
				break
			}
		}
	}

	return nil
}
