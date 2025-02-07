# Get version from git
version := `git describe --tags --always --dirty`

# Default recipe to show available commands
default:
    @just --list

# Build all tools
build:
    mkdir -p bin
    go build -o bin/go-gittoken ./cmd/gittoken
    go build -o bin/go-gitclone ./cmd/gitclone
    go build -o bin/go-gitsync ./cmd/gitsync
    go build -o bin/go-gitpublish ./cmd/gitpublish

# Run quick tests (unit tests only, no integration or extended tests)
test-quick:
    go test -short -v ./... -run "^Test[^Extended|^Integration]"

# Run all unit and integration tests with race detection
test:
    go test -v -race -timeout 10m ./...

# Run extended tests (including long-running tests)
test-extended:
    go test -v -timeout 20m ./... -run "Extended"

# Run integration tests only
test-integration:
    go test -v ./tests -run "Integration"

# Run example tests only
test-examples:
    go test -v ./examples/...

# Run all tests with coverage and generate HTML report
test-coverage:
    #!/bin/sh
    mkdir -p coverage
    go test -v -race -coverprofile=coverage/coverage.out -covermode=atomic ./...
    go tool cover -html=coverage/coverage.out -o coverage/coverage.html
    echo "Coverage report generated at coverage/coverage.html"

# Run tests in watch mode (requires watchexec)
test-watch:
    watchexec -e go -r -- go test -v ./...

# Clean build artifacts and coverage reports
clean:
    rm -rf bin coverage

# Run all test suites (unit, integration, extended, and examples)
test-all: test test-integration test-extended test-examples
    @echo "All test suites completed"

# Run all main_test.go files in cmd directory
test-main:
    #!/bin/sh
    for dir in cmd/*; do
        if [ -f "$dir/main_test.go" ]; then
            echo "Running tests in $dir"
            go test -v ./$dir || exit 1
        fi
    done
