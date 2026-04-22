package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"slices"
	"syscall"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/api/handlers"
	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/config"
	"github.com/bbu/rss-summarizer/backend/internal/crypto"
	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/bbu/rss-summarizer/backend/internal/service/auth"
	"github.com/bbu/rss-summarizer/backend/internal/service/gmail"
	"github.com/bbu/rss-summarizer/backend/internal/service/llm"
	"github.com/bbu/rss-summarizer/backend/internal/service/rss"
	"github.com/bbu/rss-summarizer/backend/internal/workflow"
	"github.com/danielgtaylor/huma/v2"
	"github.com/danielgtaylor/huma/v2/adapters/humachi"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/cors"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"go.temporal.io/sdk/client"
)

func main() {
	// Setup logger
	zerolog.TimeFieldFormat = zerolog.TimeFormatUnix
	log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr})

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to load configuration")
	}

	log.Info().Str("env", cfg.Server.Env).Msg("Starting RSS Summarizer API")

	// Connect to database
	db, err := database.New(cfg.Database.URL)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to connect to database")
	}
	defer db.Close()

	log.Info().Msg("Connected to database")

	// Initialize crypto service for API key encryption
	cryptoService, err := crypto.NewService(cfg.EncryptionKey)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to initialize crypto service")
	}

	// Initialize repositories
	userRepo := repository.NewUserRepository(db)
	sessionRepo := repository.NewSessionRepository(db)
	feedRepo := repository.NewFeedRepository(db)
	articleRepo := repository.NewArticleRepository(db)
	subscriptionRepo := repository.NewSubscriptionRepository(db)
	userArticleRepo := repository.NewUserArticleRepository(db)
	prefsRepo := repository.NewPreferencesRepository(db, cryptoService)
	topicRepo := repository.NewTopicRepository(db)
	llmConfigRepo := repository.NewLLMConfigRepository(db, cryptoService)
	emailSourceRepo := repository.NewEmailSourceRepository(db, cryptoService)
	newsletterFilterRepo := repository.NewNewsletterFilterRepository(db)

	// Initialize OAuth service
	var oauthService auth.OAuthService
	if !cfg.DevMode.Enabled {
		oauthService = auth.NewOAuthService(
			cfg.OAuth.GoogleClientID,
			cfg.OAuth.GoogleClientSecret,
			cfg.OAuth.GoogleRedirectURL,
		)
	}

	// Initialize services
	rssService := rss.NewService()
	gmailService := gmail.NewService(
		cfg.OAuth.GoogleClientID,
		cfg.OAuth.GoogleClientSecret,
		cfg.Server.PublicAPIURL+"/v1/auth/gmail/callback",
	)

	// Get LLM config from database (falls back to env vars if not set)
	llmConfig, err := llmConfigRepo.Get(context.Background())
	if err != nil {
		log.Warn().Err(err).Msg("Failed to load LLM config from database, using environment config")
		llmConfig = nil
	}

	// Use database config if available, otherwise fall back to environment variables
	llmAPIURL := cfg.LLM.APIURL
	llmAPIKey := cfg.LLM.APIKey
	llmModel := cfg.LLM.Model

	if llmConfig != nil {
		if llmConfig.APIURL != "" {
			llmAPIURL = llmConfig.APIURL
		}
		if llmConfig.APIKey != "" {
			llmAPIKey = llmConfig.APIKey
		}
		if llmConfig.Model != "" {
			llmModel = llmConfig.Model
		}
		log.Info().
			Str("provider", llmConfig.Provider).
			Str("model", llmModel).
			Msg("Using LLM config from database")
	} else {
		log.Info().
			Str("model", llmModel).
			Msg("Using LLM config from environment")
	}

	llmService := llm.NewService(llmAPIURL, llmAPIKey, llmModel)

	// Create Temporal client (for API handlers to trigger workflows)
	temporalClient, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
	})
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Temporal client")
	}
	defer temporalClient.Close()

	// Initialize Temporal worker
	activities := workflow.NewActivities(
		feedRepo,
		articleRepo,
		prefsRepo,
		topicRepo,
		subscriptionRepo,
		userArticleRepo,
		rssService,
		llmService,
	)
	emailActivities := workflow.NewEmailActivities(
		emailSourceRepo,
		newsletterFilterRepo,
		articleRepo,
		userArticleRepo,
		gmailService,
	)
	temporalWorker, err := workflow.NewWorker(cfg, activities, emailActivities)
	if err != nil {
		log.Fatal().Err(err).Msg("Failed to create Temporal worker")
	}

	// Start Temporal worker in background with retry
	go func() {
		for i := range 3 {
			log.Info().Msgf("Starting Temporal worker (attempt %d/3)", i+1)
			if err := temporalWorker.Start(); err != nil {
				log.Error().Err(err).Msgf("Temporal worker failed on attempt %d", i+1)
				time.Sleep(time.Second * 2)
				continue
			}
			log.Info().Msg("Temporal worker started successfully")
			return
		}
		log.Fatal().Msg("Failed to start Temporal worker after 3 attempts")
	}()
	defer temporalWorker.Stop()

	// Start feed polling ticker (every 5 minutes)
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()

		// Trigger immediately on startup
		triggerFeedPoller(cfg)

		for range ticker.C {
			triggerFeedPoller(cfg)
		}
	}()

	// Start email polling ticker (every 10 minutes)
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()

		// Trigger immediately on startup
		triggerEmailPoller(cfg)

		for range ticker.C {
			triggerEmailPoller(cfg)
		}
	}()

	// Start session cleanup job (runs hourly)
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()

		for range ticker.C {
			count, err := sessionRepo.DeleteExpired(context.Background())
			if err != nil {
				log.Error().Err(err).Msg("Failed to clean up expired sessions")
			} else if count > 0 {
				log.Info().Int64("count", count).Msg("Cleaned up expired sessions")
			}
		}
	}()

	// Setup HTTP server
	router := chi.NewRouter()

	// Add CORS middleware
	router.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:5173"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type"},
		ExposedHeaders:   []string{"Link"},
		AllowCredentials: true, // Enable cookies
		MaxAge:           300,
	}))

	// Add global middlewares
	router.Use(middleware.InjectResponseWriter()) // Needed for setting cookies
	router.Use(middleware.LoggingMiddleware(log.Logger))

	// Apply authentication middleware (skips public routes)
	router.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip auth middleware for public endpoints
			publicPaths := []string{
				"/auth/google/login",
				"/auth/google/callback",
				"/v1/auth/gmail/callback",
				"/openapi.json",
			}

			if slices.Contains(publicPaths, r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}

			// Apply appropriate auth middleware for other routes
			var authHandler http.Handler
			if cfg.DevMode.Enabled {
				authHandler = middleware.DevAuthMiddleware(cfg)(next)
			} else {
				authHandler = middleware.SessionAuthMiddleware(cfg, sessionRepo, userRepo)(next)
			}
			authHandler.ServeHTTP(w, r)
		})
	})

	// Setup single Huma API (automatically exposes OpenAPI schema at /openapi.json)
	api := humachi.New(router, huma.DefaultConfig("RSS Summarizer API", "1.0.0"))

	// Register auth handlers (no auth required)
	authHandlers := handlers.NewAuthHandlers(cfg, oauthService, userRepo, sessionRepo, prefsRepo)
	authHandlers.Register(api)

	// Register protected handlers
	healthHandlers := handlers.NewHealthHandlers(db)
	healthHandlers.Register(api)

	feedHandlers := handlers.NewFeedHandlers(feedRepo, subscriptionRepo, rssService)
	feedHandlers.Register(api)

	articleHandlers := handlers.NewArticleHandlers(articleRepo, userArticleRepo, temporalClient)
	articleHandlers.Register(api)

	preferencesHandlers := handlers.NewPreferencesHandlers(prefsRepo)
	preferencesHandlers.Register(api)

	topicHandlers := handlers.NewTopicHandlers(topicRepo)
	topicHandlers.Register(api)

	// Register Gmail OAuth handlers
	gmailHandlers := handlers.NewGmailHandlers(cfg, gmailService, emailSourceRepo, db)
	gmailHandlers.Register(api)

	// Register email source handlers
	emailSourceHandlers := handlers.NewEmailSourceHandlers(emailSourceRepo)
	emailSourceHandlers.Register(api)

	// Register newsletter filter handlers
	newsletterFilterHandlers := handlers.NewNewsletterFilterHandlers(newsletterFilterRepo, emailSourceRepo)
	newsletterFilterHandlers.Register(api)

	// Register admin handlers (require admin role - checked in handlers)
	adminLLMHandlers := handlers.NewAdminLLMHandlers(llmConfigRepo, userRepo)
	adminLLMHandlers.Register(api)

	adminUserHandlers := handlers.NewAdminUserHandlers(userRepo)
	adminUserHandlers.Register(api)

	// Register monitoring handlers
	monitoringHandlers := handlers.NewMonitoringHandlers(temporalClient, articleRepo, feedRepo, emailSourceRepo)
	monitoringHandlers.Register(api)

	// Start HTTP server
	addr := fmt.Sprintf(":%d", cfg.Server.Port)
	server := &http.Server{
		Addr:         addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Handle graceful shutdown
	go func() {
		log.Info().Str("addr", addr).Msg("Server listening")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatal().Err(err).Msg("Server failed")
		}
	}()

	// Wait for interrupt signal
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Info().Msg("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Fatal().Err(err).Msg("Server forced to shutdown")
	}

	log.Info().Msg("Server exited")
}

func triggerFeedPoller(cfg *config.Config) {
	c, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Temporal client for feed poller")
		return
	}
	defer c.Close()

	// Execute the feed poller workflow (will complete after one poll cycle)
	workflowID := fmt.Sprintf("feed-poller-%d", time.Now().Unix())
	_, err = c.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: workflow.FeedPollingTaskQueue,
		},
		workflow.FeedPollerWorkflow,
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to trigger feed poller workflow")
		return
	}

	log.Info().Str("workflowID", workflowID).Msg("Triggered feed poller workflow")
}

func triggerEmailPoller(cfg *config.Config) {
	c, err := client.Dial(client.Options{
		HostPort:  cfg.Temporal.Host,
		Namespace: cfg.Temporal.Namespace,
	})
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Temporal client for email poller")
		return
	}
	defer c.Close()

	// Execute the email poller workflow (will complete after one poll cycle)
	workflowID := fmt.Sprintf("email-poller-%d", time.Now().Unix())
	_, err = c.ExecuteWorkflow(
		context.Background(),
		client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: workflow.EmailPollingTaskQueue,
		},
		workflow.EmailPollerWorkflow,
	)

	if err != nil {
		log.Error().Err(err).Msg("Failed to trigger email poller workflow")
		return
	}

	log.Info().Str("workflowID", workflowID).Msg("Triggered email poller workflow")
}
