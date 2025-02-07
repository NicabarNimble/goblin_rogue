# First-Time Setup Guide

This guide walks you through setting up go-gittools for the first time, including obtaining and configuring your Git provider token.

## Getting Started

### 1. Obtaining a Provider Token

#### GitHub Token
Before using go-gittools with GitHub, you need to create a GitHub Personal Access Token:

1. Go to GitHub Settings > Developer settings > Personal access tokens > Tokens (classic)
   - Direct link: https://github.com/settings/tokens

2. Click "Generate new token (classic)"

3. Configure token settings:
   - Name: `go-gittools`
   - Expiration: Choose based on your needs (recommended: 90 days)
   - Required scopes:
     - `repo` (Full control of private repositories)
     - `workflow` (Required for GitHub Actions)

4. Click "Generate token" and **copy the token immediately**
   - Important: GitHub only shows the token once
   - Store it securely for the next step

#### GitLab Token
For GitLab integration, you need to create a GitLab Personal Access Token:

1. Go to GitLab Settings > Access Tokens
   - Direct link: https://gitlab.com/-/profile/personal_access_tokens

2. Configure token settings:
   - Name: `go-gittools`
   - Expiration: Choose based on your needs (recommended: 90 days)
   - Required scopes:
     - `api` (API access)
     - `read_repository` (Repository read access)
     - `write_repository` (Repository write access)

3. Click "Create personal access token" and **copy the token immediately**
   - Important: GitLab only shows the token once
   - Store it securely for the next step

### 2. Token Setup

The setup process has been simplified with automatic provider detection and validation:

```bash
go-gittoken setup
```

This will:
1. Automatically detect the provider from your token format
2. Validate the token format and required scopes
3. Verify the token with the provider's API
4. Check for missing permissions and expiration
5. Store the token securely

The tool now automatically:
- Detects GitHub tokens (starting with 'ghp_' or 'github_pat_')
- Detects GitLab tokens (starting with 'glpat-')
- Validates required scopes (GitHub: repo, workflow, admin:repo | GitLab: api)
- Checks token expiration and permissions via API

### 3. Headless/Automated Setup

For automated environments or CI/CD systems, there are three methods:

#### Method 1: Environment Variables (Recommended for CI/CD)

For GitHub:
```bash
export GIT_PROVIDER=GITHUB
export GIT_TOKEN_VALUE=your_token
export GIT_TOKEN_SCOPE=repo,workflow
export GIT_TOKEN_EXPIRY=90d
go-gittoken setup --non-interactive
```

For GitLab:
```bash
export GIT_PROVIDER=GITLAB
export GIT_TOKEN_VALUE=your_token
export GIT_TOKEN_SCOPE=api,read_repository,write_repository
export GIT_TOKEN_EXPIRY=90d
go-gittoken setup --non-interactive
```

#### Method 2: Command Line Arguments

For GitHub:
```bash
go-gittoken setup \
  --non-interactive \
  --provider GITHUB \
  --token your_token \
  --scope "repo,workflow" \
  --expires 90d
```

For GitLab:
```bash
go-gittoken setup \
  --non-interactive \
  --provider GITLAB \
  --token your_token \
  --scope "api,read_repository,write_repository" \
  --expires 90d
```

#### Method 3: Token File (Recommended for Initial Setup Scripts)

For GitHub:
```bash
# Create token file with secure permissions
echo "your_token" > token.txt
chmod 600 token.txt

# Set up token
go-gittoken setup \
  --provider GITHUB \
  --scope "repo,workflow" \
  --token-file token.txt

# Clean up
rm token.txt
```

For GitLab:
```bash
# Create token file with secure permissions
echo "your_token" > token.txt
chmod 600 token.txt

# Set up token
go-gittoken setup \
  --provider GITLAB \
  --scope "api,read_repository,write_repository" \
  --token-file token.txt

# Clean up
rm token.txt
```

## Best Practices for Token Setup

### Interactive Environments
1. Use interactive mode for first-time setup
2. Let the tool guide you through scope selection
3. Store the token securely before starting
4. Use expiration dates for better security

### Headless Environments
1. Create a setup script using one of the methods above
2. Use environment variables for CI/CD systems
3. For initial setup scripts:
   - Use token file method
   - Ensure secure file permissions
   - Clean up token files after setup
   - Log success/failure but not token values

### Security Considerations
1. Token Storage:
   - Never commit tokens to version control
   - Use environment variables or secure vaults
   - Clean up token files immediately after use

2. Token Permissions:
   - Use minimum required scopes
   - Set appropriate expiration dates
   - Rotate tokens regularly
   - One token per deployment/environment

3. Automation:
   - Use non-interactive mode
   - Validate token before use
   - Handle errors appropriately
   - Monitor token expiration

## Verifying Setup

After setting up your token, verify it works:

For GitHub:
```bash
# Clone a test repository (creates private-repo)
go-gitclone https://github.com/some-public-repo

# Or with a custom name
go-gitclone https://github.com/some-public-repo --name custom-test-repo

# Clean up test repository if needed (use the actual repository name)
rm -rf private-repo  # or custom-test-repo if using --name
```

For GitLab:
```bash
# Clone a test repository (creates private-repo)
go-gitclone https://gitlab.com/public-group/public-repo

# Or with a custom name
go-gitclone https://gitlab.com/public-group/public-repo --name custom-test-repo

# Clean up test repository if needed (use the actual repository name)
rm -rf private-repo  # or custom-test-repo if using --name
```

## Troubleshooting

Common issues and solutions:

1. **Token Not Found**
   - Check if token is properly set up: `go-gittoken setup`
   - Verify environment variables are set

2. **Permission Denied**
   - Verify token has required scopes
   - Check token hasn't expired
   - Ensure token is valid for the provider

3. **Invalid Token Format**
   - Ensure token wasn't truncated
   - Verify token starts with appropriate prefix:
     - GitHub: 'ghp_' or 'github_pat_'
     - GitLab: 'glpat-'

4. **File Permission Issues**
   - Check token file permissions: `chmod 600 token.txt`
   - Ensure only the owner can read the file

For more detailed information, see:
- [CLI Usage Guide](cli-usage.md)
- [Configuration Guide](configuration.md)
- [Headless Examples](../examples/headless/README.md)
