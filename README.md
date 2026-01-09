# Real-Time Collaboration Platform

A full-stack real-time collaborative application supporting text editing and Kanban boards. Built with React, Go, Yjs CRDTs, and WebSockets.

## Features

- **Collaborative Text Editor**: Real-time document editing with TipTap
- **Collaborative Kanban Boards**: Drag-and-drop task management
- **User Authentication**: OAuth2 login (Google, GitHub)
- **Document Sharing**: Share documents with users or via public links
- **Offline Support**: IndexedDB persistence for offline editing
- **User Presence**: Live cursors and user awareness

## Tech Stack

**Frontend:** React 19, TypeScript, Vite, TipTap, Yjs, y-websocket, y-indexeddb
**Backend:** Go 1.25, Gin, Gorilla WebSocket, PostgreSQL 16
**Infrastructure:** Docker Compose

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
└─────────────┘          └──────────────┘          └─────────────┘
```

### Real-Time Collaboration Flow
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

### Data Flow
1. User edits → Yjs generates binary update
2. Update sent via WebSocket to backend
3. Backend broadcasts to all clients in room
4. Each client's Yjs applies and merges update
5. IndexedDB persists state locally
6. Periodic backup to PostgreSQL (optional)

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
