#!/bin/bash

# Check if .env file exists
if [ ! -f .env ]; then
    echo "Error: .env file not found. Please create it and add your JIRA API key."
    exit 1
fi

# Check if helix.yaml file exists
if [ ! -f helix.yaml ]; then
    echo "Error: helix.yaml file not found."
    exit 1
fi

# Read .env file and export variables
export $(grep -v '^#' .env | xargs)

# Create AUTH_STRING
AUTH_STRING=$(echo -n "${JIRA_API_EMAIL}:${JIRA_API_KEY}" | base64)
export AUTH_STRING

# Create a temporary file for the processed helix.yaml
temp_file=$(mktemp)

# Process helix.yaml and substitute environment variables
envsubst < helix.yaml > "$temp_file"

# Run helix apply with the processed file
ID=$(helix apply -f "$temp_file" |grep app_)

# Remove the temporary file
rm "$temp_file"

echo "Deployment completed to $HELIX_URL/new?app_id=$ID"
