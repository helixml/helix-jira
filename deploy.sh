#!/bin/bash

set -euo pipefail

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

# Check if HELIX_URL is set
if [ -z "${HELIX_URL:-}" ]; then
    echo "Error: HELIX_URL is not set. Check your helix account page"
    exit 1
fi

# Check if HELIX_API_KEY is set
if [ -z "${HELIX_API_KEY:-}" ]; then
    echo "Error: HELIX_API_KEY is not set. Check your helix account page"
    exit 1
fi

# Create AUTH_STRING
AUTH_STRING=$(echo -n "${JIRA_API_EMAIL}:${JIRA_API_KEY}" | base64 -w 0)
export AUTH_STRING

# Create a temporary file for the processed helix.yaml
temp_file=$(mktemp)

# Delete existing secrets if they exist
helix secret delete --name JIRA_HOSTNAME || true
helix secret delete --name AUTH_STRING || true

# Create new secrets
helix secret create --name JIRA_HOSTNAME --value $JIRA_HOSTNAME
helix secret create --name AUTH_STRING --value $AUTH_STRING

# Run helix apply with the processed file
RESP=$(helix apply -f helix.yaml)
if [ $? -ne 0 ]; then
    echo "Error: Failed to deploy the app: $RESP"
    exit 1
fi

ID=$(echo "$RESP" | grep app_)

echo "Deployment completed to $HELIX_URL/new?app_id=$ID"

export HELIX_APP_ID=$ID
go run test_helix_app.go