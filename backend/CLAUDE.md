# Go Backend Development Guide

Quick reference for RSS Summarizer Go backend development.

---

## Project Structure

```
backend/
├── cmd/
│   ├── api/main.go           # Entry point
│   └── migrate/main.go       # Migration runner
├── internal/
│   ├── api/
│   │   ├── handlers/         # HTTP handlers (Huma)
│   │   ├── middleware/       # Auth, logging
│   │   └── testing/          # Test helpers
│   ├── domain/               # Business models
│   │   ├── article/
│   │   ├── feed/
│   │   ├── topic/
│   │   ├── user/
│   │   └── user_article/
│   ├── repository/           # Data access (Postgres)
│   ├── service/              # Business services (LLM, RSS, scraper)
│   ├── workflow/             # Temporal workflows & activities
│   ├── config/               # Configuration
│   ├── crypto/               # Encryption (API keys)
│   └── database/             # DB connection
├── migrations/               # SQL migrations (up/down)
└── Makefile
```

---

## Quick Start

```bash
# Start dependencies (Postgres, Temporal)
make docker-up

# Run migrations
make migrate-up

# Run tests
make test

# Run server
make run

# Build binary
make build
```

---

## Testing

### Integration Tests

Tests run against a real Postgres database (`rss_summarizer_test`). Each test:
1. Connects to test DB
2. Runs migrations to set up schema
3. Truncates tables (clean slate)
4. Executes test
5. Cleans up

**Run tests:**
```bash
# All integration tests
make test

# With coverage
make test-coverage

# Watch mode (requires entr)
make test-watch

# Specific package
go test ./internal/api/handlers/... -v -run TestCreateFeed
```

**Test pattern:**
```go
func TestCreateFeed(t *testing.T) {
    // Use test helper to set up server + dependencies
    ts := apitest.NewTestServer(t)
    defer ts.Close(t)

    // Make authenticated HTTP request
    w := ts.Request(t, "POST", "/v1/feeds", map[string]interface{}{
        "url": "https://blog.golang.org/feed.atom",
        "poll_frequency_minutes": 60,
    })

    // Assert status
    apitest.AssertStatus(t, w, http.StatusOK)

    // Decode and verify response
    var resp FeedResponse
    apitest.DecodeResponse(t, w, &resp)
    if resp.ID == "" {
        t.Error("Expected feed ID to be set")
    }
}
```

**Test database:**
- URL: `postgres://rss_user:rss_pass@localhost:5432/rss_summarizer_test?sslmode=disable`
- Automatically created by `scripts/setup-test-db.sh`
- Migrations run before each test via `runMigrations()`
- Tables truncated between tests (not dropped)

**Test helpers** (`internal/api/testing/helpers.go`):
- `NewTestServer(t)` - Full server setup with all dependencies
- `ts.Request(t, method, path, body)` - Make authenticated HTTP request
- `AssertStatus(t, w, status)` - Assert HTTP status code
- `DecodeResponse(t, w, &result)` - Parse JSON response
- Test user automatically created with ID available in `ts.UserID`

---

## API Development

### Huma Framework

Huma auto-generates OpenAPI spec from request/response types.

**Handler pattern:**
```go
// 1. Define request/response types with validation tags
type CreateFeedRequest struct {
    Body struct {
        URL                  string `json:"url" pattern:"^https?://" doc:"RSS feed URL"`
        PollFrequencyMinutes int    `json:"poll_frequency_minutes" minimum:"15" maximum:"1440"`
    }
}

type CreateFeedResponse struct {
    Body FeedResponse
}

// 2. Register operation
huma.Register(api, huma.Operation{
    OperationID: "create-feed",
    Method:      http.MethodPost,
    Path:        "/v1/feeds",
    Summary:     "Create a new RSS feed",
    Tags:        []string{"Feeds"},
}, h.CreateFeed)

// 3. Implement handler
func (h *FeedHandlers) CreateFeed(ctx context.Context, input *CreateFeedRequest) (*CreateFeedResponse, error) {
    userID, ok := middleware.GetUserIDFromContext(ctx)
    if !ok {
        return nil, huma.Error401Unauthorized("User not authenticated")
    }

    // Business logic...

    return &CreateFeedResponse{Body: response}, nil
}
```

**OpenAPI spec available at:**
- Development: `http://localhost:8080/openapi.json`
- Used by frontend to auto-generate TypeScript client

---

## Repository Pattern

**Domain models** live in `internal/domain/*` (e.g., `article.Article`)

**Repository interfaces** defined alongside domain models:
```go
// internal/domain/article/repository.go (hypothetical)
type ArticleRepository interface {
    Create(ctx context.Context, article *Article) error
    FindByID(ctx context.Context, id uuid.UUID) (*Article, error)
}
```

**Implementation** in `internal/repository/*_repository.go`:
- Uses `sqlx` for SQL queries
- Returns domain errors (e.g., `NotFoundError`)
- Handles transactions when needed

**Transaction pattern:**
```go
func (r *pgRepository) CreateWithTopics(ctx context.Context, article *Article, topics []string) error {
    tx, err := r.db.BeginTxx(ctx, nil)
    if err != nil {
        return fmt.Errorf("begin tx: %w", err)
    }
    defer tx.Rollback() // Safe to call even after commit

    if err := r.createArticle(ctx, tx, article); err != nil {
        return err
    }
    if err := r.insertTopics(ctx, tx, article.ID, topics); err != nil {
        return err
    }

    return tx.Commit()
}
```

---

## Temporal Workflows

**Workflows** are deterministic orchestrators (use `workflow.Context`)

**Activities** are actual work units (use `context.Context`)

**Key rules:**
- Use `workflow.Now()`, not `time.Now()`
- Use `workflow.Sleep()`, not `time.Sleep()`
- All non-deterministic work goes in Activities
- Activities should be idempotent

**Example:**
```go
// Workflow
func ProcessFeedWorkflow(ctx workflow.Context, feedID uuid.UUID) error {
    // Execute activity with retry policy
    var articles []domain.Article
    err := workflow.ExecuteActivity(ctx, FetchFeedActivity, feedID).Get(ctx, &articles)
    if err != nil {
        return err
    }

    // Process each article
    for _, article := range articles {
        workflow.ExecuteChildWorkflow(ctx, SummarizeArticleWorkflow, article.ID)
    }
    return nil
}

// Activity
func FetchFeedActivity(ctx context.Context, feedID uuid.UUID) ([]domain.Article, error) {
    // Actual work: fetch RSS, parse, save to DB
    return articles, nil
}
```

**Register workflows/activities** in `internal/workflow/worker.go`:
```go
w.RegisterWorkflow(ProcessFeedWorkflow)
w.RegisterActivity(FetchFeedActivity)
```

---

## Database Migrations

**Location:** `migrations/NNN_description.{up,down}.sql`

**Run migrations:**
```bash
make migrate-up    # Apply pending migrations
make migrate-down  # Rollback last migration
```

**Migration structure:**
- `001_initial_schema.up.sql` - Creates tables
- `001_initial_schema.down.sql` - Drops tables
- Migrations run sequentially by number
- Use transactions (`BEGIN;` ... `COMMIT;`) when possible

**Naming:**
- Use descriptive names: `016_refactor_topics_to_shared.up.sql`
- Include both `.up.sql` and `.down.sql` for every migration

---

## Key Patterns

### Error Handling
```go
// Wrap with context
return fmt.Errorf("failed to fetch feed %s: %w", feedID, err)

// Domain errors
if errors.Is(err, sql.ErrNoRows) {
    return &NotFoundError{Resource: "article", ID: id}
}
```

### Context Usage
```go
// Always first parameter
func (s *Service) GetFeed(ctx context.Context, id uuid.UUID) (*Feed, error)

// Timeouts
ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
defer cancel()
```

### Validation
- Use Huma tags for request validation: `minimum:"15" maximum:"1440"`
- Return `huma.Error400BadRequest()` for validation errors
- Return `huma.Error401Unauthorized()` for auth errors
- Return `huma.Error404NotFound()` for missing resources

---

## Common Commands

```bash
# Development
make run                    # Start server
make test                   # Run tests
make migrate-up             # Apply migrations

# Docker
make docker-up              # Start Postgres + Temporal
make docker-down            # Stop containers

# Testing
make test                   # Integration tests
make test-coverage          # With coverage report
make test-watch             # Watch mode

# Build
make build                  # Build binary to bin/api
```

---

## Architecture Notes

### Per-User State
- **Articles** are global (shared across users)
- **User state** stored in `user_articles` table:
  - `is_read`, `is_saved`, `is_archived` are per-user
  - Multiple users can have different states for same article

### Topics
- **Topics** are global (e.g., "Go", "Security", "AI")
- **Preferences** stored in `user_topic_preferences` table
- Case-insensitive unique constraint prevents duplicates
- Topics auto-detected via LLM and normalized to broad categories

### Feeds
- Each feed belongs to one user (`user_id` foreign key)
- Articles from feeds are global
- Subscriptions tracked separately (future multi-tenancy)
