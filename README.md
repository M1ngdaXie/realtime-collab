# Real-Time Collaboration Platform

A full-stack real-time collaborative application supporting text editing and Kanban boards. Built with React, Go, Yjs CRDTs, and WebSockets.

## Features

- **Collaborative Text Editor**: Real-time document editing with TipTap
- **Collaborative Kanban Boards**: Drag-and-drop task management
- **Multi-Server Synchronization**: Redis pub/sub enables horizontal scaling across multiple backend instances
- **User Authentication**: OAuth2 login (Google, GitHub)
- **Document Sharing**: Share documents with users or via public links
- **Offline Support**: IndexedDB persistence for offline editing
- **User Presence**: Live cursors and user awareness with cross-server sync
- **High Availability**: Automatic fallback mode when Redis is unavailable

## Tech Stack

**Frontend:** React 19, TypeScript, Vite, TipTap, Yjs, y-websocket, y-indexeddb
**Backend:** Go 1.25, Gin, Gorilla WebSocket, PostgreSQL 16, Redis 7 (pub/sub)
**Infrastructure:** Docker Compose
**Logging:** Zap (structured logging)

## Quick Start

### Prerequisites
- Node.js 18+, npm
- Go 1.25+
- Docker & Docker Compose

### Setup

1. **Clone and configure environment**
```bash
git clone <repo-url>
cd realtime-collab

# Setup environment files
cp .env.example .env
cp backend/.env.example backend/.env

# Edit .env files with your configuration
# Minimum required: DATABASE_URL, JWT_SECRET
```

2. **Start infrastructure**
```bash
docker-compose up -d
```

3. **Run backend**
```bash
cd backend
go mod download
go run cmd/server/main.go
# Server runs on http://localhost:8080
```

4. **Run frontend**
```bash
cd frontend
npm install
npm run dev
# App runs on http://localhost:5173
```

## Architecture

### System Overview
```
┌─────────────┐          ┌──────────────┐          ┌─────────────┐
│   Browser   │◄────────►│  Go Backend  │◄────────►│ PostgreSQL  │
│             │          │              │          │             │
│ React + Yjs │  WS+HTTP │  Gin + Hub   │   SQL    │  Documents  │
│ TipTap      │          │  WebSocket   │          │  Users      │
│ IndexedDB   │          │  REST API    │          │  Sessions   │
└─────────────┘          └──────┬───────┘          └─────────────┘
                                │
                                │ Pub/Sub
                                ▼
                         ┌─────────────┐
                         │    Redis    │
                         │             │
                         │  Message    │
                         │  Bus for    │
                         │  Multi-     │
                         │  Server     │
                         └─────────────┘
```

### Multi-Server Architecture (Horizontal Scaling)
```
┌──────────┐     ┌──────────┐     ┌──────────┐
│ Client A │     │ Client B │     │ Client C │
└────┬─────┘     └────┬─────┘     └────┬─────┘
     │                │                │
     │ WS             │ WS             │ WS
     ▼                ▼                ▼
┌─────────────┐  ┌─────────────┐  ┌─────────────┐
│  Server 1   │  │  Server 2   │  │  Server 3   │
│  :8080      │  │  :8081      │  │  :8082      │
└──────┬──────┘  └──────┬──────┘  └──────┬──────┘
       │                │                │
       └────────────────┼────────────────┘
                        │
                   Redis Pub/Sub
                  (room:doc-123)
       ┌────────────────┼────────────────┐
       │                │                │
       ▼                ▼                ▼
   Publish          Subscribe       Awareness
   Updates          Updates         Caching
```

### Real-Time Collaboration Flow (Single Server)
```
Client A                    Backend Hub                 Client B
   │                            │                          │
   │──1. Edit document──────────►│                          │
   │   (Yjs binary update)      │                          │
   │                            │──2. Broadcast update────►│
   │                            │                          │
   │                            │◄─3. Edit document────────│
   │◄─4. Broadcast update───────│                          │
   │                            │                          │
   ▼                            ▼                          ▼
[Apply CRDT merge]         [Relay only]            [Apply CRDT merge]
```

### Cross-Server Collaboration Flow (with Redis)
```
Client A          Server 1              Redis           Server 2          Client B
   │                 │                    │                 │                 │
   │──1. Edit───────►│                    │                 │                 │
   │                 │──2. Broadcast      │                 │                 │
   │                 │   to local─────────┼────────────────►│──3. Forward────►│
   │                 │                    │                 │   to local      │
   │                 │──4. Publish to─────►                 │                 │
   │                 │   Redis            │                 │                 │
   │                 │   (room:doc-123)   │──5. Distribute─►│                 │
   │                 │                    │   to subscribers│                 │
   │                 │                    │                 │──6. Broadcast──►│
   │                 │                    │                 │                 │
   ▼                 ▼                    ▼                 ▼                 ▼
```

### Authentication Flow
```
User                OAuth Provider          Backend              Database
 │                        │                    │                    │
 │──1. Click login───────►│                    │                    │
 │◄─2. Auth page──────────│                    │                    │
 │──3. Approve───────────►│                    │                    │
 │                        │──4. Callback──────►│                    │
 │                        │                    │──5. Create user───►│
 │                        │                    │◄──────────────────│
 │◄─6. JWT token + cookie─┤◄──────────────────│                    │
```

## Project Structure

```
realtime-collab/
├── backend/
│   ├── cmd/server/          # Entry point
│   ├── internal/
│   │   ├── auth/            # JWT & OAuth middleware
│   │   ├── handlers/        # HTTP/WebSocket handlers
│   │   ├── hub/             # WebSocket hub & rooms
│   │   ├── logger/          # Structured logging (Zap)
│   │   ├── messagebus/      # Redis pub/sub abstraction
│   │   ├── models/          # Domain models
│   │   └── store/           # PostgreSQL data layer
│   └── scripts/init.sql     # Database schema
├── frontend/
│   ├── src/
│   │   ├── api/             # REST API client
│   │   ├── components/      # React components
│   │   ├── hooks/           # Custom hooks
│   │   ├── lib/yjs.ts       # Yjs setup
│   │   └── pages/           # Page components
└── docker-compose.yml       # PostgreSQL & Redis
```

## Key Endpoints

### REST API
- `POST /api/auth/google` - Google OAuth login
- `POST /api/auth/github` - GitHub OAuth login
- `GET /api/auth/me` - Get current user
- `POST /api/auth/logout` - Logout
- `GET /api/documents` - List all documents
- `POST /api/documents` - Create document
- `GET /api/documents/:id` - Get document
- `PUT /api/documents/:id/state` - Update Yjs state
- `DELETE /api/documents/:id` - Delete document
- `POST /api/documents/:id/shares` - Share with user
- `POST /api/documents/:id/share-link` - Create public link

### WebSocket
- `GET /ws/:roomId` - Real-time sync endpoint

## Environment Variables

### Backend (.env)
```bash
PORT=8080
DATABASE_URL=postgres://user:pass@localhost:5432/collaboration?sslmode=disable
JWT_SECRET=your-secret-key-here
FRONTEND_URL=http://localhost:5173
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000

# Redis Configuration (required for multi-server setup)
REDIS_URL=redis://localhost:6379/0

# Server Instance ID (auto-generated if not set)
# Only needed when running multiple instances
SERVER_ID=

# Environment (development, production)
ENVIRONMENT=development

# OAuth (optional)
GOOGLE_CLIENT_ID=your-google-client-id
GOOGLE_CLIENT_SECRET=your-google-client-secret
GITHUB_CLIENT_ID=your-github-client-id
GITHUB_CLIENT_SECRET=your-github-client-secret
```

### Docker (.env in root)
```bash
POSTGRES_USER=collab
POSTGRES_PASSWORD=your-secure-password
POSTGRES_DB=collaboration
```

## Development Commands

### Backend
```bash
cd backend
go run cmd/server/main.go              # Run server
go test ./...                          # Run tests
go test -v ./internal/handlers         # Run handler tests
go build -o server cmd/server/main.go  # Build binary
```

### Frontend
```bash
cd frontend
npm run dev      # Dev server
npm run build    # Production build
npm run lint     # Lint code
```

### Database
```bash
docker-compose down -v   # Reset database
docker-compose up -d     # Start fresh
```

### Multi-Server Testing
```bash
# Terminal 1: Start first server
cd backend
PORT=8080 SERVER_ID=server-1 REDIS_URL=redis://localhost:6379/0 go run cmd/server/main.go

# Terminal 2: Start second server
cd backend
PORT=8081 SERVER_ID=server-2 REDIS_URL=redis://localhost:6379/0 go run cmd/server/main.go

# Terminal 3: Monitor Redis pub/sub activity
redis-cli monitor | grep -E "PUBLISH|HSET"
```

### Redis Commands
```bash
# Monitor real-time Redis activity
redis-cli monitor

# Check active subscriptions
redis-cli PUBSUB CHANNELS

# View awareness cache for a room
redis-cli HGETALL "room:doc-{id}:awareness"

# Clear Redis (for testing)
redis-cli FLUSHDB
```

## How It Works

### CRDT-Based Collaboration
- **Yjs** provides Conflict-free Replicated Data Types (CRDTs)
- Multiple users edit simultaneously without conflicts
- Changes merge automatically, no manual conflict resolution
- Works offline, syncs when reconnected

### Backend as Message Broker
- Backend doesn't understand Yjs data structure
- Simply broadcasts binary updates to all clients in a room
- All conflict resolution happens client-side via Yjs
- Rooms are isolated by document ID

### Redis Pub/Sub for Multi-Server Sync
- **MessageBus Interface**: Abstraction layer for pub/sub operations
- **Redis Implementation**: Broadcasts Yjs updates across server instances
- **Awareness Caching**: User presence synced via Redis HSET/HGET
- **Health Monitoring**: Automatic fallback to local-only mode if Redis fails
- **Binary Safety**: Preserves Yjs binary data through Redis string encoding
- **Room Isolation**: Each document gets its own Redis channel (`room:doc-{id}:messages`)

### Data Flow (Multi-Server)
1. User edits → Yjs generates binary update
2. Update sent via WebSocket to backend server (e.g., Server 1)
3. **Server 1** broadcasts to:
   - All local clients in the room (via WebSocket)
   - Redis pub/sub channel (for cross-server sync)
4. **Server 2** receives update from Redis subscription
5. **Server 2** forwards to its local clients in the same room
6. Each client's Yjs applies and merges update (CRDT magic)
7. IndexedDB persists state locally
8. Periodic backup to PostgreSQL (optional)

### Fallback Mode
- When Redis is unavailable, servers enter **fallback mode**
- Continues to work for single-server deployments
- Cross-server sync disabled until Redis recovers
- Automatic recovery when Redis comes back online
- Health checks run every 10 seconds

## Testing

The backend uses `testify/suite` for organized testing:
```bash
go test ./internal/handlers                              # All handler tests
go test -v ./internal/handlers -run TestDocumentHandler  # Specific suite
```

## Database Schema

- **documents**: Document metadata and Yjs state (BYTEA)
- **users**: OAuth user profiles
- **sessions**: JWT tokens with expiration
- **shares**: User-to-user document sharing
- **share_links**: Public shareable links

## License

MIT License

## Acknowledgments

- [Yjs](https://github.com/yjs/yjs) - CRDT framework
- [TipTap](https://tiptap.dev/) - Collaborative editor
- [Gin](https://gin-gonic.com/) - Go web framework
