package gmail

import (
	"context"
	"encoding/base64"
	"fmt"
	"slices"
	"time"

	"github.com/rs/zerolog/log"
	"golang.org/x/oauth2"
	"golang.org/x/oauth2/google"
	"google.golang.org/api/gmail/v1"
	"google.golang.org/api/option"
)

// Service provides Gmail API interactions
type Service struct {
	clientID     string
	clientSecret string
	redirectURL  string
}

// NewService creates a new Gmail service
func NewService(clientID, clientSecret, redirectURL string) *Service {
	return &Service{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
	}
}

// GetOAuthConfig returns the OAuth2 configuration for Gmail API
func (s *Service) GetOAuthConfig() *oauth2.Config {
	return &oauth2.Config{
		ClientID:     s.clientID,
		ClientSecret: s.clientSecret,
		RedirectURL:  s.redirectURL,
		Scopes: []string{
			gmail.GmailReadonlyScope, // Read-only access to Gmail
			gmail.GmailModifyScope,   // Allow marking emails as read
		},
		Endpoint: google.Endpoint,
	}
}

// ExchangeCode exchanges an authorization code for OAuth tokens
func (s *Service) ExchangeCode(ctx context.Context, code string) (*oauth2.Token, error) {
	oauthConfig := s.GetOAuthConfig()
	token, err := oauthConfig.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to exchange code for token: %w", err)
	}
	return token, nil
}

// GetUserEmail retrieves the user's email address from Gmail API
func (s *Service) GetUserEmail(ctx context.Context, token *oauth2.Token) (string, error) {
	client := s.GetOAuthConfig().Client(ctx, token)
	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return "", fmt.Errorf("failed to create Gmail service: %w", err)
	}

	profile, err := gmailService.Users.GetProfile("me").Do()
	if err != nil {
		return "", fmt.Errorf("failed to get user profile: %w", err)
	}

	return profile.EmailAddress, nil
}

// RefreshToken refreshes an expired OAuth token using the refresh token
func (s *Service) RefreshToken(ctx context.Context, refreshToken string) (*oauth2.Token, error) {
	oauthConfig := s.GetOAuthConfig()
	token := &oauth2.Token{
		RefreshToken: refreshToken,
	}

	tokenSource := oauthConfig.TokenSource(ctx, token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("failed to refresh token: %w", err)
	}

	return newToken, nil
}

// EmailMessage represents a simplified email message
type EmailMessage struct {
	ID        string
	ThreadID  string
	From      string
	Subject   string
	Date      time.Time
	Snippet   string
	BodyHTML  string
	BodyPlain string
	IsUnread  bool
}

// FetchEmails fetches emails matching the given query (legacy - use FetchEmailsWithToken instead)
// Query examples:
//   - "from:@substack.com is:unread"
//   - "subject:newsletter after:2024/01/01"
//   - "label:newsletters"
func (s *Service) FetchEmails(ctx context.Context, accessToken, refreshToken string, query string, maxResults int64) ([]*EmailMessage, *oauth2.Token, error) {
	// Create OAuth token without expiry (won't auto-refresh properly)
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}
	return s.FetchEmailsWithToken(ctx, token, query, maxResults)
}

// FetchEmailsWithToken fetches emails using a complete OAuth token (with expiry)
func (s *Service) FetchEmailsWithToken(ctx context.Context, token *oauth2.Token, query string, maxResults int64) ([]*EmailMessage, *oauth2.Token, error) {
	oauthConfig := s.GetOAuthConfig()

	log.Debug().Time("expiry", token.Expiry).Bool("expired", token.Expiry.Before(time.Now())).Msg("Gmail token status")

	// TokenSource auto-refreshes an expired access token on first use.
	tokenSource := oauthConfig.TokenSource(ctx, token)
	client := oauth2.NewClient(ctx, tokenSource)

	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	listCall := gmailService.Users.Messages.List("me").Q(query)
	if maxResults > 0 {
		listCall = listCall.MaxResults(maxResults)
	}

	response, err := listCall.Do()
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list messages: %w", err)
	}

	emails := make([]*EmailMessage, 0, len(response.Messages))
	for _, msg := range response.Messages {
		email, err := s.fetchMessageDetails(ctx, gmailService, msg.Id)
		if err != nil {
			log.Error().Err(err).Str("message_id", msg.Id).Msg("Failed to fetch Gmail message")
			continue
		}
		emails = append(emails, email)
	}

	// Retrieve the (possibly refreshed) token. If this fails the messages have
	// already been fetched successfully, so we keep them and return the
	// original token — a refresh will be retried on the next poll.
	newToken, err := tokenSource.Token()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read refreshed Gmail token; retaining original")
		return emails, token, nil
	}

	if newToken.AccessToken != token.AccessToken {
		log.Info().Time("new_expiry", newToken.Expiry).Msg("Gmail OAuth token refreshed")
	}

	return emails, newToken, nil
}

// fetchMessageDetails fetches full details for a single message
func (s *Service) fetchMessageDetails(ctx context.Context, gmailService *gmail.Service, messageID string) (*EmailMessage, error) {
	msg, err := gmailService.Users.Messages.Get("me", messageID).Format("full").Do()
	if err != nil {
		return nil, fmt.Errorf("failed to get message: %w", err)
	}

	email := &EmailMessage{
		ID:       msg.Id,
		ThreadID: msg.ThreadId,
		Snippet:  msg.Snippet,
	}

	// Parse headers
	for _, header := range msg.Payload.Headers {
		switch header.Name {
		case "From":
			email.From = header.Value
		case "Subject":
			email.Subject = header.Value
		case "Date":
			// Parse RFC2822 date format
			date, err := time.Parse(time.RFC1123Z, header.Value)
			if err != nil {
				// Try alternative format
				date, _ = time.Parse("Mon, 2 Jan 2006 15:04:05 -0700", header.Value)
			}
			email.Date = date
		}
	}

	// Check if unread
	if slices.Contains(msg.LabelIds, "UNREAD") {
		email.IsUnread = true
	}

	// Extract body content
	email.BodyHTML, email.BodyPlain = extractBody(msg.Payload)

	return email, nil
}

// extractBody extracts HTML and plain text body from message payload
func extractBody(payload *gmail.MessagePart) (html, plain string) {
	// Check if body is directly in the payload
	if payload.MimeType == "text/html" && payload.Body.Data != "" {
		decoded, _ := decodeBase64URL(payload.Body.Data)
		return decoded, ""
	}
	if payload.MimeType == "text/plain" && payload.Body.Data != "" {
		decoded, _ := decodeBase64URL(payload.Body.Data)
		return "", decoded
	}

	// Recursively search parts for HTML and plain text
	for _, part := range payload.Parts {
		if part.MimeType == "text/html" && part.Body.Data != "" {
			decoded, _ := decodeBase64URL(part.Body.Data)
			html = decoded
		} else if part.MimeType == "text/plain" && part.Body.Data != "" {
			decoded, _ := decodeBase64URL(part.Body.Data)
			plain = decoded
		} else if len(part.Parts) > 0 {
			// Recursively check nested parts
			h, p := extractBody(part)
			if h != "" {
				html = h
			}
			if p != "" {
				plain = p
			}
		}
	}

	return html, plain
}

// decodeBase64URL decodes base64url-encoded string (used by Gmail API)
func decodeBase64URL(data string) (string, error) {
	// Gmail uses URL-safe base64 encoding without padding
	// Use standard library's RawURLEncoding (no padding)
	decoded, err := base64.RawURLEncoding.DecodeString(data)
	if err != nil {
		// Try with padding if raw fails
		decoded, err = base64.URLEncoding.DecodeString(data)
		if err != nil {
			return "", err
		}
	}
	return string(decoded), nil
}

// MarkAsRead marks an email as read
func (s *Service) MarkAsRead(ctx context.Context, accessToken, refreshToken, messageID string) error {
	token := &oauth2.Token{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
	}

	oauthConfig := s.GetOAuthConfig()
	client := oauthConfig.Client(ctx, token)

	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return fmt.Errorf("failed to create Gmail service: %w", err)
	}

	modifyReq := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}

	_, err = gmailService.Users.Messages.Modify("me", messageID, modifyReq).Do()
	if err != nil {
		return fmt.Errorf("failed to mark message as read: %w", err)
	}

	return nil
}

// MarkAsReadWithToken marks an email as read using a complete OAuth token
func (s *Service) MarkAsReadWithToken(ctx context.Context, token *oauth2.Token, messageID string) (*oauth2.Token, error) {
	oauthConfig := s.GetOAuthConfig()
	tokenSource := oauthConfig.TokenSource(ctx, token)
	client := oauth2.NewClient(ctx, tokenSource)

	gmailService, err := gmail.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return nil, fmt.Errorf("failed to create Gmail service: %w", err)
	}

	modifyReq := &gmail.ModifyMessageRequest{
		RemoveLabelIds: []string{"UNREAD"},
	}

	_, err = gmailService.Users.Messages.Modify("me", messageID, modifyReq).Do()
	if err != nil {
		return nil, fmt.Errorf("failed to mark message as read: %w", err)
	}

	newToken, err := tokenSource.Token()
	if err != nil {
		log.Warn().Err(err).Msg("Failed to read refreshed Gmail token after mark-as-read; retaining original")
		return token, nil
	}

	return newToken, nil
}
