# Integration Testing

This project uses HTTP integration tests to test the API endpoints end-to-end.

## Setup

### 1. Start PostgreSQL

Make sure PostgreSQL is running:

```bash
# Using docker-compose
docker-compose up postgres -d

# Or if you have PostgreSQL installed locally, make sure it's running
```

### 2. Create Test Database

Run the setup script to create and migrate the test database:

```bash
cd backend
./scripts/setup-test-db.sh
```

This will:
- Drop the existing test database (if any)
- Create a fresh `rss_summarizer_test` database
- Run all migrations

## Running Tests

### Run All Integration Tests

```bash
go test ./internal/api/handlers/... -v
```

### Run Specific Test File

```bash
go test ./internal/api/handlers/feed_handlers_test.go -v
```

### Run Specific Test

```bash
go test ./internal/api/handlers/... -run TestCreateFeed -v
```

### Run with Coverage

```bash
go test ./internal/api/handlers/... -cover -coverprofile=coverage.out
go tool cover -html=coverage.out
```

## Test Structure

### Test Helpers (`internal/api/testing/helpers.go`)

Provides:
- `NewTestServer(t)` - Creates a test server with all dependencies
- `Request(method, path, body)` - Makes HTTP requests
- `AssertStatus(w, want)` - Asserts HTTP status code
- `DecodeResponse(w, target)` - Decodes JSON response
- `MockLLMService` - Mock LLM for testing (no real API calls)

### Test Files

- `feed_handlers_test.go` - Feed CRUD operations
- `preferences_handlers_test.go` - User preferences management
- `article_handlers_test.go` - Article listing and filtering

## Writing New Tests

Example test:

```go
func TestMyEndpoint(t *testing.T) {
    ts := apitest.NewTestServer(t)
    defer ts.Close(t)

    // Make request
    w := ts.Request(t, "POST", "/v1/my-endpoint", map[string]interface{}{
        "field": "value",
    })

    // Assert status
    apitest.AssertStatus(t, w, http.StatusOK)

    // Decode and verify response
    var resp struct {
        ID string `json:"id"`
    }
    apitest.DecodeResponse(t, w, &resp)

    if resp.ID == "" {
        t.Error("Expected ID to be set")
    }
}
```

## Test Database

- **Database:** `rss_summarizer_test`
- **User:** `rss_user`
- **Password:** `rss_pass`
- **Connection:** `postgres://rss_user:rss_pass@localhost:5432/rss_summarizer_test?sslmode=disable`

Each test runs in isolation - tables are truncated between tests.

## CI/CD

To run tests in CI:

```bash
# GitHub Actions example
- name: Setup test database
  run: |
    docker-compose up postgres -d
    sleep 5
    cd backend && ./scripts/setup-test-db.sh

- name: Run tests
  run: go test ./internal/api/handlers/... -v
```

## Troubleshooting

### "Failed to connect to test database"

Make sure PostgreSQL is running:
```bash
docker-compose up postgres -d
# Wait a few seconds for it to be ready
```

### "Table does not exist"

Run the setup script again:
```bash
./scripts/setup-test-db.sh
```

### Tests are slow

Integration tests connect to a real database, so they're slower than unit tests.
To speed up:
- Run specific test files instead of all tests
- Use a local PostgreSQL instead of Docker
- Run tests in parallel: `go test -parallel 4 ./internal/api/handlers/...`

## What We Don't Test

These tests focus on HTTP integration. We intentionally skip:
- Unit tests for individual functions
- Mocking every dependency
- Testing internal implementation details

The goal is to test the API contract and critical user flows.
