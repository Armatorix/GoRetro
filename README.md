# GoRetro

A retrospective tool for agile teams.

## Features

- **Multi-phase Retrospectives**: Ticketing, Merging, Voting, Discussion, and Summary phases
- **Real-time Collaboration**: WebSocket-based real-time updates
- **Participant Management**: Owner/Moderator roles with approval workflow
- **AI-Powered Auto-merge**: Automatically group similar tickets using AI (optional feature)

## Environment Variables

### Required
- `DATABASE_URL` - PostgreSQL connection string (default: `postgres://goretro:goretro@localhost:5432/goretro?sslmode=disable`)

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
