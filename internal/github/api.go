package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/NicabarNimble/go-gittools/internal/token"
)

const (
	apiBaseURL = "https://api.github.com"
	userAgent  = "go-gittools/1.0"
)

// UserInfo represents GitHub user information
type UserInfo struct {
	Login string `json:"login"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// Client handles GitHub API operations
type Client struct {
	httpClient *http.Client
	token      string
	baseURL    string // Allow custom base URL for testing
	username   string // Cached username after validation
}

// GitHubClient is an alias for Client to maintain backward compatibility
type GitHubClient = Client

// WorkflowRun represents a GitHub Actions workflow run
type WorkflowRun struct {
	ID         int64     `json:"id"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
	LogsURL    string    `json:"logs_url"`
}

// RepoOptions represents options for repository operations
type RepoOptions struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
	Private     bool   `json:"private,omitempty"`
	AutoInit    bool   `json:"auto_init,omitempty"`
}

// PROptions represents options for pull request operations
type PROptions struct {
	Owner string `json:"-"` // Used for routing, not sent to API
	Repo  string `json:"-"` // Used for routing, not sent to API
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  string `json:"head"`
	Base  string `json:"base"`
}

// NewClient creates a new GitHub API client with token validation
func NewClient(ctx context.Context, t *token.Token) (*Client, error) {
	client := &Client{
		httpClient: &http.Client{Timeout: 30 * time.Second},
		token:      t.Value,
		baseURL:    apiBaseURL,
	}

	validator := &TokenValidator{baseURL: client.baseURL}
	if err := validator.Validate(ctx, t); err != nil {
		return nil, fmt.Errorf("token validation failed: %w", err)
	}

	// Get and cache username during client creation
	userInfo, err := client.GetUserInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	client.username = userInfo.Login

	return client, nil
}

// GetUserInfo retrieves authenticated user information
func (c *Client) GetUserInfo(ctx context.Context) (*UserInfo, error) {
	url := fmt.Sprintf("%s/user", c.baseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}
	defer resp.Body.Close()

	var userInfo UserInfo
	if err := json.NewDecoder(resp.Body).Decode(&userInfo); err != nil {
		return nil, fmt.Errorf("failed to decode user info: %w", err)
	}

	return &userInfo, nil
}

// GetUsername returns the cached username
func (c *Client) GetUsername() string {
	return c.username
}

// CreateOrUpdateWorkflow creates or updates a workflow file in the repository
func (c *Client) CreateOrUpdateWorkflow(ctx context.Context, owner, repo, path string, content []byte) error {
	url := fmt.Sprintf("%s/repos/%s/%s/contents/%s", c.baseURL, owner, repo, path)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.sendRequest(req)
	if err != nil && resp != nil && resp.StatusCode != http.StatusNotFound {
		return fmt.Errorf("failed to check workflow existence: %w", err)
	}

	// Prepare the request body
	body := map[string]interface{}{
		"message": "Update workflow file",
		"content": content,
	}

	if resp != nil && resp.StatusCode != http.StatusNotFound {
		// File exists, need to include sha
		var fileInfo struct {
			SHA string `json:"sha"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&fileInfo); err != nil {
			return fmt.Errorf("failed to decode file info: %w", err)
		}
		body["sha"] = fileInfo.SHA
	}

	// Create or update the file
	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err = http.NewRequestWithContext(ctx, "PUT", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if _, err = c.sendRequest(req); err != nil {
		return fmt.Errorf("failed to update workflow: %w", err)
	}

	return nil
}

// TriggerWorkflow triggers a workflow_dispatch event
func (c *Client) TriggerWorkflow(ctx context.Context, owner, repo, workflowID string, inputs map[string]interface{}) error {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%s/dispatches", c.baseURL, owner, repo, workflowID)
	body := map[string]interface{}{
		"ref":    "main",
		"inputs": inputs,
	}

	jsonBody, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if _, err = c.sendRequest(req); err != nil {
		return fmt.Errorf("failed to trigger workflow: %w", err)
	}

	return nil
}

// GetWorkflowRun gets the status of a workflow run
func (c *Client) GetWorkflowRun(ctx context.Context, owner, repo string, runID int64) (*WorkflowRun, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%d", c.baseURL, owner, repo, runID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow run: %w", err)
	}

	var run WorkflowRun
	if err := json.NewDecoder(resp.Body).Decode(&run); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &run, nil
}

// GetWorkflowLogs gets the logs for a workflow run
func (c *Client) GetWorkflowLogs(ctx context.Context, owner, repo string, runID int64) ([]byte, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/runs/%d/logs", c.baseURL, owner, repo, runID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to get workflow logs: %w", err)
	}

	logs, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read logs: %w", err)
	}

	return logs, nil
}

// ListWorkflowRuns lists recent workflow runs
func (c *Client) ListWorkflowRuns(ctx context.Context, owner, repo, workflowID string) ([]WorkflowRun, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/actions/workflows/%s/runs", c.baseURL, owner, repo, workflowID)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.sendRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to list workflow runs: %w", err)
	}

	var response struct {
		WorkflowRuns []WorkflowRun `json:"workflow_runs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return response.WorkflowRuns, nil
}

// CreateRepository creates a new repository
func (c *Client) CreateRepository(ctx context.Context, opts RepoOptions) error {
	url := fmt.Sprintf("%s/user/repos", c.baseURL)
	jsonBody, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if _, err = c.sendRequest(req); err != nil {
		return fmt.Errorf("failed to create repository: %w", err)
	}

	return nil
}

// CreateFork creates a fork of a repository
func (c *Client) CreateFork(ctx context.Context, repoString string) error {
	owner, repo, err := ParseRepo(repoString)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("%s/repos/%s/%s/forks", c.baseURL, owner, repo)
	req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if _, err = c.sendRequest(req); err != nil {
		return fmt.Errorf("failed to create fork: %w", err)
	}

	return nil
}

// CreatePullRequest creates a new pull request
func (c *Client) CreatePullRequest(ctx context.Context, opts PROptions) error {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls", c.baseURL, opts.Owner, opts.Repo)
	jsonBody, err := json.Marshal(opts)
	if err != nil {
		return fmt.Errorf("failed to marshal request body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(jsonBody))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	if _, err = c.sendRequest(req); err != nil {
		return fmt.Errorf("failed to create pull request: %w", err)
	}

	return nil
}

// makeRequest is an alias for sendRequest to maintain backward compatibility
func (c *Client) makeRequest(req *http.Request) (*http.Response, error) {
	return c.sendRequest(req)
}

// sendRequest sends an HTTP request with the necessary headers
func (c *Client) sendRequest(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	req.Header.Set("User-Agent", userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return resp, fmt.Errorf("GitHub API error: %s: %s", resp.Status, string(body))
	}

	return resp, nil
}

// ParseRepo parses an owner/repo string into separate owner and repo parts
func ParseRepo(repoString string) (owner, repo string, err error) {
	parts := strings.Split(repoString, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("invalid repository format: %s (expected owner/repo)", repoString)
	}
	return parts[0], parts[1], nil
}
