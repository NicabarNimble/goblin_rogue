// Package git provides core git operations for repository management.
//
// This package implements fundamental git operations such as cloning repositories,
// pushing changes, and managing remotes. It is designed to be used by higher-level
// packages that implement specific workflows like repository synchronization and
// publishing.
//
// Key Components:
//
// CloneOptions: Configuration struct for repository cloning operations.
// Contains settings for source and target URLs, working directory,
// authentication tokens, and progress tracking.
//
// CloneRepository: Main function for cloning git repositories.
// Handles the complete workflow of cloning from a source and
// configuring the target remote.
//
// Example Usage:
//
//	opts := CloneOptions{
//	    SourceURL:  "https://github.com/org/repo.git",
//	    TargetURL:  "https://github.com/fork/repo.git",
//	    WorkingDir: "/path/to/workspace",
//	    Token:      "github-token",
//	    Progress:   progressTracker,
//	}
//
//	if err := CloneRepository(opts); err != nil {
//	    log.Fatalf("Failed to clone repository: %v", err)
//	}
//
// Error Handling:
//
// All operations return detailed errors that can be handled by the caller.
// Errors are wrapped with context about the operation that failed.
// Progress tracking is integrated throughout operations to provide
// real-time feedback.
//
// Thread Safety:
//
// Git operations are not guaranteed to be thread-safe.
// Callers should ensure proper synchronization when operating
// on the same repository from multiple goroutines.
package git
