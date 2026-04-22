package handlers_test

import (
	"net/http"
	"testing"

	apitest "github.com/bbu/rss-summarizer/backend/internal/api/testing"
)

func TestGetPreferences(t *testing.T) {
	ts := apitest.NewTestServer(t)
	defer ts.Close(t)

	w := ts.Request(t, "GET", "/v1/user/preferences", nil)
	apitest.AssertStatus(t, w, http.StatusOK)

	var prefs struct {
		ID                  string `json:"id"`
		DefaultPollInterval int    `json:"default_poll_interval"`
		MaxArticlesPerFeed  int    `json:"max_articles_per_feed"`
	}
	apitest.DecodeResponse(t, w, &prefs)

	// Should have default values
	if prefs.DefaultPollInterval == 0 {
		t.Error("Expected default poll interval to be set")
	}
	if prefs.MaxArticlesPerFeed == 0 {
		t.Error("Expected max articles per feed to be set")
	}
}

func TestUpdatePreferences(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantErr    bool
	}{
		{
			name: "valid update",
			body: map[string]any{
				"default_poll_interval": 120,
				"max_articles_per_feed": 50,
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "minimum values",
			body: map[string]any{
				"default_poll_interval": 15,
				"max_articles_per_feed": 1,
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "poll interval too low",
			body: map[string]any{
				"default_poll_interval": 5,
				"max_articles_per_feed": 20,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name: "poll interval too high",
			body: map[string]any{
				"default_poll_interval": 2000,
				"max_articles_per_feed": 20,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name: "max articles too low",
			body: map[string]any{
				"default_poll_interval": 60,
				"max_articles_per_feed": 0,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name: "max articles too high",
			body: map[string]any{
				"default_poll_interval": 60,
				"max_articles_per_feed": 200,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := apitest.NewTestServer(t)
			defer ts.Close(t)

			w := ts.Request(t, "PUT", "/v1/user/preferences", tt.body)
			apitest.AssertStatus(t, w, tt.wantStatus)

			if !tt.wantErr {
				var prefs struct {
					DefaultPollInterval int `json:"default_poll_interval"`
					MaxArticlesPerFeed  int `json:"max_articles_per_feed"`
				}
				apitest.DecodeResponse(t, w, &prefs)

				if prefs.DefaultPollInterval != tt.body["default_poll_interval"] {
					t.Errorf("Expected poll interval %d, got %d",
						tt.body["default_poll_interval"], prefs.DefaultPollInterval)
				}
				if prefs.MaxArticlesPerFeed != tt.body["max_articles_per_feed"] {
					t.Errorf("Expected max articles %d, got %d",
						tt.body["max_articles_per_feed"], prefs.MaxArticlesPerFeed)
				}
			}
		})
	}
}
