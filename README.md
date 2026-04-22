# RSS Summarizer

AI-powered RSS feed aggregator that summarizes articles using Claude, rates them by importance, and helps you stay informed without the noise.

## Features

- **AI Summarization**: Auto-summarize articles with Anthropic Claude
- **Importance Scoring**: Articles rated 1-5 based on your interests
- **Smart Organization**: Save for later, archive, mark as read
- **Topic Detection**: Auto-categorize with normalized broad topics
- **Intelligent Filtering**: By importance, topic, feed, read/saved/archived status
- **Background Processing**: Temporal workflows poll feeds and process articles automatically

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ and Node.js 18+ (for local development)
- Anthropic API key ([get one here](https://console.anthropic.com/))

### Setup

```bash
# 1. Clone repo
git clone <repo-url>
cd rss-summarizer

# 2. Configure backend
cp backend/.env.example backend/.env
# Edit backend/.env and add your Anthropic API key:
# LLM_API_KEY=sk-ant-api03-your-key-here

# 3. Start everything
make start
```

**That's it!** Open http://localhost:5173

The `make start` command:
- Starts PostgreSQL + Temporal (Docker)
- Runs database migrations
- Starts backend API (port 8080)
- Starts frontend dev server (port 5173)

## Development

```bash
# Daily workflow
make start          # Start everything
make stop           # Stop all services

# Run components separately
make dev-backend    # Backend only (requires Docker running)
make dev-frontend   # Frontend only (requires backend)

# Testing
make test           # Run all tests

# Database
make migrate-up     # Apply migrations
make migrate-down   # Rollback last migration

# Other
make help           # Show all commands
```

## Tech Stack

**Backend:** Go, PostgreSQL, Temporal, Huma v2 (OpenAPI auto-generation)
**Frontend:** SvelteKit, Svelte 5, TypeScript, Tailwind CSS
**AI:** Anthropic Claude API

## Project Structure

```
rss-summarizer/
├── backend/
│   ├── cmd/api/           # Entry point
│   ├── internal/
│   │   ├── api/           # HTTP handlers (Huma)
│   │   ├── domain/        # Business models
│   │   ├── repository/    # Data access (Postgres)
│   │   ├── service/       # LLM, RSS, scraper
│   │   └── workflow/      # Temporal workflows
│   ├── migrations/        # SQL migrations
│   └── CLAUDE.md          # Backend dev guide
├── frontend/
│   ├── src/
│   │   ├── lib/           # Components, stores, API client
│   │   └── routes/        # SvelteKit pages
│   └── CLAUDE.md          # Frontend dev guide
├── prd.md                 # Product requirements
└── Makefile               # Unified commands
```

## Key Concepts

### Articles & User State

- **Articles** are global (shared across users)
- **User state** is per-user:
  - `is_read` - Track what you've read
  - `is_saved` - Save for later reading
  - `is_archived` - Archive to remove from main view

### Topics

- Auto-detected by LLM and normalized to broad categories (e.g., "Go", "Security", "AI")
- Set preferences per topic: "High Interest", "Normal", or "Hide"
- Topics are global, preferences are per-user

### API Client

- Frontend TypeScript client **auto-generated** from backend OpenAPI spec
- Backend changes → Regenerate client: `npm run generate:api` (from frontend dir)
- See `frontend/CLAUDE.md` for details

## Configuration

### Required Environment Variables

**Backend** (`backend/.env`):
```bash
DATABASE_URL=postgres://rss_user:rss_pass@localhost:5432/rss_summarizer?sslmode=disable
LLM_API_KEY=sk-ant-api03-your-key-here  # Get from console.anthropic.com
LLM_MODEL=claude-haiku-4-5                # Or claude-sonnet-4-5 for better quality
```

**Frontend** (`frontend/.env` - optional):
```bash
VITE_API_URL=http://localhost:8080  # Defaults to localhost:8080
```

### Development Mode

Set `DEV_MODE=true` in backend `.env` to bypass authentication (default user created automatically).

## Documentation

- **[prd.md](prd.md)** - Full product requirements and features
- **[backend/CLAUDE.md](backend/CLAUDE.md)** - Backend development guide (testing, API, workflows)
- **[frontend/CLAUDE.md](frontend/CLAUDE.md)** - Frontend development guide (API generation, stores, components)

## API Documentation

Once running, visit:
- **OpenAPI spec**: http://localhost:8080/openapi.json
- **Temporal UI**: http://localhost:8233

## Troubleshooting

**Frontend build fails with "Cannot fetch OpenAPI spec":**
- Ensure backend is running: `make dev-backend`
- Manually generate API client: `cd frontend && npm run generate:api`

**Database connection errors:**
- Check Docker is running: `docker ps`
- Restart services: `make docker-down && make docker-up`

**Temporal workflows not running:**
- Check Temporal UI: http://localhost:8233
- Ensure worker is running (part of backend)

## License

MIT
