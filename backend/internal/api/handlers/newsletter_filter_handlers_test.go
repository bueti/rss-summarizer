package handlers_test

import (
	"net/http"
	"testing"
	"time"

	apitest "github.com/bbu/rss-summarizer/backend/internal/api/testing"
	"github.com/bbu/rss-summarizer/backend/internal/domain/email_source"
	"github.com/google/uuid"
)

func TestCreateNewsletterFilter(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*apitest.TestServer) string
		body       map[string]any
		wantStatus int
		wantErr    bool
	}{
		{
			name: "valid filter with sender pattern",
			setup: func(ts *apitest.TestServer) string {
				return createTestEmailSource(t, ts)
			},
			body: func() map[string]any {
				return map[string]any{
					"name":           "Substack Newsletters",
					"sender_pattern": "*@substack.com",
				}
			}(),
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "valid filter with sender and subject pattern",
			setup: func(ts *apitest.TestServer) string {
				return createTestEmailSource(t, ts)
			},
			body: func() map[string]any {
				return map[string]any{
					"name":            "Weekly Digests",
					"sender_pattern":  "*@example.com",
					"subject_pattern": "^Weekly Digest",
				}
			}(),
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "missing name",
			setup: func(ts *apitest.TestServer) string {
				return createTestEmailSource(t, ts)
			},
			body: func() map[string]any {
				return map[string]any{
					"sender_pattern": "*@substack.com",
				}
			}(),
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name: "missing sender pattern",
			setup: func(ts *apitest.TestServer) string {
				return createTestEmailSource(t, ts)
			},
			body: func() map[string]any {
				return map[string]any{
					"name": "Test Filter",
				}
			}(),
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name: "invalid email source id",
			setup: func(ts *apitest.TestServer) string {
				return uuid.New().String() // Non-existent
			},
			body: func() map[string]any {
				return map[string]any{
					"name":           "Test Filter",
					"sender_pattern": "*@substack.com",
				}
			}(),
			wantStatus: http.StatusNotFound, // Email source doesn't exist
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := apitest.NewTestServer(t)
			defer ts.Close(t)

			emailSourceID := tt.setup(ts)
			tt.body["email_source_id"] = emailSourceID

			w := ts.Request(t, "POST", "/v1/newsletter-filters", tt.body)

			apitest.AssertStatus(t, w, tt.wantStatus)

			if !tt.wantErr {
				var resp struct {
					ID             string `json:"id"`
					EmailSourceID  string `json:"email_source_id"`
					Name           string `json:"name"`
					SenderPattern  string `json:"sender_pattern"`
					SubjectPattern string `json:"subject_pattern"`
					IsActive       bool   `json:"is_active"`
				}
				apitest.DecodeResponse(t, w, &resp)

				if resp.ID == "" {
					t.Error("Expected filter to have an ID")
				}
				if resp.EmailSourceID != emailSourceID {
					t.Errorf("Expected email_source_id %s, got %s", emailSourceID, resp.EmailSourceID)
				}
				if !resp.IsActive {
					t.Error("Expected new filter to be active by default")
				}
			}
		})
	}
}

func TestListNewsletterFilters(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*apitest.TestServer)
		wantStatus int
		wantCount  int
	}{
		{
			name: "no filters",
			setup: func(ts *apitest.TestServer) {
				// No setup needed
			},
			wantStatus: http.StatusOK,
			wantCount:  0,
		},
		{
			name: "with filters",
			setup: func(ts *apitest.TestServer) {
				sourceID := createTestEmailSource(t, ts)

				// Create two filters
				body1 := map[string]any{
					"email_source_id": sourceID,
					"name":            "Substack",
					"sender_pattern":  "*@substack.com",
				}
				w := ts.Request(t, "POST", "/v1/newsletter-filters", body1)
				apitest.AssertStatus(t, w, http.StatusOK)

				body2 := map[string]any{
					"email_source_id": sourceID,
					"name":            "Medium",
					"sender_pattern":  "*@medium.com",
				}
				w = ts.Request(t, "POST", "/v1/newsletter-filters", body2)
				apitest.AssertStatus(t, w, http.StatusOK)
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

			w := ts.Request(t, "GET", "/v1/newsletter-filters", nil)

			apitest.AssertStatus(t, w, tt.wantStatus)

			var resp struct {
				Filters []struct {
					ID             string `json:"id"`
					Name           string `json:"name"`
					SenderPattern  string `json:"sender_pattern"`
					SubjectPattern string `json:"subject_pattern"`
					IsActive       bool   `json:"is_active"`
				} `json:"filters"`
			}
			apitest.DecodeResponse(t, w, &resp)

			if len(resp.Filters) != tt.wantCount {
				t.Errorf("Expected %d filters, got %d", tt.wantCount, len(resp.Filters))
			}
		})
	}
}

func TestGetNewsletterFilter(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*apitest.TestServer) string
		wantStatus int
	}{
		{
			name: "existing filter",
			setup: func(ts *apitest.TestServer) string {
				sourceID := createTestEmailSource(t, ts)
				body := map[string]any{
					"email_source_id": sourceID,
					"name":            "Test Filter",
					"sender_pattern":  "*@test.com",
				}
				w := ts.Request(t, "POST", "/v1/newsletter-filters", body)
				apitest.AssertStatus(t, w, http.StatusOK)

				var resp struct {
					ID string `json:"id"`
				}
				apitest.DecodeResponse(t, w, &resp)
				return resp.ID
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "non-existent filter",
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

			filterID := tt.setup(ts)

			w := ts.Request(t, "GET", "/v1/newsletter-filters/"+filterID, nil)

			apitest.AssertStatus(t, w, tt.wantStatus)

			if tt.wantStatus == http.StatusOK {
				var resp struct {
					ID            string `json:"id"`
					Name          string `json:"name"`
					SenderPattern string `json:"sender_pattern"`
				}
				apitest.DecodeResponse(t, w, &resp)

				if resp.ID != filterID {
					t.Errorf("Expected ID %s, got %s", filterID, resp.ID)
				}
			}
		})
	}
}

func TestUpdateNewsletterFilter(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*apitest.TestServer) string
		body       map[string]any
		wantStatus int
		validate   func(*testing.T, map[string]any)
	}{
		{
			name: "update name",
			setup: func(ts *apitest.TestServer) string {
				return createTestFilter(t, ts, "Original Name", "*@test.com")
			},
			body: map[string]any{
				"name": "Updated Name",
			},
			wantStatus: http.StatusOK,
			validate: func(t *testing.T, resp map[string]any) {
				if resp["name"] != "Updated Name" {
					t.Errorf("Expected name 'Updated Name', got %v", resp["name"])
				}
			},
		},
		{
			name: "update sender pattern",
			setup: func(ts *apitest.TestServer) string {
				return createTestFilter(t, ts, "Test", "*@old.com")
			},
			body: map[string]any{
				"sender_pattern": "*@new.com",
			},
			wantStatus: http.StatusOK,
			validate: func(t *testing.T, resp map[string]any) {
				if resp["sender_pattern"] != "*@new.com" {
					t.Errorf("Expected sender_pattern '*@new.com', got %v", resp["sender_pattern"])
				}
			},
		},
		{
			name: "deactivate filter",
			setup: func(ts *apitest.TestServer) string {
				return createTestFilter(t, ts, "Test", "*@test.com")
			},
			body: map[string]any{
				"is_active": false,
			},
			wantStatus: http.StatusOK,
			validate: func(t *testing.T, resp map[string]any) {
				if resp["is_active"] != false {
					t.Error("Expected filter to be inactive")
				}
			},
		},
		{
			name: "add subject pattern",
			setup: func(ts *apitest.TestServer) string {
				return createTestFilter(t, ts, "Test", "*@test.com")
			},
			body: map[string]any{
				"subject_pattern": "^Weekly",
			},
			wantStatus: http.StatusOK,
			validate: func(t *testing.T, resp map[string]any) {
				if resp["subject_pattern"] != "^Weekly" {
					t.Errorf("Expected subject_pattern '^Weekly', got %v", resp["subject_pattern"])
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := apitest.NewTestServer(t)
			defer ts.Close(t)

			filterID := tt.setup(ts)

			w := ts.Request(t, "PUT", "/v1/newsletter-filters/"+filterID, tt.body)

			apitest.AssertStatus(t, w, tt.wantStatus)

			if tt.wantStatus == http.StatusOK {
				var resp map[string]any
				apitest.DecodeResponse(t, w, &resp)

				if tt.validate != nil {
					tt.validate(t, resp)
				}
			}
		})
	}
}

func TestDeleteNewsletterFilter(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(*apitest.TestServer) string
		wantStatus int
	}{
		{
			name: "delete existing filter",
			setup: func(ts *apitest.TestServer) string {
				return createTestFilter(t, ts, "Test Filter", "*@test.com")
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "delete non-existent filter",
			setup: func(ts *apitest.TestServer) string {
				return uuid.New().String()
			},
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := apitest.NewTestServer(t)
			defer ts.Close(t)

			filterID := tt.setup(ts)

			w := ts.Request(t, "DELETE", "/v1/newsletter-filters/"+filterID, nil)

			apitest.AssertStatus(t, w, tt.wantStatus)

			// Verify deletion
			if tt.wantStatus == http.StatusNoContent {
				w = ts.Request(t, "GET", "/v1/newsletter-filters/"+filterID, nil)
				apitest.AssertStatus(t, w, http.StatusNotFound)
			}
		})
	}
}

// Helper functions
func createTestEmailSource(t *testing.T, ts *apitest.TestServer) string {
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
}

func createTestFilter(t *testing.T, ts *apitest.TestServer, name, senderPattern string) string {
	sourceID := createTestEmailSource(t, ts)

	body := map[string]any{
		"email_source_id": sourceID,
		"name":            name,
		"sender_pattern":  senderPattern,
	}

	w := ts.Request(t, "POST", "/v1/newsletter-filters", body)
	apitest.AssertStatus(t, w, http.StatusOK)

	var resp struct {
		ID string `json:"id"`
	}
	apitest.DecodeResponse(t, w, &resp)
	return resp.ID
}
