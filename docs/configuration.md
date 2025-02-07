# Go-GitTools Configuration Guide

This guide explains how to configure Go-GitTools for various use cases.

## Table of Contents
- [Configuration Files](#configuration-files)
  - [Repository Sync (GitHub Actions)](#repository-sync)
  - [Publish Configuration](#publish-configuration)
  - [Clone Configuration](#clone-configuration)
- [Environment Variables](#environment-variables)
- [Authentication](#authentication)
- [Examples](#examples)

## Configuration Files

Go-GitTools supports JSON configuration files for publish operations and GitHub Actions for sync operations.

### Repository Sync (GitHub Actions)

Repository synchronization is configured through GitHub Actions workflow files. See [docs/github-actions-sync.md](github-actions-sync.md) for complete setup and configuration instructions.

The workflow configuration includes:
- Automated sync schedules
- Branch mapping configuration
- Error handling and notifications
- Manual trigger options

### Publish Configuration

The publish configuration file (`publish-config.json`) defines how changes should be published to public forks.

```json
{
  "private": {
    "url": "https://github.com/user/private-repo",
    "token": "ghp_your_token"
  },
  "public": {
    "url": "https://github.com/user/public-fork",
    "createFork": true
  },
  "branch": "feature",
  "pullRequest": {
    "enabled": true,
    "title": "Feature Implementation",
    "description": "Implemented new feature with improvements",
    "targetBranch": "main"
  }
}
```

#### Fields
- `private`: Configuration for the private repository
  - `url`: Repository URL (HTTPS only)
  - `token`: GitHub token for authentication (prefer using go-gittoken)
- `public`: Configuration for the public fork
  - `url`: Repository URL (HTTPS only)
  - `createFork`: Whether to create the fork if it doesn't exist
- `branch`: Branch to publish
- `pullRequest`: Pull request configuration
  - `enabled`: Whether to create a pull request
  - `title`: Pull request title
  - `description`: Pull request description
  - `targetBranch`: Target branch for the pull request

### Clone Configuration

The `go-gitclone` tool uses a simplified configuration approach based on command-line arguments and environment variables.

#### Repository Naming

By default, `go-gitclone` automatically generates the target repository name using the format:
```
private-{original-repo-name}
```

For example:
- Source: `https://github.com/user/repo.git`
- Generated target: `private-repo`

You can override this naming convention using the `--name` flag:
```bash
go-gitclone https://github.com/user/repo.git --name custom-repo
```

#### Authentication

Authentication for `go-gitclone` can be configured in two ways:

1. Using `go-gittoken` (recommended):
   ```bash
   # Set up token once
   go-gittoken setup

   # Use go-gitclone without explicit token
   go-gitclone https://github.com/user/repo.git
   ```

2. Direct token usage:
   ```bash
   # Use token directly
   go-gitclone https://github.com/user/repo.git --token ghp_your_token

   # Or via environment variable
   export GIT_TOKEN_GITHUB=ghp_your_token
   go-gitclone https://github.com/user/repo.git
   ```

## Authentication

Authentication is managed through the `go-gittoken` tool, which provides secure token storage and management.

### Token Setup

1. **Interactive Setup**
   ```bash
   go-gittoken setup
   ```
   This will guide you through the process of configuring a token for your Git provider.

2. **Non-interactive Setup**
   ```bash
   # Using command-line flags
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
   export GIT_TOKEN_EXPIRY=90d
   go-gittoken setup --non-interactive

   # Using token file
   echo "ghp_your_token" > token.txt
   chmod 600 token.txt
   go-gittoken setup \
     --provider GITHUB \
     --scope "repo,workflow" \
     --token-file token.txt
   ```

### Token Security

1. **File-based Input**
   - Use `--token-file` for secure token input
   - Ensure file has secure permissions (600)
   - Delete token file after setup
   - Tool warns about insecure permissions

2. **Token Validation**
   - Validates token format and minimum length
   - Verifies required scopes for provider
   - Checks token with provider's API
   - Warns when token nears expiration (30 days)

### Token Storage

The `go-gittoken` tool stores tokens securely using environment variables with the `GIT_TOKEN_` prefix:

```bash
# Format of stored tokens
export GIT_TOKEN_GITHUB='{"Value":"ghp_your_token","Scope":"repo,workflow"}'
export GIT_TOKEN_GITLAB='{"Value":"glpat_token","Scope":"api"}'
```

For headless and automated environments:
- Use non-interactive mode with environment variables
- Configure tokens via CI/CD secrets
- Use token files for secure input
- Implement regular token rotation

### Token Scopes

Different operations require different token scopes:

1. **Private Repository Access**
   - Required scope: `repo`
   - Allows read/write access to private repositories

2. **Public Repository Access**
   - Required scope: `public_repo`
   - Allows operations on public repositories

3. **Fork Creation**
   - Required scope: `repo` or `public_repo`
   - Depends on whether the target is private or public

4. **Workflow Actions**
   - Required scope: `workflow`
   - Needed for workflow-related operations

### Best Practices

1. **Token Management**
   - Use `go-gittoken` for secure storage
   - Set appropriate expiration times
   - Use minimal required scopes
   - Rotate tokens regularly

2. **CI/CD Systems**
   - Use secrets management
   - Configure tokens via environment variables
   - Consider using shorter expiration times

## Environment Variables

Additional configuration can be provided through environment variables:

```bash
# Authentication
export GIT_TOKEN_GITHUB=ghp_your_token

# Repository URLs
export PRIVATE_REPO="https://github.com/user/private-repo"
export PUBLIC_FORK="https://github.com/user/public-fork"

# Pull Request Configuration
export PR_TITLE="Feature Implementation"
export PR_DESCRIPTION="Implemented new feature"
```

Environment variables take precedence over configuration file values.

## Examples

### Basic Publish Configuration

```json
{
  "private": {
    "url": "https://github.com/user/private-repo",
    "token": "ghp_your_token"
  },
  "public": {
    "url": "https://github.com/user/public-fork"
  },
  "branch": "main"
}
```

### Publish with Pull Request

```json
{
  "private": {
    "url": "https://github.com/user/private-repo",
    "token": "ghp_your_token"
  },
  "public": {
    "url": "https://github.com/user/public-fork",
    "createFork": true
  },
  "branch": "feature",
  "pullRequest": {
    "enabled": true,
    "title": "New Feature Implementation",
    "description": "- Added feature X\n- Improved performance\n- Fixed bug Y",
    "targetBranch": "main"
  }
}
