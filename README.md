# helix-jira

This project helps you deploy the Helix app for Jira so that you can talk to your JIRA instance with natural language, and ask questions like:

1. Show me the tickets assigned to me, Adam Fox
2. Display the tickets assigned to Bob Gelder
3. What's the current sprint?
3. Summarize ticket number PROJ-123
4. Generate code for the frontend task in ticket number PROJ-927
5. Which tickets have exceeded their deadline?

Future (with GitHub integration):
6. How many pull requests are there in the development branch?

## Setup

1. Go to [https://app.tryhelix.ai/account](https://app.tryhelix.ai/account) or the corresponding domain for your private Helix deployment and run the "CLI install & login" section, e.g:
   ```
   export HELIX_URL=...
   export HELIX_API_KEY=
   ```

2. Copy the example environment file:
   ```
   cp .env.example .env
   ```

3. Edit `.env` and add your Jira information:
   ```
   JIRA_HOSTNAME=https://yourco.atlassian.net
   JIRA_API_EMAIL=you@domain.com
   JIRA_API_KEY=your_jira_api_key_here
   ```
   Replace the values with your actual Jira instance details and API key.


## Deployment

To deploy the Helix JIRA app:

```
bash deploy.sh
```

This script will process your configuration and apply it using Helix.

## Requirements

- `envsubst` command (part of GNU gettext utilities)
