#!/bin/bash

# Example script demonstrating headless token setup using various methods

# Method 1: Using environment variables
setup_with_env() {
    echo "Setting up token using environment variables..."
    export GIT_PROVIDER=GITHUB
    export GIT_TOKEN_VALUE=$1
    export GIT_TOKEN_SCOPE="repo,workflow"
    export GIT_TOKEN_EXPIRY="90d"
    
    $(dirname $0)/../../bin/go-gittoken setup --non-interactive
}

# Method 2: Using command line arguments
setup_with_args() {
    echo "Setting up token using command line arguments..."
    $(dirname $0)/../../bin/go-gittoken setup \
        --non-interactive \
        --provider GITHUB \
        --token $1 \
        --scope "repo,workflow" \
        --expires "90d"
}

# Method 3: Using token file
setup_with_file() {
    echo "Setting up token using token file..."
    # Create temporary token file with secure permissions
    TOKEN_FILE=$(mktemp)
    echo "$1" > "$TOKEN_FILE"
    chmod 600 "$TOKEN_FILE"
    
    $(dirname $0)/../../bin/go-gittoken setup \
        --provider GITHUB \
        --scope "repo,workflow" \
        --token-file "$TOKEN_FILE"
    
    # Clean up token file
    rm "$TOKEN_FILE"
}

# Check if token is provided
if [ "$#" -ne 1 ]; then
    echo "Usage: $0 <github_token>"
    echo "Example: $0 ghp_your_token_here"
    exit 1
fi

TOKEN=$1

# Example using all three methods
echo "Demonstrating different methods for headless token setup..."

echo -e "\n1. Environment Variables Method"
setup_with_env "$TOKEN"

echo -e "\n2. Command Line Arguments Method"
setup_with_args "$TOKEN"

echo -e "\n3. Token File Method"
setup_with_file "$TOKEN"

echo -e "\nToken setup complete. You can now use other tools like go-gitclone."
