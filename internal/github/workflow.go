package github

import (
	"bytes"
	"fmt"
	"text/template"
)

// DefaultWorkflowTemplate is the default GitHub Actions workflow template for repository synchronization
const DefaultWorkflowTemplate = `name: Repository Sync

on:
  workflow_dispatch:  # For CLI triggers
  schedule:
    - cron: '{{ .Schedule }}'  # Default: Every 6 hours

jobs:
  sync:
    runs-on: ubuntu-latest
    steps:
      - name: Checkout code
        uses: actions/checkout@v4
        with:
          fetch-depth: 0

      - name: Set up Go
        uses: actions/setup-go@v4
        with:
          go-version: '1.21'

      - name: Configure Git
        run: |
          git config --global user.name 'GitHub Actions'
          git config --global user.email 'actions@github.com'

      - name: Run sync operation
        env:
          GITHUB_TOKEN: "${{ secrets.GITHUB_TOKEN }}"
          SOURCE_REPO: {{ .SourceRepo }}
          TARGET_REPO: {{ .TargetRepo }}
          {{- range $key, $value := .BranchMappings }}
          BRANCH_MAP_{{ $key }}: {{ $value }}
          {{- end }}
        run: |
          go run ./cmd/gitsync sync \
            --source $SOURCE_REPO \
            --target $TARGET_REPO \
            {{- range $key, $value := .BranchMappings }}
            --branch-map {{ $key }}:{{ $value }} \
            {{- end }}

      - name: Handle errors
        if: failure()
        uses: actions/github-script@v7
        with:
          script: |
            const issue = await github.rest.issues.create({
              owner: context.repo.owner,
              repo: context.repo.repo,
              title: 'Sync workflow failed',
              body: 'The repository sync workflow failed. Please check the workflow logs for details.'
            });
            console.log('Created issue #' + issue.data.number);`

// WorkflowData represents the data needed to generate a workflow file
type WorkflowData struct {
	SourceRepo      string
	TargetRepo      string
	Schedule        string
	BranchMappings  map[string]string
	ErrorHandling   bool
}

// GenerateWorkflow generates a workflow file from the template and data
func GenerateWorkflow(data *WorkflowData) (string, error) {
	if data.Schedule == "" {
		data.Schedule = "0 */6 * * *" // Default: Every 6 hours
	}

	tmpl, err := template.New("workflow").Parse(DefaultWorkflowTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse workflow template: %w", err)
	}

	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("failed to execute workflow template: %w", err)
	}

	return buf.String(), nil
}

// ValidateRepoFormat validates the owner/repo format
func ValidateRepoFormat(repo string) error {
	if repo == "" {
		return fmt.Errorf("repository cannot be empty")
	}

	// Check for owner/repo format
	if len(bytes.Split([]byte(repo), []byte("/"))) != 2 {
		return fmt.Errorf("invalid repository format, expected 'owner/repo'")
	}

	return nil
}
