<div align="center">
  <img src="static/icon.png" alt="GoRetro Logo" width="128" height="128">
  <h1>GoRetro</h1>
  <p>A retrospective tool for agile teams.</p>
</div>

## ⚠️ Authentication Requirement

**GoRetro requires an OAuth2 proxy to function.** The application does not handle authentication directly. It expects an OAuth2 proxy (such as [oauth2-proxy](https://oauth2-proxy.github.io/oauth2-proxy/)) to be configured in front of it, setting proper authentication headers (`X-Forwarded-User`, `X-Forwarded-Email`, etc.).

The included `docker-compose.yml` provides a complete setup with OAuth2 Proxy and Dex OIDC provider.

## Features

- **Multi-phase Retrospectives**: Ticketing, Merging, Voting, Discussion, and Summary phases
- **Real-time Collaboration**: WebSocket-based real-time updates
- **Participant Management**: Owner/Moderator roles with approval workflow
- **AI-Powered Auto-merge**: Automatically group similar tickets using AI (optional feature)
- **AI-Powered Action Proposals**: Generate actionable items from retrospective feedback (optional feature)

## Environment Variables

### Required
- `DATABASE_URL` - PostgreSQL connection string (default: `postgres://goretro:goretro@localhost:5432/goretro?sslmode=disable`)

### Authentication
**Note:** Authentication is handled by an OAuth2 proxy, not by environment variables. The application expects authentication headers from the proxy.

### Optional
- `REDIS_URL` - Redis server address for distributed synchronization (format: `host:port`)
- `CHAT_COMPLETION_ENDPOINT` - Chat completion API endpoint (e.g., OpenAI API compatible endpoint)
- `CHAT_COMPLETION_API_KEY` - API key for chat completion service
- `CHAT_COMPLETION_MODEL` - Model to use for chat completion (default: `gpt-4`)

## Auto-merge Feature

When `CHAT_COMPLETION_ENDPOINT` and `CHAT_COMPLETION_API_KEY` are configured, moderators will see an "Auto-merge" button during the DISCUSSION phase. This feature uses AI to:

1. Analyze all ticket content
2. Identify similar tickets based on semantic similarity
3. Automatically group related tickets together
4. Provide reasoning for each grouping

### Supported Chat Completion APIs

The auto-merge feature is compatible with OpenAI-compatible APIs:
- OpenAI API (https://api.openai.com/v1/chat/completions)
- Azure OpenAI Service
- Other OpenAI-compatible endpoints

### Example Configuration

```bash
export CHAT_COMPLETION_ENDPOINT="https://api.openai.com/v1/chat/completions"
export CHAT_COMPLETION_API_KEY="sk-your-api-key-here"
export CHAT_COMPLETION_MODEL="gpt-4"  # Optional, defaults to gpt-4
# Other model options: gpt-4-turbo, gpt-3.5-turbo, etc.
```

## Auto-propose Actions Feature

When the chat completion API is configured, moderators will see an "Auto-propose" button during the SUMMARY phase. This feature uses AI to:

1. Analyze all retrospective tickets and feedback
2. Consider team context if provided
3. Generate specific, actionable items that address the issues raised
4. Link each action to the relevant ticket
5. Mark AI-generated actions with a 🤖 robot icon prefix

The moderator can optionally provide team context (tech stack, constraints, etc.) to get more relevant action suggestions. The AI focuses on the most important issues, especially those with more votes or marked as discussed.

## Deployment

### Docker Compose (Recommended)

The easiest way to run GoRetro is with Docker Compose, which includes all required services including OAuth2 authentication:

```bash
# Clone the repository
git clone https://github.com/Armatorix/GoRetro.git
cd GoRetro

# Start all services (PostgreSQL, Redis, App, OAuth2 Proxy, Dex)
docker compose up -d

# Access at http://localhost (through OAuth2 proxy)
```

**Important:** The application runs on port 8080 internally but should only be accessed through the OAuth2 proxy on port 80. Never expose port 8080 directly to users.

For production, update the OAuth2 configuration in `docker-compose.yml` and `dex/config.yaml` with proper credentials, callback URLs, and OIDC providers.

## Usage

1. Start the application with the required environment variables
2. Create a new retrospective room
3. Share the room link with participants
4. Progress through the phases:
   - **Ticketing**: Add retrospective items
   - **Merging**: Group similar items (manual or auto-merge)
   - **Voting**: Vote on important items
   - **Discussion**: Discuss top items and create action items
   - **Summary**: Review all feedback

## TODO

* auto refresh WS
