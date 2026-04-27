package handlers

import (
	"context"
	"crypto/subtle"
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

const oauthStateCookieName = "rss_oauth_state"

type GoogleLoginResponse struct {
	SetCookie []http.Cookie `header:"Set-Cookie"`
	Body      struct {
		AuthURL string `json:"auth_url" doc:"Google OAuth authorization URL"`
	}
}

type GoogleCallbackRequest struct {
	Code        string `query:"code" required:"true" doc:"OAuth authorization code"`
	State       string `query:"state" required:"true" doc:"OAuth state token"`
	StateCookie string `cookie:"rss_oauth_state" required:"false" doc:"CSRF state token stored at login"`
}

type GoogleCallbackResponse struct {
	Location  string        `header:"Location"`
	SetCookie []http.Cookie `header:"Set-Cookie"`
}

type LogoutResponse struct {
	SetCookie http.Cookie `header:"Set-Cookie"`
}

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
		OperationID:   "google-callback",
		Method:        http.MethodGet,
		Path:          "/auth/google/callback",
		Summary:       "Handle Google OAuth callback",
		Description:   "Processes the OAuth callback, creates a session, and redirects to the frontend",
		Tags:          []string{"Auth"},
		DefaultStatus: http.StatusFound,
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

// oauthStateCookieTTL bounds how long a generated state token is valid. The
// OAuth round trip normally completes in seconds; 10 minutes tolerates slow
// networks and IdP interstitials without leaving stale tokens on disk.
const oauthStateCookieTTL = 10 * time.Minute

func (h *AuthHandlers) stateCookie(value string, clear bool) http.Cookie {
	// Path "/" so the cookie is sent regardless of any reverse-proxy prefix
	// (e.g. when the API is mounted at /api/). The cookie is HttpOnly,
	// Secure, SameSite=Lax, and lives for oauthStateCookieTTL.
	c := http.Cookie{
		Name:     oauthStateCookieName,
		Value:    value,
		Path:     "/",
		HttpOnly: true,
		Secure:   h.cfg.Server.Env != "development",
		SameSite: http.SameSiteLaxMode,
	}
	if clear {
		c.MaxAge = -1
	} else {
		c.MaxAge = int(oauthStateCookieTTL.Seconds())
	}
	return c
}

func (h *AuthHandlers) GoogleLogin(ctx context.Context, input *struct{}) (*GoogleLoginResponse, error) {
	state, err := h.oauthService.GenerateStateToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate state token: %w", err)
	}

	resp := &GoogleLoginResponse{
		SetCookie: []http.Cookie{h.stateCookie(state, false)},
	}
	resp.Body.AuthURL = h.oauthService.GetAuthURL(state)
	return resp, nil
}

func (h *AuthHandlers) GoogleCallback(ctx context.Context, input *GoogleCallbackRequest) (*GoogleCallbackResponse, error) {
	log.Info().
		Str("code", truncateForLog(input.Code)).
		Str("state", truncateForLog(input.State)).
		Msg("Processing OAuth callback")

	// CSRF defense: the state we set as a cookie in GoogleLogin must match the
	// state Google echoed back in the query string. Without this check an
	// attacker can force the victim into the attacker's Google session.
	if input.StateCookie == "" || input.State == "" ||
		subtle.ConstantTimeCompare([]byte(input.State), []byte(input.StateCookie)) != 1 {
		log.Warn().Msg("OAuth state mismatch; rejecting callback")
		return nil, huma.Error400BadRequest("Invalid OAuth state")
	}

	token, err := h.oauthService.ExchangeCode(ctx, input.Code)
	if err != nil {
		log.Error().Err(err).Msg("Failed to exchange OAuth code")
		return nil, huma.Error400BadRequest(fmt.Sprintf("Failed to exchange code: %v", err))
	}

	userInfo, err := h.oauthService.GetUserInfo(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user info from Google")
		return nil, huma.Error400BadRequest(fmt.Sprintf("Failed to get user info: %v", err))
	}

	if !userInfo.EmailVerified {
		log.Warn().Str("email", userInfo.Email).Msg("Email not verified")
		return nil, huma.Error400BadRequest("Email not verified")
	}

	u, err := h.findOrCreateUser(ctx, userInfo)
	if err != nil {
		return nil, fmt.Errorf("failed to find or create user: %w", err)
	}

	expiresAt := time.Now().Add(h.cfg.OAuth.SessionDuration)
	sess, err := h.sessionRepo.Create(ctx, &session.CreateSessionInput{
		UserID:    u.ID,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	log.Info().
		Str("user_id", u.ID.String()).
		Str("email", u.Email).
		Msg("User logged in via Google OAuth")

	secure := h.cfg.Server.Env != "development"
	redirectURL := h.cfg.Server.FrontendURL
	if redirectURL == "" {
		redirectURL = "/"
	}

	return &GoogleCallbackResponse{
		Location: redirectURL,
		SetCookie: []http.Cookie{
			middleware.SessionCookie(sess.SessionToken, expiresAt, secure),
			h.stateCookie("", true),
		},
	}, nil
}

func (h *AuthHandlers) Logout(ctx context.Context, input *struct{}) (*LogoutResponse, error) {
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		// Already logged out; still send a clearing cookie so stale browsers
		// drop their token.
		return &LogoutResponse{SetCookie: middleware.ClearSessionCookieValue(h.cfg.Server.Env != "development")}, nil
	}

	if err := h.sessionRepo.DeleteByUserID(ctx, userID); err != nil {
		log.Error().Err(err).Msg("Failed to delete sessions during logout")
	}

	log.Info().Str("user_id", userID.String()).Msg("User logged out")

	return &LogoutResponse{
		SetCookie: middleware.ClearSessionCookieValue(h.cfg.Server.Env != "development"),
	}, nil
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
