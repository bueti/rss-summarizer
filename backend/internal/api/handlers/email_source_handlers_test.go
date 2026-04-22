package handlers_test

import (
	"net/http"
	"testing"
	"time"

	apitest "github.com/bbu/rss-summarizer/backend/internal/api/testing"
	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/google/uuid"
)

func TestListEmailSources(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*apitest.TestServer)
		wantStatus int
		wantCount  int
	}{
		{
			name: "no email sources",
			setup: func(ts *apitest.TestServer) {
				// No setup needed
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "with email sources",
			setup: func(ts *apitest.TestServer) {
				// Create test email sources directly in database
				now := time.Now()
				source1 := &email_source.CreateEmailSourceInput{
					UserID:         ts.UserID,
					EmailAddress:   "test1@gmail.com",
					Provider:       email_source.ProviderGmail,
					AccessToken:    "test-access-token-1",
					RefreshToken:   "test-refresh-token-1",
					TokenExpiresAt: now.Add(1 * time.Hour),
				}
				source2 := &email_source.CreateEmailSourceInput{
					UserID:         ts.UserID,
					EmailAddress:   "test2@gmail.com",
					Provider:       email_source.ProviderGmail,
					AccessToken:    "test-access-token-2",
					RefreshToken:   "test-refresh-token-2",
					TokenExpiresAt: now.Add(1 * time.Hour),
				}

				_, err := ts.EmailSourceRepo.Create(ts.Ctx, source1)
				if err != nil {
					t.Fatalf("Failed to create test email source 1: %v", err)
				}
				_, err = ts.EmailSourceRepo.Create(ts.Ctx, source2)
				if err != nil {
					t.Fatalf("Failed to create test email source 2: %v", err)
				}
			},
			wantStatus: http.StatusOK,
			wantCount:  2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := apitest.NewTestServer(t)
			defer ts.Close(t)

			if tt.setup != nil {
				tt.setup(ts)
			}

			w := ts.Request(t, "GET", "/v1/email-sources", nil)

			apitest.AssertStatus(t, w, tt.wantStatus)

			var resp struct {
				EmailSources []struct {
					ID           string `json:"id"`
					EmailAddress string `json:"email_address"`
					Provider     string `json:"provider"`
					IsActive     bool   `json:"is_active"`
				} `json:"email_sources"`
			}
			apitest.DecodeResponse(t, w, &resp)

			if len(resp.EmailSources) != tt.wantCount {
				t.Errorf("Expected %d email sources, got %d", tt.wantCount, len(resp.EmailSources))
			}

			// Verify tokens are not exposed
			if tt.wantCount > 0 {
				for _, source := range resp.EmailSources {
					if source.ID == "" {
						t.Error("Expected email source to have an ID")
					}
					if source.EmailAddress == "" {
						t.Error("Expected email source to have an email address")
					}
					if source.Provider == "" {
						t.Error("Expected email source to have a provider")
					}
				}
			}
		})
	}
}

func TestGetEmailSource(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*apitest.TestServer) string
		wantStatus int
	}{
		{
			name: "existing email source",
			setup: func(ts *apitest.TestServer) string {
				now := time.Now()
				source := &email_source.CreateEmailSourceInput{
					UserID:         ts.UserID,
					EmailAddress:   "test@gmail.com",
					Provider:       email_source.ProviderGmail,
					AccessToken:    "test-access-token",
					RefreshToken:   "test-refresh-token",
					TokenExpiresAt: now.Add(1 * time.Hour),
				}

				created, err := ts.EmailSourceRepo.Create(ts.Ctx, source)
				if err != nil {
					t.Fatalf("Failed to create test email source: %v", err)
				}
				return created.ID.String()
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "non-existent email source",
			setup: func(ts *apitest.TestServer) string {
				return uuid.New().String()
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "invalid uuid",
			setup: func(ts *apitest.TestServer) string {
				return "not-a-uuid"
			},
			wantStatus: http.StatusUnprocessableEntity, // Huma returns 422 for validation errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := apitest.NewTestServer(t)
			defer ts.Close(t)

			sourceID := tt.setup(ts)

			w := ts.Request(t, "GET", "/v1/email-sources/"+sourceID, nil)

			apitest.AssertStatus(t, w, tt.wantStatus)

			if tt.wantStatus == http.StatusOK {
				var resp struct {
					ID           string `json:"id"`
					EmailAddress string `json:"email_address"`
					Provider     string `json:"provider"`
					IsActive     bool   `json:"is_active"`
				}
				apitest.DecodeResponse(t, w, &resp)

				if resp.ID != sourceID {
					t.Errorf("Expected ID %s, got %s", sourceID, resp.ID)
				}
			}
		})
	}
}

func TestDeleteEmailSource(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*apitest.TestServer) string
		wantStatus int
	}{
		{
			name: "delete existing email source",
			setup: func(ts *apitest.TestServer) string {
				now := time.Now()
				source := &email_source.CreateEmailSourceInput{
					UserID:         ts.UserID,
					EmailAddress:   "test@gmail.com",
					Provider:       email_source.ProviderGmail,
					AccessToken:    "test-access-token",
					RefreshToken:   "test-refresh-token",
					TokenExpiresAt: now.Add(1 * time.Hour),
				}

				created, err := ts.EmailSourceRepo.Create(ts.Ctx, source)
				if err != nil {
					t.Fatalf("Failed to create test email source: %v", err)
				}
				return created.ID.String()
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "delete non-existent email source",
			setup: func(ts *apitest.TestServer) string {
				return uuid.New().String()
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "delete with invalid uuid",
			setup: func(ts *apitest.TestServer) string {
				return "not-a-uuid"
			},
			wantStatus: http.StatusUnprocessableEntity, // Huma returns 422 for validation errors
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := apitest.NewTestServer(t)
			defer ts.Close(t)

			sourceID := tt.setup(ts)

			w := ts.Request(t, "DELETE", "/v1/email-sources/"+sourceID, nil)

			apitest.AssertStatus(t, w, tt.wantStatus)

			// Verify deletion
			if tt.wantStatus == http.StatusNoContent {
				_, err := ts.EmailSourceRepo.FindByID(ts.Ctx, uuid.MustParse(sourceID))
				if err == nil {
					t.Error("Expected email source to be deleted")
				}
			}
		})
	}
}

func TestDeleteEmailSourceCascadesFilters(t *testing.T) {
	ts := apitest.NewTestServer(t)
	defer ts.Close(t)

	// Create email source
	now := time.Now()
	source := &email_source.CreateEmailSourceInput{
		UserID:         ts.UserID,
		EmailAddress:   "test@gmail.com",
		Provider:       email_source.ProviderGmail,
		AccessToken:    "test-access-token",
		RefreshToken:   "test-refresh-token",
		TokenExpiresAt: now.Add(1 * time.Hour),
	}

	created, err := ts.EmailSourceRepo.Create(ts.Ctx, source)
	if err != nil {
		t.Fatalf("Failed to create test email source: %v", err)
	}

	// Create newsletter filter
	body := map[string]any{
		"email_source_id": created.ID.String(),
		"name":            "Test Filter",
		"sender_pattern":  "*@substack.com",
	}

	w := ts.Request(t, "POST", "/v1/newsletter-filters", body)
	apitest.AssertStatus(t, w, http.StatusOK)

	var filterResp struct {
		ID string `json:"id"`
	}
	apitest.DecodeResponse(t, w, &filterResp)

	// Delete email source
	w = ts.Request(t, "DELETE", "/v1/email-sources/"+created.ID.String(), nil)
	apitest.AssertStatus(t, w, http.StatusNoContent)

	// Verify filter was also deleted (cascade)
	w = ts.Request(t, "GET", "/v1/newsletter-filters/"+filterResp.ID, nil)
	apitest.AssertStatus(t, w, http.StatusNotFound)
}
