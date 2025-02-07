# Go-GitTools CLI Usage Guide

This guide provides detailed information about using the Go-GitTools command-line tools.

## Table of Contents
- [Building the Tools](#building-the-tools)
- [go-gittoken](#go-gittoken)
- [go-gitclone](#go-gitclone)
- [Repository Sync](#repository-sync)
- [go-gitpublish](#go-gitpublish)

## Building the Tools

The project uses Just as its build system. After installing Just, you can use the following commands:

```bash
# List all available commands
just --list

# Build all tools
just build

# Run all tests
just test

# Clean build artifacts
just clean
```

All built binaries are placed in the `bin/` directory. You can also build using Go directly:

```bash
go build -o bin/go-gittoken ./cmd/gittoken
go build -o bin/go-gitclone ./cmd/gitclone
go build -o bin/go-gitpublish ./cmd/gitpublish
```

## go-gittoken

`go-gittoken` is a tool for managing Git authentication tokens. It provides an interactive setup process and validates tokens before storing them.

### Usage
```bash
go-gittoken setup [flags]
```

### Flags
- `-p, --provider`: Git provider (GITHUB, GITLAB, AZURE)
- `-t, --token`: Token value
- `-s, --scope`: Token scope/permissions
- `-e, --expires`: Token expiration (e.g., 30d, 1y)
- `-f, --token-file`: File containing the token value
- `-n, --non-interactive`: Run in non-interactive mode

### Environment Variables
When running in non-interactive mode, the following environment variables can be used:
- `GIT_PROVIDER`: Git provider name
- `GIT_TOKEN_VALUE`: Token value
- `GIT_TOKEN_SCOPE`: Token scope/permissions
- `GIT_TOKEN_EXPIRY`: Token expiration

### Examples
```bash
# Interactive token setup
go-gittoken setup

# Non-interactive setup with flags
go-gittoken setup \
  --non-interactive \
  --provider GITHUB \
  --token ghp_your_token \
  --scope "repo,workflow" \
  --expires 90d

# Using environment variables
export GIT_PROVIDER=GITHUB
export GIT_TOKEN_VALUE=ghp_your_token
export GIT_TOKEN_SCOPE=repo,workflow
go-gittoken setup --non-interactive

# Using token file
echo "ghp_your_token" > token.txt
chmod 600 token.txt
go-gittoken setup \
  --provider GITHUB \
  --scope "repo,workflow" \
  --token-file token.txt
```

### Security Notes
1. When using a token file:
   - Ensure the file has secure permissions (600)
   - The tool will warn if permissions are too open
   - Delete the file after token setup

2. Token Validation:
   - Validates token format and minimum length
   - Checks for required scopes based on provider
   - Warns when token is near expiration (30 days)
   - Verifies token with provider's API

## go-gitclone

`go-gitclone` is a tool for cloning public repositories to private repositories while maintaining proper remote configuration and authentication. It automatically creates a private repository with a default naming scheme or a custom name.

### Usage
```bash
go-gitclone [source-repo-url] [flags]
```

### Arguments
- `source-repo-url`: URL of the source public repository (required)

### Flags
- `--name`: Custom name for the target repository (optional)
- `--token`: GitHub token for authentication (required)

### Examples
```bash
# Clone with default naming (creates private-repo)
go-gitclone https://github.com/user/repo.git

# Clone with custom repository name
go-gitclone https://github.com/user/repo.git --name custom-repo

# Specify token directly (if not using go-gittoken)
go-gitclone https://github.com/user/repo.git --token ghp_your_token
```

### Default Behavior
- Target repository is automatically created in your GitHub account
- Default naming scheme: `private-{original-repo-name}`
- Repository is created as private
- Workflows are removed for security
- Authentication is handled automatically using stored token or --token flag

## Repository Sync

Repository synchronization is now handled through GitHub Actions. See [docs/github-actions-sync.md](github-actions-sync.md) for setup and usage instructions.

Features include:
- Automated repository synchronization
- Configurable sync schedules
- Branch mapping support
- Error handling and notifications
- Manual trigger option

## go-gitpublish

`go-gitpublish` is a tool for publishing changes from private repositories to public forks, with optional pull request creation.

### Usage
```bash
go-gitpublish [flags]
```

### Flags
- `--private`: Private repository path (required)
- `--public`: Public fork repository URL (required)
- `--branch`: Branch to publish (default: "main")
- `--create-fork`: Create a fork if it doesn't exist
- `--pr`: Create a pull request after publishing
- `--pr-title`: Title for the pull request (required if --pr is set)
- `--pr-desc`: Description for the pull request
- `--target-branch`: Target branch for the pull request (default: "main")

### Examples
```bash
# Basic publish operation
go-gitpublish \
  --private https://github.com/user/private-repo \
  --public https://github.com/user/public-fork

# Publish with fork creation and pull request
go-gitpublish \
  --private https://github.com/user/private-repo \
  --public https://github.com/user/public-fork \
  --branch feature \
  --create-fork \
  --pr \
  --pr-title "New Feature Implementation" \
  --pr-desc "Implemented feature X with improvements to Y" \
  --target-branch main
```

## Authentication

Authentication is managed through the `go-gittoken` tool, which supports multiple Git providers and token types.

1. Set up authentication using the interactive guide:
   ```bash
   go-gittoken setup
   ```

2. Or configure a token directly:
   ```bash
   go-gittoken setup \
     --provider GITHUB \
     --token your_token \
     --scope "repo,workflow"
   ```

The token will be automatically used by other tools through environment variables.

### Token Scopes

Required scopes for different operations:
- Private repositories: `repo`
- Public repositories: `public_repo`
- Workflow actions: `workflow`
- Fork creation: `repo`

## Error Handling

All tools provide detailed error messages and proper exit codes:
- Exit code 0: Success
- Exit code 1: Error occurred
- Exit code 2: Repository already exists (for go-gitclone)

Error messages include:
- Missing required flags
- Invalid repository URLs
- Authentication failures
- Network issues
- Git operation failures

## Best Practices

1. **Token Security**
   - Use `go-gittoken` for secure token management
   - Never commit tokens to version control
   - Rotate tokens regularly using the expiration feature

2. **Branch Management**
   - Use branch mapping for clear source-to-target relationships
   - Keep branch names consistent across repositories

3. **Continuous Sync**
   - Use reasonable sync intervals (1 hour minimum)
   - Monitor sync logs for potential issues
   - Set up proper error notifications

4. **Pull Requests**
   - Provide clear, descriptive titles
   - Include detailed descriptions
   - Reference related issues or documentation
