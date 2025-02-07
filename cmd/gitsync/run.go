package main

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/github"
	"github.com/NicabarNimble/go-gittools/internal/progress"
	"github.com/NicabarNimble/go-gittools/internal/token"
	"github.com/spf13/cobra"
)

type runOptions struct {
	repo    string
	timeout time.Duration
	wait    bool
}

func newRunCmd() *cobra.Command {
	opts := &runOptions{}

	cmd := &cobra.Command{
		Use:   "run",
		Short: "Trigger sync workflow",
		Long: `Trigger a GitHub Actions workflow to sync repositories.
The command can either trigger the workflow and exit, or wait for completion.`,
		Example: `  gitsync run --repo owner/repo
  gitsync run --repo owner/repo --wait
  gitsync run --repo owner/repo --wait --timeout 10m`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runSync(opts)
		},
	}

	cmd.Flags().StringVar(&opts.repo, "repo", "", "Repository to sync (owner/repo)")
	cmd.Flags().BoolVar(&opts.wait, "wait", false, "Wait for workflow completion")
	cmd.Flags().DurationVar(&opts.timeout, "timeout", 30*time.Minute, "Timeout duration when waiting")
	cmd.MarkFlagRequired("repo")

	return cmd
}

func runSync(opts *runOptions) error {
	// Validate repository format
	if err := github.ValidateRepoFormat(opts.repo); err != nil {
		return fmt.Errorf("invalid repository: %w", err)
	}

	// Create context with timeout if waiting
	ctx := context.Background()
	if opts.wait {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, opts.timeout)
		defer cancel()
	}

	// Initialize progress tracker
	tracker := progress.NewWorkflowTracker()

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

	client, err := github.NewClient(ctx, &t)
	if err != nil {
		return fmt.Errorf("failed to create GitHub client: %w", err)
	}

	// Trigger workflow
	if err := client.TriggerWorkflow(ctx, owner, repo, "sync.yml", nil); err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	// Get the latest workflow run
	runs, err := client.ListWorkflowRuns(ctx, owner, repo, "sync.yml")
	if err != nil {
		return fmt.Errorf("failed to list workflow runs: %w", err)
	}

	if len(runs) == 0 {
		return fmt.Errorf("no workflow runs found")
	}

	latestRun := runs[0]
	workflow := tracker.StartWorkflow("Repository Sync", latestRun.ID, latestRun.ID)

	fmt.Printf("Triggered workflow run #%d\n", latestRun.ID)

	if !opts.wait {
		fmt.Printf("Run 'gitsync status --repo %s --run-id %d' to check status\n", opts.repo, latestRun.ID)
		return nil
	}

	// Monitor workflow progress
	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for workflow completion")
		default:
			run, err := client.GetWorkflowRun(ctx, owner, repo, latestRun.ID)
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

			// Poll every 5 seconds
			time.Sleep(5 * time.Second)
		}
	}
}
