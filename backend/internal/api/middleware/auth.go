package middleware

import (
	"context"
	"net/http"
	"time"

	"github.com/bbu/rss-summarizer/backend/internal/config"
	"github.com/bbu/rss-summarizer/backend/internal/repository"
	"github.com/google/uuid"
)

type contextKey string

const (
	userIDKey         contextKey = "user_id"
	sessionCookieName            = "rss_session"
)

// DevAuthMiddleware provides simple bypass authentication for development
func DevAuthMiddleware(cfg *config.Config) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !cfg.DevMode.Enabled {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Inject dev user ID into context
			userID, err := uuid.Parse(cfg.DevMode.UserID)
			if err != nil {
				http.Error(w, "Invalid dev user ID", http.StatusInternalServerError)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, userID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetUserIDFromContext extracts the user ID from the request context
func GetUserIDFromContext(ctx context.Context) (uuid.UUID, bool) {
	userID, ok := ctx.Value(userIDKey).(uuid.UUID)
	return userID, ok
}

// WithUserID creates a new context with the user ID set (useful for testing)
func WithUserID(ctx context.Context, userID uuid.UUID) context.Context {
	return context.WithValue(ctx, userIDKey, userID)
}

// SessionAuthMiddleware validates session cookies for production
func SessionAuthMiddleware(cfg *config.Config, sessionRepo repository.SessionRepository, userRepo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Skip session auth if in dev mode
			if cfg.DevMode.Enabled {
				next.ServeHTTP(w, r)
				return
			}

			// Get session cookie
			cookie, err := r.Cookie(sessionCookieName)
			if err != nil {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Validate session
			session, err := sessionRepo.FindByToken(r.Context(), cookie.Value)
			if err != nil {
				// Clear invalid cookie
				ClearSessionCookie(w)
				http.Error(w, "Invalid or expired session", http.StatusUnauthorized)
				return
			}

			// Verify user still exists
			_, err = userRepo.FindByID(r.Context(), session.UserID)
			if err != nil {
				sessionRepo.DeleteByToken(r.Context(), cookie.Value)
				ClearSessionCookie(w)
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			// Inject user ID into context
			ctx := context.WithValue(r.Context(), userIDKey, session.UserID)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// SetSessionCookie sets a session cookie
func SetSessionCookie(w http.ResponseWriter, token string, expiresAt time.Time, secure bool) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    token,
		Path:     "/",
		Expires:  expiresAt,
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearSessionCookie clears the session cookie
func ClearSessionCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     sessionCookieName,
		Value:    "",
		Path:     "/",
		MaxAge:   -1,
		HttpOnly: true,
	})
}

// AdminMiddleware ensures the user has admin role
// This middleware should be applied AFTER SessionAuthMiddleware
func AdminMiddleware(userRepo repository.UserRepository) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Get user ID from context (set by auth middleware)
			userID, ok := GetUserIDFromContext(r.Context())
			if !ok {
				http.Error(w, "Authentication required", http.StatusUnauthorized)
				return
			}

			// Get user from database
			user, err := userRepo.FindByID(r.Context(), userID)
			if err != nil {
				http.Error(w, "User not found", http.StatusUnauthorized)
				return
			}

			// Check if user is admin
			if !user.IsAdmin() {
				http.Error(w, "Admin access required", http.StatusForbidden)
				return
			}

			// User is admin, continue
			next.ServeHTTP(w, r)
		})
	}
}
