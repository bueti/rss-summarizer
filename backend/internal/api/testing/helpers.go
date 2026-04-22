package testing

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/bbu/rss-summarizer/backend/internal/api/handlers"
	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/config"
	"github.com/bbu/rss-summarizer/backend/internal/crypto"
	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/bbu/rss-summarizer/backend/internal/domain/newsletter_filter"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/bbu/rss-summarizer/backend/internal/service/llm"
	"github.com/bbu/rss-summarizer/backend/internal/service/rss"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// TestServer wraps the HTTP server and dependencies for testing
type TestServer struct {
	Router              *chi.Mux
	API                 huma.API
	DB                  *database.DB
	UserID              uuid.UUID
	Ctx                 context.Context
	FeedRepo            repository.FeedRepository
	EmailSourceRepo     email_source.Repository
	NewsletterFilterRepo newsletter_filter.Repository
	Config              *config.Config
}

// NewTestServer creates a new test server with all dependencies
func NewTestServer(t *testing.T) *TestServer {
	t.Helper()

	// Use test database URL from environment or default
	dbURL := "postgres://rss_user:rss_pass@localhost:5432/rss_summarizer_test?sslmode=disable"

	// Connect to test database
	db, err := database.New(dbURL)
	if err != nil {
		t.Fatalf("Failed to connect to test database: %v", err)
	}

	// Run migrations to set up schema
	if err := runMigrations(t, db); err != nil {
		t.Fatalf("Failed to run migrations: %v", err)
	}

	// Clean database before test (truncate tables, not drop them)
	cleanDatabase(t, db)

	// Initialize crypto service (must be exactly 32 bytes)
	cryptoService, err := crypto.NewService("12345678901234567890123456789012")
	if err != nil {
		t.Fatalf("Failed to initialize crypto service: %v", err)
	}

	// Initialize repositories
	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	prefsRepo := repository.NewPreferencesRepository(db, cryptoService)
	topicRepo := repository.NewTopicRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	userArticleRepo := repository.NewUserArticleRepository(db)
	emailSourceRepo := repository.NewEmailSourceRepository(db, cryptoService)
	newsletterFilterRepo := repository.NewNewsletterFilterRepository(db)

	// Initialize services (mock LLM service for tests)
	rssService := rss.NewService()

	// Create test user
	userID := uuid.New()
	_, err = db.DB.Exec(
		"INSERT INTO users (id, email, name) VALUES ($1, $2, $3) ON CONFLICT (id) DO NOTHING",
		userID, "test@example.com", "Test User",
	)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create default preferences for test user (LLM config is now global, not per-user)
	_, err = db.DB.Exec(`
		INSERT INTO user_preferences (id, user_id, default_poll_interval, max_articles_per_feed)
		VALUES ($1, $2, 60, 20)
		ON CONFLICT (user_id) DO NOTHING
	`, uuid.New(), userID)
	if err != nil {
		t.Fatalf("Failed to create default preferences: %v", err)
	}

	// Setup config
	cfg := &config.Config{
		Server: config.ServerConfig{
			Port: 8080,
			Env:  "test",
		},
		DevMode: config.DevModeConfig{
			Enabled: true,
			UserID:  userID.String(),
		},
	}

	// Setup logger
	logger := zerolog.New(zerolog.NewTestWriter(t)).With().Timestamp().Logger()
	log.Logger = logger

	// Setup HTTP server
	router := chi.NewRouter()
	router.Use(middleware.LoggingMiddleware(logger))
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Inject test user ID into context using the same method as auth middleware
			ctx := middleware.WithUserID(r.Context(), userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	})

	// Setup Huma API
	api := humachi.New(router, huma.DefaultConfig("RSS Summarizer API", "1.0.0"))

	// Register handlers
	healthHandlers := handlers.NewHealthHandlers(db)
	healthHandlers.Register(api)

	feedHandlers := handlers.NewFeedHandlers(feedRepo, subscriptionRepo, rssService)
	feedHandlers.Register(api)

	articleHandlers := handlers.NewArticleHandlers(articleRepo, userArticleRepo, nil) // nil temporal client for tests
	articleHandlers.Register(api)

	preferencesHandlers := handlers.NewPreferencesHandlers(prefsRepo)
	preferencesHandlers.Register(api)

	topicHandlers := handlers.NewTopicHandlers(topicRepo)
	topicHandlers.Register(api)

	// Register email source handlers
	emailSourceHandlers := handlers.NewEmailSourceHandlers(emailSourceRepo)
	emailSourceHandlers.Register(api)

	// Register newsletter filter handlers
	newsletterFilterHandlers := handlers.NewNewsletterFilterHandlers(newsletterFilterRepo, emailSourceRepo)
	newsletterFilterHandlers.Register(api)

	// Create test context
	ctx := context.Background()

	return &TestServer{
		Router:              router,
		API:                 api,
		DB:                  db,
		UserID:              userID,
		Ctx:                 ctx,
		FeedRepo:            feedRepo,
		EmailSourceRepo:     emailSourceRepo,
		NewsletterFilterRepo: newsletterFilterRepo,
		Config:              cfg,
	}
}

// Close cleans up the test server
func (ts *TestServer) Close(t *testing.T) {
	t.Helper()
	cleanDatabase(t, ts.DB)
	ts.DB.Close()
}

// Request makes an HTTP request to the test server
func (ts *TestServer) Request(t *testing.T, method, path string, body interface{}) *httptest.ResponseRecorder {
	t.Helper()

	var bodyReader io.Reader
	if body != nil {
		jsonBody, err := json.Marshal(body)
		if err != nil {
			t.Fatalf("Failed to marshal request body: %v", err)
		}
		bodyReader = bytes.NewReader(jsonBody)
	}

	req := httptest.NewRequest(method, path, bodyReader)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	ts.Router.ServeHTTP(w, req)

	return w
}

// DecodeResponse decodes the JSON response into the target
func DecodeResponse(t *testing.T, w *httptest.ResponseRecorder, target interface{}) {
	t.Helper()

	if err := json.NewDecoder(w.Body).Decode(target); err != nil {
		t.Fatalf("Failed to decode response: %v. Body: %s", err, w.Body.String())
	}
}

// AssertStatus checks that the response has the expected status code
func AssertStatus(t *testing.T, w *httptest.ResponseRecorder, want int) {
	t.Helper()

	if w.Code != want {
		t.Errorf("Expected status %d, got %d. Body: %s", want, w.Code, w.Body.String())
	}
}

// cleanDatabase truncates all tables for a clean test state
func runMigrations(t *testing.T, db *database.DB) error {
	t.Helper()

	// Create migrations table
	_, err := db.DB.Exec(`
		CREATE TABLE IF NOT EXISTS schema_migrations (
			version VARCHAR(255) PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL DEFAULT NOW()
		)
	`)
	if err != nil {
		return fmt.Errorf("failed to create migrations table: %w", err)
	}

	// Get migration files - try different paths depending on where tests are run from
	var files []string
	for _, pattern := range []string{"migrations/*.up.sql", "../../../migrations/*.up.sql", "../../migrations/*.up.sql"} {
		files, err = filepath.Glob(pattern)
		if err == nil && len(files) > 0 {
			break
		}
	}
	if len(files) == 0 {
		return fmt.Errorf("no migration files found")
	}

	sort.Strings(files)

	for _, upPath := range files {
		version := strings.TrimSuffix(filepath.Base(upPath), ".up.sql")

		// Check if already applied
		var applied bool
		err := db.DB.QueryRow("SELECT EXISTS(SELECT 1 FROM schema_migrations WHERE version = $1)", version).Scan(&applied)
		if err != nil {
			return fmt.Errorf("failed to check if migration %s is applied: %w", version, err)
		}
		if applied {
			continue
		}

		// Read and execute migration
		content, err := os.ReadFile(upPath)
		if err != nil {
			return fmt.Errorf("failed to read migration %s: %w", version, err)
		}

		if _, err := db.DB.Exec(string(content)); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", version, err)
		}

		// Mark as applied
		_, err = db.DB.Exec("INSERT INTO schema_migrations (version) VALUES ($1)", version)
		if err != nil {
			return fmt.Errorf("failed to mark migration %s as applied: %w", version, err)
		}

		t.Logf("Applied migration: %s", version)
	}

	return nil
}

func cleanDatabase(t *testing.T, db *database.DB) {
	t.Helper()

	tables := []string{
		"newsletter_filters",
		"email_sources",
		"user_articles",
		"user_topic_preferences",
		"user_feed_subscriptions",
		"articles",
		"topics",
		"feeds",
		"user_preferences",
		"sessions",
		"users",
	}

	for _, table := range tables {
		_, err := db.DB.Exec("TRUNCATE TABLE " + table + " CASCADE")
		if err != nil {
			t.Logf("Warning: Failed to truncate %s: %v", table, err)
		}
	}
}

// MockLLMService is a mock LLM service for testing
type MockLLMService struct{}

func (m *MockLLMService) SummarizeArticle(ctx context.Context, title, content string) (*llm.ArticleSummary, error) {
	return &llm.ArticleSummary{
		Summary:         "This is a test summary",
		KeyPoints:       []string{"Point 1", "Point 2", "Point 3"},
		ImportanceScore: 3,
		Topics:          []string{"Testing", "Go"},
	}, nil
}

func (m *MockLLMService) SummarizeArticleWithKey(ctx context.Context, title, content, apiKey string) (*llm.ArticleSummary, error) {
	return m.SummarizeArticle(ctx, title, content)
}

func (m *MockLLMService) SummarizeArticleWithConfig(ctx context.Context, title, content, provider, apiURL, apiKey, model string) (*llm.ArticleSummary, error) {
	return m.SummarizeArticle(ctx, title, content)
}

// CreateTestArticle creates a test article with default values
func CreateTestArticle(t *testing.T, ts *TestServer, feedID uuid.UUID) uuid.UUID {
	articleID := uuid.New()
	_, err := ts.DB.DB.Exec(`
		INSERT INTO articles (id, feed_id, title, url, published_at, summary, processing_status)
		VALUES ($1, $2, $3, $4, NOW(), $5, $6)
	`, articleID, feedID, "Test Article", "https://example.com/article-"+articleID.String(), "Test summary", "completed")
	if err != nil {
		t.Fatalf("Failed to create test article: %v", err)
	}
	return articleID
}

// SetArticleSaved sets the saved status for an article
func SetArticleSaved(t *testing.T, ts *TestServer, articleID uuid.UUID, isSaved bool) {
	_, err := ts.DB.DB.Exec(`
		INSERT INTO user_articles (id, user_id, article_id, is_saved)
		VALUES (uuid_generate_v4(), $1, $2, $3)
		ON CONFLICT (user_id, article_id)
		DO UPDATE SET is_saved = EXCLUDED.is_saved
	`, ts.UserID, articleID, isSaved)
	if err != nil {
		t.Fatalf("Failed to set article saved status: %v", err)
	}
}

// GetArticleUserState retrieves user-specific state for an article
func GetArticleUserState(t *testing.T, ts *TestServer, articleID uuid.UUID) (isRead, isSaved, isArchived bool) {
	err := ts.DB.DB.QueryRow(`
		SELECT COALESCE(is_read, false), COALESCE(is_saved, false), COALESCE(is_archived, false)
		FROM user_articles
		WHERE user_id = $1 AND article_id = $2
	`, ts.UserID, articleID).Scan(&isRead, &isSaved, &isArchived)
	if err == sql.ErrNoRows {
		return false, false, false
	}
	if err != nil {
		t.Fatalf("Failed to get article state: %v", err)
	}
	return
}
