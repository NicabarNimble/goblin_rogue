# Headless Git Operations

This directory contains examples of using go-gittools in headless/automated environments.

## Token Setup

The `setup-token.sh` script demonstrates three different methods for setting up GitHub tokens in a non-interactive way:

1. Using Environment Variables:
```bash
export GIT_PROVIDER=GITHUB
export GIT_TOKEN_VALUE=your_token
export GIT_TOKEN_SCOPE=repo,workflow
./setup-token.sh your_github_token
```

2. Using Command Line Arguments:
```bash
./setup-token.sh your_github_token
```

3. Using Token File:
```bash
echo "your_github_token" > token.txt
chmod 600 token.txt
go-gittoken setup --provider GITHUB --scope "repo,workflow" --token-file token.txt
```

The script demonstrates secure token handling practices:
- Secure file permissions (600)
- Automatic cleanup of token files
- Token validation and scope verification
- Expiration management

## Repository Cloning

The `clone.sh` script demonstrates how to clone a public repository to a private one:

```bash
# Basic usage (creates private-{repo})
./clone.sh <source_repo_url>

# With custom repository name
./clone.sh <source_repo_url> --name custom-repo
```

Example:
```bash
# Creates private-ts-drp
./clone.sh https://github.com/topology-foundation/ts-drp.git

# Creates custom named repository
./clone.sh https://github.com/topology-foundation/ts-drp.git --name my-private-drp
```

The script demonstrates:
- Automatic target repository naming
- Custom repository name support via --name flag
- Secure token handling
- Clear error messaging

## LLM Integration

These tools are designed to work seamlessly with LLM-based automation:

1. Token Setup:
   - LLM can choose the most appropriate token setup method
   - Handles token validation and security automatically
   - Provides clear feedback on token status

2. Repository Operations:
   - LLM can execute clone operations after token setup
   - Handles the entire process without user interaction
   - Provides clear error messages and status updates

## Error Handling

The scripts include error handling for:
- Missing or invalid tokens
- Insufficient token scopes
- Insecure file permissions
- Invalid repository URLs
- Missing command line arguments

## Requirements

- GitHub personal access token with required scopes:
  - `repo` for private repository operations
  - `workflow` for GitHub Actions integration
- go-gittools binaries (built using `just build` in the root directory)

## Security Notes

1. Token Storage:
   - Tokens are stored securely in environment variables
   - File-based tokens are cleaned up after use
   - Permissions are verified before operations

2. Best Practices:
   - Use non-interactive mode for automation
   - Implement token rotation
   - Monitor token expiration
   - Use minimal required scopes
