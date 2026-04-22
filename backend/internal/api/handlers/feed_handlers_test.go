package handlers_test

import (
	"net/http"
	"testing"

	apitest "github.com/bbu/rss-summarizer/backend/internal/api/testing"
)

func TestCreateFeed(t *testing.T) {
	tests := []struct {
		name       string
		body       map[string]any
		wantStatus int
		wantErr    bool
	}{
		{
			name: "valid feed",
			body: map[string]any{
				"url":                    "https://blog.golang.org/feed.atom",
				"poll_frequency_minutes": 60,
			},
			wantStatus: http.StatusOK,
			wantErr:    false,
		},
		{
			name: "invalid url",
			body: map[string]any{
				"url":                    "not-a-valid-url",
				"poll_frequency_minutes": 60,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name: "poll frequency too low",
			body: map[string]any{
				"url":                    "https://blog.golang.org/feed.atom",
				"poll_frequency_minutes": 5,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name: "poll frequency too high",
			body: map[string]any{
				"url":                    "https://blog.golang.org/feed.atom",
				"poll_frequency_minutes": 2000,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
		{
			name: "missing url",
			body: map[string]any{
				"poll_frequency_minutes": 60,
			},
			wantStatus: http.StatusUnprocessableEntity,
			wantErr:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ts := apitest.NewTestServer(t)
			defer ts.Close(t)

			w := ts.Request(t, "POST", "/v1/feeds", tt.body)

			apitest.AssertStatus(t, w, tt.wantStatus)

			if !tt.wantErr {
				var resp struct {
					ID                   string `json:"id"`
					URL                  string `json:"url"`
					PollFrequencyMinutes int    `json:"poll_frequency_minutes"`
				}
				apitest.DecodeResponse(t, w, &resp)

				if resp.ID == "" {
					t.Error("Expected feed ID to be set")
				}
				if resp.URL != tt.body["url"] {
					t.Errorf("Expected URL %s, got %s", tt.body["url"], resp.URL)
				}
			}
		})
	}
}

func TestListFeeds(t *testing.T) {
	ts := apitest.NewTestServer(t)
	defer ts.Close(t)

	// Initially should be empty
	w := ts.Request(t, "GET", "/v1/feeds", nil)
	apitest.AssertStatus(t, w, http.StatusOK)

	var resp struct {
		Feeds      []map[string]any `json:"feeds"`
		TotalCount int              `json:"total_count"`
		Limit      int              `json:"limit"`
		Offset     int              `json:"offset"`
	}
	apitest.DecodeResponse(t, w, &resp)

	if len(resp.Feeds) != 0 {
		t.Errorf("Expected 0 feeds, got %d", len(resp.Feeds))
	}

	// Create a feed
	ts.Request(t, "POST", "/v1/feeds", map[string]any{
		"url":                    "https://blog.golang.org/feed.atom",
		"poll_frequency_minutes": 60,
	})

	// Should now have 1 feed
	w = ts.Request(t, "GET", "/v1/feeds", nil)
	apitest.AssertStatus(t, w, http.StatusOK)
	apitest.DecodeResponse(t, w, &resp)

	if len(resp.Feeds) != 1 {
		t.Errorf("Expected 1 feed, got %d", len(resp.Feeds))
	}
}

func TestGetFeed(t *testing.T) {
	ts := apitest.NewTestServer(t)
	defer ts.Close(t)

	// Create a feed
	w := ts.Request(t, "POST", "/v1/feeds", map[string]any{
		"url":                    "https://blog.golang.org/feed.atom",
		"poll_frequency_minutes": 60,
	})

	var created struct {
		ID string `json:"id"`
	}
	apitest.DecodeResponse(t, w, &created)

	// Get the feed
	w = ts.Request(t, "GET", "/v1/feeds/"+created.ID, nil)
	apitest.AssertStatus(t, w, http.StatusOK)

	var feed struct {
		ID  string `json:"id"`
		URL string `json:"url"`
	}
	apitest.DecodeResponse(t, w, &feed)

	if feed.ID != created.ID {
		t.Errorf("Expected feed ID %s, got %s", created.ID, feed.ID)
	}
	if feed.URL != "https://blog.golang.org/feed.atom" {
		t.Errorf("Expected URL https://blog.golang.org/feed.atom, got %s", feed.URL)
	}
}

func TestGetFeedNotFound(t *testing.T) {
	ts := apitest.NewTestServer(t)
	defer ts.Close(t)

	w := ts.Request(t, "GET", "/v1/feeds/00000000-0000-0000-0000-000000000000", nil)
	apitest.AssertStatus(t, w, http.StatusNotFound)
}

func TestUpdateFeed(t *testing.T) {
	ts := apitest.NewTestServer(t)
	defer ts.Close(t)

	// Create a feed
	w := ts.Request(t, "POST", "/v1/feeds", map[string]any{
		"url":                    "https://blog.golang.org/feed.atom",
		"poll_frequency_minutes": 60,
	})

	var created struct {
		ID string `json:"id"`
	}
	apitest.DecodeResponse(t, w, &created)

	// Update the feed
	w = ts.Request(t, "PUT", "/v1/feeds/"+created.ID, map[string]any{
		"poll_frequency_minutes": 120,
	})
	apitest.AssertStatus(t, w, http.StatusOK)

	var updated struct {
		PollFrequencyMinutes int `json:"poll_frequency_minutes"`
	}
	apitest.DecodeResponse(t, w, &updated)

	if updated.PollFrequencyMinutes != 120 {
		t.Errorf("Expected poll frequency 120, got %d", updated.PollFrequencyMinutes)
	}
}

func TestDeleteFeed(t *testing.T) {
	ts := apitest.NewTestServer(t)
	defer ts.Close(t)

	// Create a feed
	w := ts.Request(t, "POST", "/v1/feeds", map[string]any{
		"url":                    "https://blog.golang.org/feed.atom",
		"poll_frequency_minutes": 60,
	})

	var created struct {
		ID string `json:"id"`
	}
	apitest.DecodeResponse(t, w, &created)

	// Delete the feed
	w = ts.Request(t, "DELETE", "/v1/feeds/"+created.ID, nil)
	apitest.AssertStatus(t, w, http.StatusNoContent)

	// Verify it's deleted
	w = ts.Request(t, "GET", "/v1/feeds/"+created.ID, nil)
	apitest.AssertStatus(t, w, http.StatusNotFound)
}

func TestDuplicateFeedURL(t *testing.T) {
	ts := apitest.NewTestServer(t)
	defer ts.Close(t)

	feedURL := "https://blog.golang.org/feed.atom"

	// Create first feed
	w := ts.Request(t, "POST", "/v1/feeds", map[string]any{
		"url":                    feedURL,
		"poll_frequency_minutes": 60,
	})
	apitest.AssertStatus(t, w, http.StatusOK)

	// Try to create duplicate
	w = ts.Request(t, "POST", "/v1/feeds", map[string]any{
		"url":                    feedURL,
		"poll_frequency_minutes": 60,
	})
	// Should fail with conflict or validation error
	if w.Code == http.StatusOK {
		t.Error("Expected duplicate feed creation to fail")
	}
}
