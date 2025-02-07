# go-gittools

A suite of Go-based CLI tools and GitHub Actions workflows for managing advanced Git workflows, particularly focused on maintaining private experimental versions of public repositories while enabling selective contributions back to public repositories.

## Tools

### go-gitclone
- **Purpose**: Clone public repositories to private repositories
- **Features**:
  - Automatic target repository naming
  - Custom repository name support
  - Secure authentication handling
  - Remote configuration setup
  - Comprehensive error reporting

### go-gitsync & Repository Sync
- **Purpose**: Maintain synchronization with upstream repositories
- **Features**:
  - Full CLI interface for workflow management
  - Automated sync via GitHub Actions
  - Advanced branch mapping via JSON configuration
  - Configurable sync schedules (6-hour default)
  - Manual trigger option via workflow_dispatch
  - Real-time workflow status tracking
  - Detailed workflow logs
  - Comprehensive error handling
  - Retry mechanisms for transient failures

### go-gitpublish
- **Purpose**: Manage public contributions from private repositories
- **Features**:
  - Publication workflow management
  - Automatic fork creation
  - Pull request generation
  - Branch management

## Quick Start

1. **Installation**
   - Ensure Go is installed (version 1.23.4 or later)
   - Install Just (optional but recommended):
     ```bash
     # macOS
     brew install just

     # Ubuntu/Debian
     apt install just

     # Windows
     winget install just
     ```
   - Clone and build:
     ```bash
     git clone https://github.com/NicabarNimble/go-gittools.git
     cd go-gittools
     just build  # or 'go build ./cmd/...'
     ```

2. **First-Time Setup**
   - Follow our [First-Time Setup Guide](docs/first-time-setup.md) to:
     - Create your GitHub token
     - Configure token securely
     - Verify your setup
   - For automated environments, see our headless setup instructions in the guide

3. **Verify Installation**
   ```bash
   # Test your setup with a simple clone operation
   go-gitclone https://github.com/example/repo.git
   ```

## Usage

### go-gitclone
```bash
# Basic usage (creates private-repo)
go-gitclone https://github.com/example/repo.git

# With custom repository name
go-gitclone https://github.com/example/repo.git --name custom-repo

# With explicit token (if not using go-gittoken)
go-gitclone https://github.com/example/repo.git --token ghp_your_token
```

### go-gitsync & Repository Sync
```bash
# Initialize sync workflow
go-gitsync init --source user/repo --target fork/repo

# Configure sync settings
go-gitsync configure --repo user/repo --schedule "0 0 * * *" --branch-map main:main,dev:develop

# Trigger manual sync
go-gitsync run --repo user/repo

# Check sync status
go-gitsync status --repo user/repo --watch

# View sync logs
go-gitsync logs --repo user/repo --run-id 12345 --follow
```

For detailed setup and usage instructions, see:
- [GitHub Actions Sync Documentation](docs/github-actions-sync.md)

### go-gitpublish
```bash
# Basic publish
go-gitpublish --private https://github.com/user/private-repo --public https://github.com/user/public-fork

# Publish with PR creation
go-gitpublish \
  --private https://github.com/user/private-repo \
  --public https://github.com/user/public-fork \
  --token ghp_your_token \
  --branch feature \
  --create-fork \
  --pr \
  --pr-title "New Feature Implementation" \
  --pr-desc "Implemented feature X with improvements"
```

## Configuration

All tools support both command-line flags and configuration files.

### Configuration Files

- **Sync Configuration** (`sync-config.json`):
  ```json
  {
    "sourceRepo": "user/repo",
    "targetRepo": "fork/repo",
    "schedule": "0 */6 * * *",
    "branchMappings": {
      "main": "main",
      "develop": "dev"
    },
    "errorHandling": {
      "notifyOnError": true,
      "retryAttempts": 3,
      "retryDelay": "5m"
    }
  }
  ```

- **Publish Configuration** (`publish-config.json`):
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
    "pullRequest": {
      "enabled": true,
      "title": "Feature Implementation",
      "targetBranch": "main"
    }
  }
  ```

### Authentication

Authentication is handled through the `go-gittoken` tool which manages tokens securely with automatic source detection:

```bash
# First-time interactive setup
go-gittoken setup

# Non-interactive setup with token file
go-gittoken setup --token-file /path/to/token.txt

# Direct token setup
go-gittoken setup \
  --provider GITHUB \
  --token ghp_your_token \
  --scope "repo,workflow"
```

The tool supports multiple token storage methods:
1. Environment variables (with `GIT_TOKEN_` prefix)
2. Token files (automatically detected)
3. Memory-based storage for temporary sessions

Example environment variable format:
```bash
# GitHub token
export GIT_TOKEN_GITHUB='{"value":"ghp_your_token","scope":"repo,workflow","refresh":true}'

# GitLab token
export GIT_TOKEN_GITLAB='{"value":"glpat-token","scope":"api"}'

# Repository configuration
export PRIVATE_REPO="https://github.com/user/private-repo"
export PUBLIC_FORK="https://github.com/user/public-fork"
```

For automated environments, see the headless setup script in `examples/headless/setup-token.sh`.

## Documentation

- [First-Time Setup Guide](docs/first-time-setup.md) - **Start here if you're new!**
- [CLI Usage Guide](docs/cli-usage.md) - Detailed command usage
- [Configuration Guide](docs/configuration.md) - Configuration options
- [GitHub Actions Sync](docs/github-actions-sync.md) - Repository sync setup
- [API Documentation](docs/api.md) - API integration details
- [Project Structure](docs/project-structure.md) - Codebase organization

## Development

### Project Structure

See [Project Structure Documentation](docs/project-structure.md) for a detailed overview of the codebase organization.

### Testing Structure
The project follows Go's testing conventions with:
1. Unit tests placed alongside the code they test (e.g., `file_test.go` next to `file.go`)
2. Integration tests in the dedicated `tests/` directory
3. Example tests in the `examples/` directory that serve as both documentation and functional tests
4. Extended tests for complex functionality (e.g., `clone_extended_test.go`)

Each package maintains its own tests, ensuring comprehensive coverage of:
- Unit tests for individual components
- Integration tests for end-to-end functionality
- Example tests that demonstrate usage patterns
- Workflow integration tests for GitHub Actions functionality

### Building and Testing

The project uses Just as its build system. List all available commands with:
```bash
just --list
```

Available commands:
```bash
# Build all tools
just build

# Run all tests
just test

# Clean build artifacts
just clean

# Or using Go directly:
go test ./...
go test ./internal/git/...
go build -o bin/go-gitclone ./cmd/gitclone
go install ./cmd/...
```

All built binaries are placed in the `bin/` directory.

### Error Handling

The project uses domain-specific error types from the `internal/errors` package:

```go
if err := operation(); err != nil {
    if errors.IsDomain(err, "git") {
        // Handle git-specific error
    }
}
```

### API Integration

The `internal` packages provide interfaces for custom integrations. See [API Documentation](docs/api.md) for details.

## Contributing

1. Fork the repository
2. Create a feature branch
3. Submit a pull request

Please ensure:
- Code follows Go style guidelines
- Tests are included for new features
- Documentation is updated
- Error handling follows project conventions
- Progress tracking is implemented where appropriate

## License

MIT License - See [LICENSE](LICENSE) for details
