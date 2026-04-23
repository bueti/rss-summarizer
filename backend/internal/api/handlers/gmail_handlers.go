package handlers

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/base64"
	stderrors "errors"
	"fmt"
	"net/http"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/api/middleware"
	"github.com/bbu/rss-summarizer/backend/internal/config"
	"github.com/bbu/rss-summarizer/backend/internal/database"
	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/bbu/rss-summarizer/backend/internal/service/gmail"
	"github.com/danielgtaylor/huma/v2"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
)

// oauthStateTokenTTL bounds how long a Gmail-OAuth state token is considered
// fresh. A user typically completes consent in seconds; this is deliberately
// tight so stolen or leaked tokens have a short window.
const oauthStateTokenTTL = 15 * time.Minute

type GmailHandlers struct {
	cfg             *config.Config
	gmailService    *gmail.Service
	emailSourceRepo email_source.Repository
	db              *database.DB
}

func NewGmailHandlers(
	cfg *config.Config,
	gmailService *gmail.Service,
	emailSourceRepo email_source.Repository,
	db *database.DB,
) *GmailHandlers {
	return &GmailHandlers{
		cfg:             cfg,
		gmailService:    gmailService,
		emailSourceRepo: emailSourceRepo,
		db:              db,
	}
}

type ConnectGmailResponse struct {
	Body struct {
		AuthURL string `json:"auth_url" doc:"Gmail OAuth authorization URL"`
	}
}

type GmailCallbackRequest struct {
	Code  string `query:"code" required:"true" doc:"OAuth authorization code"`
	State string `query:"state" required:"true" doc:"OAuth state token"`
}

type GmailCallbackResponse struct {
	Location string `header:"Location"`
}

func (h *GmailHandlers) Register(api huma.API) {
	huma.Register(api, huma.Operation{
		OperationID: "connect-gmail",
		Method:      http.MethodGet,
		Path:        "/v1/auth/gmail/connect",
		Summary:     "Connect Gmail account for newsletter fetching",
		Description: "Initiates OAuth flow to connect user's Gmail account",
		Tags:        []string{"Gmail"},
	}, h.ConnectGmail)

	huma.Register(api, huma.Operation{
		OperationID:   "gmail-callback",
		Method:        http.MethodGet,
		Path:          "/v1/auth/gmail/callback",
		Summary:       "Handle Gmail OAuth callback",
		Description:   "Processes the Gmail OAuth callback and stores access tokens",
		Tags:          []string{"Gmail"},
		DefaultStatus: http.StatusFound,
	}, h.GmailCallback)
}

func (h *GmailHandlers) ConnectGmail(ctx context.Context, input *struct{}) (*ConnectGmailResponse, error) {
	// User must be authenticated
	userID, ok := middleware.GetUserIDFromContext(ctx)
	if !ok {
		return nil, huma.Error401Unauthorized("Not authenticated")
	}

	// Generate CSRF state token
	state, err := h.generateStateToken(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to generate state token: %w", err)
	}

	// Get OAuth authorization URL
	oauthConfig := h.gmailService.GetOAuthConfig()
	authURL := oauthConfig.AuthCodeURL(state, oauth2.AccessTypeOffline, oauth2.ApprovalForce)

	log.Info().
		Str("user_id", userID.String()).
		Msg("Generated Gmail OAuth URL")

	return &ConnectGmailResponse{
		Body: struct {
			AuthURL string `json:"auth_url" doc:"Gmail OAuth authorization URL"`
		}{
			AuthURL: authURL,
		},
	}, nil
}

func (h *GmailHandlers) GmailCallback(ctx context.Context, input *GmailCallbackRequest) (*GmailCallbackResponse, error) {
	log.Info().
		Str("code", truncateForLog(input.Code)).
		Str("state", truncateForLog(input.State)).
		Msg("Processing Gmail OAuth callback")

	userID, err := h.verifyStateToken(ctx, input.State)
	if err != nil {
		log.Error().Err(err).Msg("Invalid or expired state token")
		return nil, huma.Error400BadRequest("Invalid or expired state token")
	}

	token, err := h.gmailService.ExchangeCode(ctx, input.Code)
	if err != nil {
		log.Error().Err(err).Msg("Failed to exchange OAuth code")
		return nil, huma.Error400BadRequest(fmt.Sprintf("Failed to exchange code: %v", err))
	}

	emailAddress, err := h.gmailService.GetUserEmail(ctx, token)
	if err != nil {
		log.Error().Err(err).Msg("Failed to get user email from Gmail")
		return nil, huma.Error400BadRequest(fmt.Sprintf("Failed to get email address: %v", err))
	}

	log.Info().
		Str("user_id", userID.String()).
		Str("email", emailAddress).
		Msg("Retrieved email address from Gmail")

	existingSources, err := h.emailSourceRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to check existing email sources: %w", err)
	}

	for _, source := range existingSources {
		if source.EmailAddress == emailAddress && source.Provider == email_source.ProviderGmail {
			updateInput := &email_source.UpdateEmailSourceInput{
				AccessToken:    &token.AccessToken,
				RefreshToken:   &token.RefreshToken,
				TokenExpiresAt: &token.Expiry,
			}
			if _, err := h.emailSourceRepo.Update(ctx, source.ID, updateInput); err != nil {
				return nil, fmt.Errorf("failed to update email source: %w", err)
			}

			log.Info().
				Str("user_id", userID.String()).
				Str("email", emailAddress).
				Msg("Updated existing Gmail connection")

			return &GmailCallbackResponse{
				Location: h.getRedirectURL("/email-sources/callback?status=success&message=Gmail+account+reconnected"),
			}, nil
		}
	}

	createInput := &email_source.CreateEmailSourceInput{
		UserID:         userID,
		EmailAddress:   emailAddress,
		Provider:       email_source.ProviderGmail,
		AccessToken:    token.AccessToken,
		RefreshToken:   token.RefreshToken,
		TokenExpiresAt: token.Expiry,
	}

	if _, err = h.emailSourceRepo.Create(ctx, createInput); err != nil {
		return nil, fmt.Errorf("failed to create email source: %w", err)
	}

	log.Info().
		Str("user_id", userID.String()).
		Str("email", emailAddress).
		Msg("Gmail account connected successfully")

	return &GmailCallbackResponse{
		Location: h.getRedirectURL("/email-sources/callback?status=success&message=Gmail+account+connected"),
	}, nil
}

// generateStateToken generates a random state token for CSRF protection.
func (h *GmailHandlers) generateStateToken(ctx context.Context, userID uuid.UUID) (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	state := base64.URLEncoding.EncodeToString(b)

	query := `INSERT INTO oauth_state_tokens (state, user_id, created_at) VALUES ($1, $2, $3)`
	if _, err := h.db.ExecContext(ctx, query, state, userID, time.Now()); err != nil {
		return "", fmt.Errorf("failed to store state token: %w", err)
	}

	return state, nil
}

// verifyStateToken atomically consumes a state token and returns the bound
// userID. Using DELETE ... RETURNING makes verification and single-use
// consumption one operation, closing the race where two callbacks with the
// same state could both pass a SELECT check.
func (h *GmailHandlers) verifyStateToken(ctx context.Context, state string) (uuid.UUID, error) {
	var userID uuid.UUID
	query := `
		DELETE FROM oauth_state_tokens
		WHERE state = $1 AND created_at > NOW() - $2::interval
		RETURNING user_id
	`
	interval := fmt.Sprintf("%d seconds", int(oauthStateTokenTTL.Seconds()))
	err := h.db.QueryRowContext(ctx, query, state, interval).Scan(&userID)
	if err != nil {
		if stderrors.Is(err, sql.ErrNoRows) {
			// Opportunistically purge an expired row with the same state so it
			// doesn't linger until the background cleanup fires.
			_, _ = h.db.ExecContext(ctx, `DELETE FROM oauth_state_tokens WHERE state = $1`, state)
			return uuid.Nil, fmt.Errorf("state token not found or expired")
		}
		return uuid.Nil, fmt.Errorf("failed to verify state token: %w", err)
	}
	return userID, nil
}

// RunStateTokenCleanup periodically removes expired state tokens until ctx is
// cancelled. Intended to be started as a goroutine from main() so the lifetime
// is tied to the server's lifetime.
func (h *GmailHandlers) RunStateTokenCleanup(ctx context.Context) {
	const interval = time.Hour
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			query := `DELETE FROM oauth_state_tokens WHERE created_at < NOW() - $1::interval`
			ttlArg := fmt.Sprintf("%d seconds", int(oauthStateTokenTTL.Seconds()))
			result, err := h.db.ExecContext(ctx, query, ttlArg)
			if err != nil {
				log.Error().Err(err).Msg("Failed to cleanup expired state tokens")
				continue
			}
			if rowsAffected, _ := result.RowsAffected(); rowsAffected > 0 {
				log.Info().Int64("count", rowsAffected).Msg("Cleaned up expired OAuth state tokens")
			}
		}
	}
}

// getRedirectURL returns the appropriate redirect URL based on environment
func (h *GmailHandlers) getRedirectURL(path string) string {
	return h.cfg.Server.FrontendURL + path
}
