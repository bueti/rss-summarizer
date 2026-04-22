package handlers

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/config"
	"github.com/bbu/rss-summarizer/backend/internal/domain/preferences"
	"github.com/bbu/rss-summarizer/backend/internal/domain/session"
	"github.com/bbu/rss-summarizer/backend/internal/domain/user"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/bbu/rss-summarizer/backend/internal/service/auth"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

type AuthHandlers struct {
	cfg          *config.Config
	oauthService auth.OAuthService
	userRepo     repository.UserRepository
	sessionRepo  repository.SessionRepository
	prefsRepo    repository.PreferencesRepository
}

func NewAuthHandlers(
	cfg *config.Config,
	oauthService auth.OAuthService,
	userRepo repository.UserRepository,
	sessionRepo repository.SessionRepository,
	prefsRepo repository.PreferencesRepository,
) *AuthHandlers {
	return &AuthHandlers{
		cfg:          cfg,
		oauthService: oauthService,
		userRepo:     userRepo,
		sessionRepo:  sessionRepo,
		prefsRepo:    prefsRepo,
	}
}

type GoogleLoginResponse struct {
	Body struct {
		AuthURL string `json:"auth_url" doc:"Google OAuth authorization URL"`
	}
}

type GoogleCallbackRequest struct {
	Code  string `query:"code" required:"true" doc:"OAuth authorization code"`
	State string `query:"state" required:"true" doc:"OAuth state token"`
}

type GoogleCallbackResponse struct {
	Body struct {
		RedirectURL string `json:"redirect_url" doc:"URL to redirect user to after authentication"`
	}
}

type LogoutResponse struct{}

type MeResponse struct {
	Body struct {
		ID         uuid.UUID `json:"id"`
		Email      string    `json:"email"`
		Name       string    `json:"name"`
		PictureURL *string   `json:"picture_url,omitempty"`
		Role       string    `json:"role"`
		CreatedAt  time.Time `json:"created_at"`
	}
}

func (h *AuthHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "google-login",
		Method:      http.MethodGet,
		Path:        "/auth/google/login",
		Summary:     "Initiate Google OAuth login",
		Description: "Returns the Google OAuth authorization URL to redirect the user to",
		Tags:        []string{"Auth"},
	}, h.GoogleLogin)

	huma.Register(api, huma.Operation{
		OperationID: "google-callback",
		Method:      http.MethodGet,
		Path:        "/auth/google/callback",
		Summary:     "Handle Google OAuth callback",
		Description: "Processes the OAuth callback, creates a session, and returns redirect URL",
		Tags:        []string{"Auth"},
	}, h.GoogleCallback)

	huma.Register(api, huma.Operation{
		OperationID: "logout",
		Method:      http.MethodPost,
		Path:        "/auth/logout",
		Summary:     "Logout and clear session",
		Description: "Deletes all sessions for the current user and clears the session cookie",
		Tags:        []string{"Auth"},
	}, h.Logout)

	huma.Register(api, huma.Operation{
		OperationID: "me",
		Method:      http.MethodGet,
		Path:        "/auth/me",
		Summary:     "Get current user info",
		Description: "Returns information about the currently authenticated user",
		Tags:        []string{"Auth"},
	}, h.Me)
}

func (h *AuthHandlers) GoogleLogin(ctx context.Context, input *struct{}) (*GoogleLoginResponse, error) {
	// Generate CSRF state token
	state, err := h.oauthService.GenerateStateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state token: %w", err)
	}

	// Get OAuth authorization URL
	authURL := h.oauthService.GetAuthURL(state)

	return &GoogleLoginResponse{
		Body: struct {
			AuthURL string `json:"auth_url" doc:"Google OAuth authorization URL"`
		}{
			AuthURL: authURL,
		},
	}, nil
}

func (h *AuthHandlers) GoogleCallback(ctx context.Context, input *GoogleCallbackRequest) (*GoogleCallbackResponse, error) {
	log.Info().
		Str("code", input.Code[:10]+"...").
		Str("state", input.State[:10]+"...").
		Msg("Processing OAuth callback")

	// Exchange authorization code for token
	token, err := h.oauthService.ExchangeCode(ctx, input.Code)
	if err != nil {
		log.Error().Err(err).Msg("Failed to exchange OAuth code")
		return nil, huma.Error400BadRequest(fmt.Sprintf("Failed to exchange code: %v", err))
	}

	log.Info().Msg("Successfully exchanged OAuth code for token")

	// Get user info from Google
	userInfo, err := h.oauthService.GetUserInfo(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user info from Google")
		return nil, huma.Error400BadRequest(fmt.Sprintf("Failed to get user info: %v", err))
	}

	log.Info().
		Str("email", userInfo.Email).
		Bool("email_verified", userInfo.EmailVerified).
		Msg("Retrieved user info from Google")

	if !userInfo.EmailVerified {
		log.Warn().Str("email", userInfo.Email).Msg("Email not verified")
		return nil, huma.Error400BadRequest("Email not verified")
	}

	// Find or create user
	u, err := h.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	// Create session
	expiresAt := time.Now().Add(h.cfg.OAuth.SessionDuration)
	sess, err := h.sessionRepo.Create(ctx, &session.CreateSessionInput{
		UserID:    u.ID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	// Set session cookie and redirect
	w, ok := middleware.GetResponseWriter(ctx)
	if ok {
		secure := h.cfg.Server.Env != "development"
		middleware.SetSessionCookie(w, sess.SessionToken, expiresAt, secure)

		log.Info().
			Str("user_id", u.ID.String()).
			Str("email", u.Email).
			Msg("User logged in via Google OAuth")

		// Determine redirect URL based on environment
		redirectURL := "http://localhost:5173/"
		if h.cfg.Server.Env == "production" {
			redirectURL = "/"
		}

		// Send HTTP redirect
		w.Header().Set("Location", redirectURL)
		w.WriteHeader(http.StatusFound) // 302 redirect
		return nil, nil                 // Signal to Huma that we've handled the response
	}

	// Fallback if we don't have ResponseWriter (shouldn't happen)
	log.Info().
		Str("user_id", u.ID.String()).
		Str("email", u.Email).
		Msg("User logged in via Google OAuth")

	return &GoogleCallbackResponse{
		Body: struct {
			RedirectURL string `json:"redirect_url" doc:"URL to redirect user to after authentication"`
		}{
			RedirectURL: "/",
		},
	}, nil
}

func (h *AuthHandlers) Logout(ctx context.Context, input *struct{}) (*LogoutResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return &LogoutResponse{}, nil // Already logged out
	}

	// Delete all sessions for this user
	if err := h.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		log.Error().Err(err).Msg("Failed to delete sessions during logout")
	}

	// Clear cookie
	w, ok := middleware.GetResponseWriter(ctx)
	if ok {
		middleware.ClearSessionCookie(w)
	}

	log.Info().Str("user_id", userID.String()).Msg("User logged out")

	return &LogoutResponse{}, nil
}

func (h *AuthHandlers) Me(ctx context.Context, input *struct{}) (*MeResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	u, err := h.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, huma.Error404NotFound("User not found")
	}

	return &MeResponse{
		Body: struct {
			ID         uuid.UUID `json:"id"`
			Email      string    `json:"email"`
			Name       string    `json:"name"`
			PictureURL *string   `json:"picture_url,omitempty"`
			Role       string    `json:"role"`
			CreatedAt  time.Time `json:"created_at"`
		}{
			ID:         u.ID,
			Email:      u.Email,
			Name:       u.Name,
			PictureURL: u.PictureURL,
			Role:       u.Role,
			CreatedAt:  u.CreatedAt,
		},
	}, nil
}

func (h *AuthHandlers) findOrCreateUser(ctx context.Context, userInfo *auth.GoogleUserInfo) (*user.User, error) {
	// Try to find by Google ID first
	u, err := h.userRepo.FindByGoogleID(ctx, userInfo.ID)
	if err == nil {
		// User exists, update info if changed
		updated := false
		if u.Email != userInfo.Email {
			u.Email = userInfo.Email
			updated = true
		}
		if u.Name != userInfo.Name {
			u.Name = userInfo.Name
			updated = true
		}
		if u.PictureURL == nil || *u.PictureURL != userInfo.Picture {
			u.PictureURL = &userInfo.Picture
			updated = true
		}

		if updated {
			if err := h.userRepo.Update(ctx, u); err != nil {
				log.Error().Err(err).Msg("Failed to update user info")
			}
		}

		return u, nil
	}

	// Try to find by email (legacy user or email changed in Google)
	u, err = h.userRepo.FindByEmail(ctx, userInfo.Email)
	if err == nil {
		// Link Google ID to existing user
		googleID := userInfo.ID
		u.GoogleID = &googleID
		u.PictureURL = &userInfo.Picture
		if err := h.userRepo.Update(ctx, u); err != nil {
			return nil, fmt.Errorf("failed to link google_id: %w", err)
		}
		log.Info().
			Str("user_id", u.ID.String()).
			Str("email", u.Email).
			Msg("Linked Google ID to existing user")
		return u, nil
	}

	// Create new user
	googleID := userInfo.ID
	u = &user.User{
		Email:      userInfo.Email,
		Name:       userInfo.Name,
		GoogleID:   &googleID,
		PictureURL: &userInfo.Picture,
	}

	if err := h.userRepo.Create(ctx, u); err != nil {
		return nil, fmt.Errorf("failed to create user: %w", err)
	}

	// Create default preferences for new user
	defaultPrefs := &preferences.UserPreferences{
		UserID:              u.ID,
		DefaultPollInterval: 60, // 60 minutes default
		MaxArticlesPerFeed:  50,
	}
	if err := h.prefsRepo.Upsert(ctx, defaultPrefs); err != nil {
		log.Error().Err(err).Str("user_id", u.ID.String()).Msg("Failed to create default preferences")
	}

	log.Info().
		Str("user_id", u.ID.String()).
		Str("email", u.Email).
		Msg("New user created via Google OAuth")

	return u, nil
}
