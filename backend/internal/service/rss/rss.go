package rss

import (
	"context"
	"fmt"
	"time"

	"github.com/mmcdole/gofeed"
)

type FeedMetadata struct {
	Title       string
	Description string
	Items       []FeedItem
}

type FeedItem struct {
	Title       string
	URL         string
	Content     string
	PublishedAt *time.Time
}

type Service interface {
	FetchFeed(ctx context.Context, url string) (*FeedMetadata, error)
	ValidateFeedURL(ctx context.Context, url string) error
}

type service struct {
	parser *gofeed.Parser
}

func NewService() Service {
	return &service{
		parser: gofeed.NewParser(),
	}
}

func (s *service) FetchFeed(ctx context.Context, url string) (*FeedMetadata, error) {
	feed, err := s.parser.ParseURLWithContext(url, ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to parse feed: %w", err)
	}

	metadata := &FeedMetadata{
		Title:       feed.Title,
		Description: feed.Description,
		Items:       make([]FeedItem, 0, len(feed.Items)),
	}

	for _, item := range feed.Items {
		feedItem := FeedItem{
			Title: item.Title,
			URL:   item.Link,
		}

		// Prefer content over description
		if item.Content != "" {
			feedItem.Content = item.Content
		} else if item.Description != "" {
			feedItem.Content = item.Description
		}

		// Parse published date
		if item.PublishedParsed != nil {
			feedItem.PublishedAt = item.PublishedParsed
		} else if item.UpdatedParsed != nil {
			feedItem.PublishedAt = item.UpdatedParsed
		}

		metadata.Items = append(metadata.Items, feedItem)
	}

	return metadata, nil
}

func (s *service) ValidateFeedURL(ctx context.Context, url string) error {
	_, err := s.parser.ParseURLWithContext(url, ctx)
	if err != nil {
		return fmt.Errorf("invalid feed URL: %w", err)
	}
	return nil
}
