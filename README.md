# Real-Time Collaboration Platform

A full-stack real-time collaborative application supporting both text editing and Kanban boards, built with React and Go. The system uses Yjs CRDTs for conflict-free synchronization and WebSockets for real-time updates.

## Features

- **Collaborative Text Editor**: Real-time document editing with TipTap editor
- **Collaborative Kanban Boards**: Drag-and-drop task management
- **Real-Time Synchronization**: Instant updates across all connected clients
- **Offline Support**: IndexedDB persistence for offline editing
- **User Presence**: See who's currently editing with live cursors and awareness

## Tech Stack

### Frontend
- React 19 + TypeScript
- Vite (build tool)
- TipTap (collaborative rich text editor)
- Yjs (CRDT for conflict-free replication)
- y-websocket (WebSocket sync provider)
- y-indexeddb (offline persistence)

### Backend
- Go 1.25
- Gin (web framework)
- Gorilla WebSocket
- PostgreSQL 16 (document storage)
- Redis 7 (future use)

### Infrastructure
- Docker Compose
- PostgreSQL
- Redis

## Getting Started

### Prerequisites

- Node.js 18+ and npm
- Go 1.25+
- Docker and Docker Compose

### Installation

1. **Clone the repository**
```bash
git clone <your-repo-url>
cd realtime-collab
```

2. **Set up environment variables**

Create backend environment file:
```bash
cp backend/.env.example backend/.env
# Edit backend/.env with your configuration
```

Create frontend environment file:
```bash
cp frontend/.env.example frontend/.env
# Edit frontend/.env with your configuration (optional for local development)
```

Create root environment file for Docker:
```bash
cp .env.example .env
# Edit .env with your Docker database credentials
```

3. **Start the infrastructure**
```bash
docker-compose up -d
```

This will start PostgreSQL and Redis containers.

4. **Install and run the backend**
```bash
cd backend
go mod download
go run cmd/server/main.go
```

The backend server will start on `http://localhost:8080`

5. **Install and run the frontend**
```bash
cd frontend
npm install
npm run dev
```

The frontend will start on `http://localhost:5173`

6. **Access the application**

Open your browser and navigate to `http://localhost:5173`

## Environment Variables

### Backend (backend/.env)

```env
PORT=8080
DATABASE_URL=postgres://user:password@localhost:5432/collaboration?sslmode=disable
REDIS_URL=redis://localhost:6379
ALLOWED_ORIGINS=http://localhost:5173,http://localhost:3000
```

### Frontend (frontend/.env)

```env
VITE_API_URL=http://localhost:8080/api
VITE_WS_URL=ws://localhost:8080/ws
```

### Docker Compose (.env)

```env
POSTGRES_USER=collab
POSTGRES_PASSWORD=your_secure_password_here
POSTGRES_DB=collaboration
```

## Architecture

### Real-Time Collaboration Flow

1. Client connects to WebSocket endpoint `/ws/:documentId`
2. WebSocket handler creates a client and registers it with the Hub
3. Hub manages rooms (one room per document)
4. Yjs generates binary updates on document changes
5. Client sends updates to WebSocket
6. Hub broadcasts updates to all clients in the same room
7. Yjs applies updates and merges changes automatically (CRDT)
8. IndexedDB persists state locally for offline support

### API Endpoints

#### REST API
- `GET /api/documents` - List all documents
- `POST /api/documents` - Create a new document
- `GET /api/documents/:id` - Get document metadata
- `GET /api/documents/:id/state` - Get document Yjs state
- `PUT /api/documents/:id/state` - Update document Yjs state
- `DELETE /api/documents/:id` - Delete a document

#### WebSocket
- `GET /ws/:roomId` - WebSocket connection for real-time sync

#### Health Check
- `GET /health` - Server health status

## Project Structure

```
realtime-collab/
├── backend/
│   ├── cmd/server/          # Application entry point
│   ├── internal/
│   │   ├── handlers/        # HTTP/WebSocket handlers
│   │   ├── hub/             # WebSocket hub (room management)
│   │   ├── models/          # Domain models
│   │   └── store/           # Database layer
│   └── scripts/             # Database initialization scripts
├── frontend/
│   ├── src/
│   │   ├── api/             # REST API client
│   │   ├── components/      # React components
│   │   ├── lib/             # Yjs integration
│   │   ├── pages/           # Page components
│   │   └── hooks/           # Custom React hooks
│   └── public/              # Static assets
└── docker-compose.yml       # Infrastructure setup
```

## Development

### Backend Commands
```bash
cd backend
go run cmd/server/main.go  # Run development server
go build -o server cmd/server/main.go  # Build binary
go fmt ./...               # Format code
```

### Frontend Commands
```bash
cd frontend
npm run dev      # Start dev server
npm run build    # Production build
npm run preview  # Preview production build
npm run lint     # Run ESLint
```

### Database

The database schema is automatically initialized on first run using `backend/scripts/init.sql`.

To reset the database:
```bash
docker-compose down -v
docker-compose up -d
```

## How It Works

### Conflict-Free Replication (CRDT)

The application uses Yjs, a CRDT implementation, which allows:
- Multiple users to edit simultaneously without conflicts
- Automatic merging of concurrent changes
- Offline editing with eventual consistency
- No need for operational transformation or locking

### WebSocket Broadcasting

The backend acts as a message broker:
1. Receives binary Yjs updates from clients
2. Broadcasts updates to all clients in the same room
3. Does not interpret or modify the updates
4. Yjs handles all conflict resolution on the client side

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License.

## Acknowledgments

- [Yjs](https://github.com/yjs/yjs) - CRDT framework
- [TipTap](https://tiptap.dev/) - Rich text editor
- [Gin](https://gin-gonic.com/) - Go web framework
