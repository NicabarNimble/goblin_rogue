#!/bin/bash

# Parse arguments
SOURCE_REPO=""
CUSTOM_NAME=""

while [[ $# -gt 0 ]]; do
    case $1 in
        --name)
            CUSTOM_NAME="$2"
            shift 2
            ;;
        *)
            if [ -z "$SOURCE_REPO" ]; then
                SOURCE_REPO="$1"
            else
                echo "Error: Unexpected argument: $1"
                echo "Usage: $0 <source_repo_url> [--name custom_name]"
                echo "Example: $0 https://github.com/topology-foundation/ts-drp.git"
                echo "Example with custom name: $0 https://github.com/topology-foundation/ts-drp.git --name my-private-drp"
                exit 1
            fi
            shift
            ;;
    esac
done

# Check if source URL is provided
if [ -z "$SOURCE_REPO" ]; then
    echo "Error: Source repository URL is required"
    echo "Usage: $0 <source_repo_url> [--name custom_name]"
    echo "Example: $0 https://github.com/topology-foundation/ts-drp.git"
    echo "Example with custom name: $0 https://github.com/topology-foundation/ts-drp.git --name my-private-drp"
    exit 1
fi

# Check if GIT_TOKEN_GITHUB is set
if [ -z "$GIT_TOKEN_GITHUB" ]; then
    echo "Error: GIT_TOKEN_GITHUB environment variable is not set"
    echo "Please set it first: export GIT_TOKEN_GITHUB=your_token"
    exit 1
fi

# Build the command
CMD="$(dirname $0)/../../bin/go-gitclone \"$SOURCE_REPO\" --token \"$GIT_TOKEN_GITHUB\""
if [ ! -z "$CUSTOM_NAME" ]; then
    CMD="$CMD --name \"$CUSTOM_NAME\""
fi

# Execute go-gitclone with the provided parameters
eval $CMD
