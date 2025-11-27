# PostgreSQL Migration

This document describes the migration from in-memory storage to PostgreSQL.

## Changes Made

### 1. Database Setup
- Added PostgreSQL service to `docker-compose.yml`
- PostgreSQL 16 Alpine image with persistent volume
- Default credentials: `goretro/goretro`
- Database name: `goretro`

### 2. Dependencies
- Added `github.com/lib/pq` PostgreSQL driver to `go.mod`

### 3. Database Schema
Created schema in `internal/models/schema.sql` with the following tables:
- `rooms` - Main room data (id, name, owner_id, phase, votes_per_user, created_at)
- `participants` - Room participants with roles and vote tracking
- `tickets` - Retrospective tickets with voting information
- `ticket_groups` - Grouped tickets for brainstorming phase
- `action_tickets` - Action items from discussion phase

### 4. Store Implementation
Rewrote `internal/models/store.go` to use PostgreSQL:
- `NewRoomStore(db *sql.DB)` - Now requires database connection
- `InitSchema()` - Initializes database tables
- `Create()`, `Get()`, `Delete()`, `Update()` - Now use SQL queries
- `List()`, `ListByOwner()`, `ListByParticipant()` - Query-based implementations

### 5. Application Updates
Updated `main.go`:
- Initialize database connection from `DATABASE_URL` environment variable
- Default: `postgres://goretro:goretro@localhost:5432/goretro?sslmode=disable`
- Initialize schema on startup
- Pass database connection to store

Updated `internal/handlers/handlers.go`:
- Handle errors from store operations
- Call `store.Update()` when modifying rooms

Updated `internal/websocket/hub.go`:
- Persist all room modifications to database
- Added error handling for failed updates

### 6. Tests
Updated `internal/models/store_test.go`:
- Tests now skip by default (require PostgreSQL instance)
- Can be enabled by removing the `t.Skip()` call
- Tests expect a running PostgreSQL instance at `goretro_test` database

## Running the Application

### With Docker Compose
```bash
make up
```

This will:
1. Start PostgreSQL container
2. Build and start the application
3. Start OAuth2 proxy and Dex

### Locally (for development)
```bash
# Start PostgreSQL (if not using docker-compose)
docker run -d \
  -e POSTGRES_DB=goretro \
  -e POSTGRES_USER=goretro \
  -e POSTGRES_PASSWORD=goretro \
  -p 5432:5432 \
  postgres:16-alpine

# Run the application
export DATABASE_URL="postgres://goretro:goretro@localhost:5432/goretro?sslmode=disable"
go run main.go
```

## Environment Variables

- `DATABASE_URL` - PostgreSQL connection string (required)
- `PORT` - Application port (default: 8080)

## Migration from Old Version

Since this changes from in-memory to persistent storage:
1. All existing room data will be lost on restart
2. The database will be empty on first run
3. Schema is automatically initialized on application startup

## Database Persistence

Data is now persisted in PostgreSQL:
- Room data survives application restarts
- All modifications are immediately saved to database
- Concurrent access is handled by PostgreSQL's transaction system

## Performance Considerations

- Each room modification triggers a full `UPDATE` operation
- This writes all room data (participants, tickets, groups, actions)
- For high-traffic scenarios, consider:
  - Implementing separate update methods for different entities
  - Using Redis for caching frequently accessed rooms
  - Optimizing the `Update()` method to only update changed data
