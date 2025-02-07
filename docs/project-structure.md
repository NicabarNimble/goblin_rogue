# Project Structure

This document outlines the organization and structure of the go-gittools project.

## Directory Layout

```
.
├── .github/                # GitHub specific configurations
│   └── workflows/         # GitHub Actions workflows
│       └── sync-repos.yml # Repository sync workflow
│
├── bin/                    # Binary executables
│   ├── gitclone          # Alternative name for go-gitclone
│   ├── go-gitclone       # Compiled gitclone binary
│   ├── go-gitpublish     # Compiled gitpublish binary
│   ├── go-gitsync        # Compiled gitsync binary
│   └── go-gittoken       # Compiled gittoken binary
│
├── cmd/                    # Command-line applications
│   ├── gitclone/          # Git repository cloning tool
│   │   ├── main.go
│   │   └── main_test.go
│   ├── gitpublish/        # Git repository publishing tool
│   │   ├── main.go
│   │   └── main_test.go
│   ├── gitsync/           # Repository synchronization tool
│   │   ├── configure.go
│   │   ├── configure_test.go
│   │   ├── init.go
│   │   ├── init_test.go
│   │   ├── logs.go
│   │   ├── logs_test.go
│   │   ├── main.go
│   │   ├── main_test.go
│   │   ├── run.go
│   │   ├── run_test.go
│   │   ├── status.go
│   │   └── status_test.go
│   └── gittoken/          # GitHub token management tool
│       ├── main.go
│       └── main_test.go
│
├── docs/                   # Project documentation
│   ├── api.md             # API reference documentation
│   ├── cli-usage.md       # Command-line usage guide
│   ├── configuration.md   # Configuration guide
│   ├── documentation-update-checklist.md
│   ├── first-time-setup.md
│   ├── github-actions-sync.md
│   └── project-structure.md
│
├── examples/              # Example implementations
│   ├── headless/         # Non-interactive usage examples
│   │   ├── README.md
│   │   ├── clone.sh      # Automated cloning script
│   │   └── setup-token.sh # Token setup automation
│   └── publish/          # Publishing workflow examples
│       ├── publish-config.json
│       └── publish_example_test.go
│
├── internal/              # Private application packages
│   ├── config/           # Configuration handling
│   │   ├── publish_config.go
│   │   ├── publish_config_test.go
│   │   ├── sync_config.go
│   │   └── sync_config_test.go
│   ├── errors/           # Error definitions and handling
│   │   ├── errors.go
│   │   ├── errors_test.go
│   │   ├── workflow_errors.go
│   │   └── workflow_errors_test.go
│   ├── git/              # Git operations implementation
│   │   ├── clone.go
│   │   ├── clone_extended_test.go
│   │   ├── clone_test.go
│   │   └── doc.go
│   ├── github/           # GitHub API integration
│   │   ├── api.go
│   │   ├── api_test.go
│   │   ├── token.go
│   │   ├── token_test.go
│   │   └── workflow.go
│   ├── gitlab/           # GitLab integration
│   │   ├── token.go
│   │   └── token_test.go
│   ├── gitops/           # Git operations utilities
│   │   └── testdata/
│   │       └── README.md
│   ├── gitutils/         # Additional git utilities
│   │   ├── clone.go
│   │   └── clone_test.go
│   ├── progress/         # Progress tracking utilities
│   │   ├── tracker.go
│   │   ├── tracker_test.go
│   │   ├── workflow.go
│   │   └── workflow_test.go
│   ├── retry/            # Retry mechanism implementation
│   ├── token/            # Token management and storage
│   │   ├── detect.go     # Token source detection
│   │   ├── detect_test.go
│   │   ├── env.go
│   │   ├── env_test.go
│   │   ├── memory.go
│   │   ├── memory_test.go
│   │   ├── refresh.go
│   │   ├── refresh_test.go
│   │   ├── storage.go
│   │   └── storage_test.go
│   └── urlutils/         # URL handling utilities
│       ├── url.go
│       └── url_test.go
│
├── tests/                # Integration tests
│   ├── integration_test.go
│   ├── sync_integration_test.go
│   └── test_helpers.go
│
├── .trunk/               # Trunk.io configuration and cache
│   ├── plugins/         # Trunk plugins
│   └── trunk.yaml       # Trunk configuration file
│
├── go.mod               # Go module definition
├── go.sum               # Go module checksums
└── justfile            # Build system tasks
```

## Package Descriptions

### Command Line Tools (cmd/)

- **gitclone**: Implements repository cloning functionality with extended features
  - Supports automatic target repository naming
  - Provides custom repository naming via --name flag
  - Handles authentication and token management
  - Includes comprehensive test coverage
- **gitpublish**: Handles repository publishing workflows
- **gitsync**: Manages repository synchronization between sources
  - Implements GitHub Actions workflow integration
  - Provides configuration management
  - Handles workflow status and logs
  - Includes comprehensive test coverage
- **gittoken**: Provides GitHub token management utilities
  - Supports interactive and non-interactive modes
  - Implements file-based token input
  - Handles environment variable configuration
  - Includes security validations and checks

### Internal Packages (internal/)

#### Core Functionality
- **git/**: Core Git operations implementation
  - Handles clone operations with extended functionality
  - Includes comprehensive testing suite
  - Contains package documentation in doc.go

#### Configuration and Settings
- **config/**: Configuration management
  - Handles publish and sync configurations
  - Includes validation and parsing logic
  - Provides test coverage for configuration scenarios

#### Error Handling
- **errors/**: Custom error types and handling
  - Defines workflow-specific errors
  - Implements error wrapping and context
  - Includes comprehensive error testing

#### GitHub Integration
- **github/**: GitHub API integration
  - Implements API client functionality
  - Handles token management and authentication
  - Provides workflow management capabilities
  - Includes comprehensive API testing

#### GitLab Integration
- **gitlab/**: GitLab API integration
  - Implements token management for GitLab
  - Includes test coverage for token operations

#### Utility Packages
- **gitops/**: Git operation utilities
  - Contains helper functions for git operations
  - Includes test data and examples

- **gitutils/**: Additional git utilities
  - Implements supplementary git functionality
  - Provides helper functions for common operations

- **progress/**: Progress tracking
  - Implements progress monitoring
  - Provides workflow status tracking
  - Handles log streaming
  - Includes progress indicators

- **retry/**: Retry mechanisms
  - Implements retry logic for failed operations
  - Handles transient failures gracefully

- **token/**: Token management
  - Implements secure token storage and retrieval
  - Handles token refresh mechanisms
  - Provides environment-based token management
  - Supports file-based token input
  - Implements token validation and security checks
  - Includes expiration management and warnings
  - Automatic token source detection
  - Smart token refresh handling

- **urlutils/**: URL handling
  - Provides URL parsing and manipulation
  - Implements URL validation and formatting

### Testing (tests/)
- Integration tests for end-to-end functionality
- Sync workflow integration tests
- Test helpers and utilities
- Comprehensive test coverage for core features

### Binary Output (bin/)
- Contains compiled executables for all tools
- Binaries are platform-specific and built during development
- Named consistently with go- prefix
- Alternative names without go- prefix also provided

### GitHub Configuration (.github/)
- Contains GitHub-specific configurations
- Includes GitHub Actions workflows for repository synchronization

### Development Tools (.trunk/)
- Trunk.io integration for development workflow
- Plugin management and configuration
- Development environment standardization

## Dependencies

The project uses Go modules for dependency management, with dependencies defined in:
- `go.mod`: Module definition and dependencies
- `go.sum`: Dependency version checksums

## Build System

The project includes a `justfile` for common development tasks and build operations.
