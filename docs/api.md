# Go-GitTools API Documentation

This document describes the public API provided by Go-GitTools for integration into other applications.

## Table of Contents
- [Package Overview](#package-overview)
- [Git Operations](#git-operations)
- [Progress Tracking](#progress-tracking)
- [Error Handling](#error-handling)

## Package Overview

Go-GitTools provides several packages under `internal`:

```
internal/
├── errors/    # Error handling utilities
├── git/       # Git operations interface
└── progress/  # Progress tracking interface
```

## Git Operations

The `git` package provides functionality for Git repository operations, primarily focused on cloning repositories.

### CloneOptions

The `CloneOptions` struct configures the repository cloning operation:

```go
type CloneOptions struct {
    SourceURL  string            // URL of the source repository
    TargetURL  string            // Optional URL of the target repository
    WorkingDir string            // Optional working directory for the clone
    Token      string            // Token for HTTPS authentication
    Progress   progress.Tracker  // Optional progress tracking
    Context    context.Context   // Context for cancellation/timeout
}
```

### CloneRepository Function

```go
func CloneRepository(opts CloneOptions) error
```

CloneRepository clones a source repository to either a target repository or local working directory. It supports:
- HTTPS URLs only (SSH URLs are not supported)
- Progress tracking
- Context-based cancellation and timeouts
- Authentication via tokens
- Direct cloning to a working directory
- Repository mirroring (clone to target URL)

### Usage Example

```go
import (
    "context"
    "time"

    "github.com/NicabarNimble/go-gittools/internal/git"
    "github.com/NicabarNimble/go-gittools/internal/progress"
)

func main() {
    // Create a progress tracker
    tracker := &progress.DefaultTracker{}

    // Set up clone options
    opts := git.CloneOptions{
        SourceURL:  "https://github.com/user/source-repo",
        WorkingDir: "/path/to/clone",
        Token:      "github_token",
        Progress:   tracker,
        Context:    context.Background(),
    }

    // Clone the repository
    err := git.CloneRepository(opts)
    if err != nil {
        // Handle error
    }
}
```

### Repository Mirroring Example

```go
func mirrorRepository(source, target, token string) error {
    opts := git.CloneOptions{
        SourceURL: source,
        TargetURL: target,
        Token:     token,
        Context:   context.Background(),
    }

    return git.CloneRepository(opts)
}
```

## Progress Tracking

The `progress` package provides interfaces for tracking operation progress.

### Tracker Interface

```go
type Tracker interface {
    // Start initializes progress tracking with a description
    Start(description string)

    // Update updates the current progress
    Update(current int64)

    // Increment increases the current progress by the specified amount
    Increment(amount int64)

    // GetProgress returns the current progress percentage
    GetProgress() float64

    // GetRate returns the current progress rate (units per second)
    GetRate() float64

    // GetETA returns the estimated time remaining
    GetETA() time.Duration

    // Complete marks the tracking as complete
    Complete()

    // Error records an error that occurred during the operation
    Error(err error)
}
```

### DefaultTracker Implementation

```go
import "github.com/NicabarNimble/go-gittools/internal/progress"

// Create a new tracker
tracker := &progress.DefaultTracker{}

// Start tracking
tracker.Start("Clone Repository")

// Update progress
tracker.Update(50)

// Get progress information
percentage := tracker.GetProgress() // 50.0
rate := tracker.GetRate()          // units per second
eta := tracker.GetETA()            // estimated time remaining

// Mark as complete
tracker.Complete()
```

### Custom Progress Reporter Example

```go
type ProgressReporter struct {
    progress.DefaultTracker
    lastReport time.Time
}

func (p *ProgressReporter) Update(current int64) {
    p.DefaultTracker.Update(current)

    // Report progress every second
    if time.Since(p.lastReport) >= time.Second {
        fmt.Printf(
            "Progress: %.2f%% (%.2f/s) ETA: %v\n",
            p.GetProgress(),
            p.GetRate(),
            p.GetETA().Round(time.Second),
        )
        p.lastReport = time.Now()
    }
}

func main() {
    reporter := &ProgressReporter{}

    opts := git.CloneOptions{
        SourceURL:  "https://github.com/user/repo",
        WorkingDir: "path",
        Token:      "token",
        Progress:   reporter,
    }

    err := git.CloneRepository(opts)
    if err != nil {
        fmt.Printf("Error: %v\n", err)
    }
}
```

## Error Handling

The `errors` package provides domain-specific error types and utilities.

### Error Types

```go
import "github.com/NicabarNimble/go-gittools/internal/errors"

// Create a domain error
err := errors.New("git", fmt.Errorf("failed to clone repository"))

// Check error domain
if errors.IsDomain(err, "git") {
    // Handle git-specific error
}

// Get original error
originalErr := errors.Unwrap(err)
```

### Error Handling Example

```go
func cloneRepository(url, path string) error {
    if url == "" {
        return errors.New("validation", fmt.Errorf("url cannot be empty"))
    }

    opts := git.CloneOptions{
        SourceURL:  url,
        WorkingDir: path,
    }

    if err := git.CloneRepository(opts); err != nil {
        // Wrap the error with domain context
        return errors.New("clone", fmt.Errorf("failed to clone %s: %w", url, err))
    }

    return nil
}

func main() {
    err := cloneRepository("", "")
    if errors.IsDomain(err, "validation") {
        // Handle validation error
    } else if errors.IsDomain(err, "clone") {
        // Handle clone error
    }
}
