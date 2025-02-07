package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/NicabarNimble/go-gittools/internal/github"
	"github.com/spf13/cobra"
)

type initOptions struct {
	sourceRepo string
	targetRepo string
	schedule   string
	branches   []string
}

func newInitCmd() *cobra.Command {
	opts := &initOptions{}

	cmd := &cobra.Command{
		Use:   "init",
		Short: "Initialize sync workflow",
		Long: `Initialize a new GitHub Actions workflow for repository synchronization.
This command creates a workflow file in .github/workflows/ that will handle the sync process.`,
		Example: `  gitsync init --source owner/repo --target fork/repo
  gitsync init --source owner/repo --target fork/repo --schedule "0 0 * * *"
  gitsync init --source owner/repo --target fork/repo --branch main:master,dev:development`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInit(opts)
		},
	}

	cmd.Flags().StringVar(&opts.sourceRepo, "source", "", "Source repository (owner/repo)")
	cmd.Flags().StringVar(&opts.targetRepo, "target", "", "Target repository (owner/repo)")
	cmd.Flags().StringVar(&opts.schedule, "schedule", "", "Cron schedule for automated syncs (default: every 6 hours)")
	cmd.Flags().StringSliceVar(&opts.branches, "branch", nil, "Branch mappings (source:target)")

	cmd.MarkFlagRequired("source")
	cmd.MarkFlagRequired("target")

	return cmd
}

func runInit(opts *initOptions) error {
	// Validate repository formats
	if err := github.ValidateRepoFormat(opts.sourceRepo); err != nil {
		return fmt.Errorf("invalid source repository: %w", err)
	}
	if err := github.ValidateRepoFormat(opts.targetRepo); err != nil {
		return fmt.Errorf("invalid target repository: %w", err)
	}

	// Parse branch mappings
	branchMappings := make(map[string]string)
	if len(opts.branches) > 0 {
		for _, mapping := range opts.branches {
			parts := strings.Split(mapping, ":")
			if len(parts) != 2 {
				return fmt.Errorf("invalid branch mapping format: %s (expected source:target)", mapping)
			}
			branchMappings[parts[0]] = parts[1]
		}
	}

	// Generate workflow file
	data := &github.WorkflowData{
		SourceRepo:     opts.sourceRepo,
		TargetRepo:     opts.targetRepo,
		Schedule:       opts.schedule,
		BranchMappings: branchMappings,
		ErrorHandling:  true,
	}

	workflow, err := github.GenerateWorkflow(data)
	if err != nil {
		return fmt.Errorf("failed to generate workflow: %w", err)
	}

	// Create .github/workflows directory if it doesn't exist
	workflowDir := ".github/workflows"
	if err := os.MkdirAll(workflowDir, 0755); err != nil {
		return fmt.Errorf("failed to create workflow directory: %w", err)
	}

	// Write workflow file
	workflowPath := filepath.Join(workflowDir, "sync.yml")
	if err := os.WriteFile(workflowPath, []byte(workflow), 0644); err != nil {
		return fmt.Errorf("failed to write workflow file: %w", err)
	}

	fmt.Printf("Successfully created workflow file: %s\n", workflowPath)
	fmt.Println("Next steps:")
	fmt.Println("1. Review and commit the workflow file")
	fmt.Println("2. Ensure GITHUB_TOKEN has necessary permissions")
	fmt.Println("3. Run 'gitsync run' to trigger the workflow")

	return nil
}
