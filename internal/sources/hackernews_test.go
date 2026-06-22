package sources

import (
	"testing"
	"time"
)

func TestParseHackerNewsFrontPageJSON(t *testing.T) {
	payload := []byte(`{
	  "hits": [
	    {
	      "objectID": "123",
	      "title": "Example story",
	      "url": "https://example.com/story",
	      "author": "alice",
	      "points": 42,
	      "num_comments": 11,
	      "created_at": "2026-06-21T12:44:13Z",
	      "story_text": "<p>Example &amp; summary.</p>"
	    },
	    {
	      "objectID": "456",
	      "title": "Ask HN: Fallback URL",
	      "author": "bob",
	      "points": 7,
	      "num_comments": 3,
	      "created_at": "2026-06-21T13:00:00Z"
	    }
	  ]
	}`)

	items, err := parseHackerNews(payload)
	if err != nil {
		t.Fatalf("parse hacker news payload: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 items, got %d", len(items))
	}

	first := items[0]
	if first.Source != "hackernews" {
		t.Fatalf("unexpected source: %q", first.Source)
	}
	if first.ExternalID != "123" {
		t.Fatalf("unexpected external id: %q", first.ExternalID)
	}
	if first.Title != "Example story" {
		t.Fatalf("unexpected title: %q", first.Title)
	}
	if first.URL != "https://example.com/story" {
		t.Fatalf("unexpected url: %q", first.URL)
	}
	if first.Author == nil || *first.Author != "alice" {
		t.Fatalf("unexpected author: %#v", first.Author)
	}
	if first.Score == nil || *first.Score != 42 {
		t.Fatalf("unexpected score: %#v", first.Score)
	}
	if first.Summary == nil || *first.Summary != "Example & summary." {
		t.Fatalf("unexpected summary: %#v", first.Summary)
	}
	if first.CommentsURL == nil || *first.CommentsURL != "https://news.ycombinator.com/item?id=123" {
		t.Fatalf("unexpected comments url: %#v", first.CommentsURL)
	}
	if got, ok := first.Metadata["comments_count"].(int); !ok || got != 11 {
		t.Fatalf("unexpected comments_count metadata: %#v", first.Metadata["comments_count"])
	}
	wantPublishedAt := time.Date(2026, time.June, 21, 12, 44, 13, 0, time.UTC)
	if first.PublishedAt == nil || !first.PublishedAt.Equal(wantPublishedAt) {
		t.Fatalf("unexpected publishedAt: %#v", first.PublishedAt)
	}
	if first.SourceRank != 1 {
		t.Fatalf("unexpected source rank: %d", first.SourceRank)
	}

	second := items[1]
	if second.URL != "https://news.ycombinator.com/item?id=456" {
		t.Fatalf("expected comments-url fallback, got %q", second.URL)
	}
	if second.CommentsURL == nil || *second.CommentsURL != "https://news.ycombinator.com/item?id=456" {
		t.Fatalf("unexpected fallback comments url: %#v", second.CommentsURL)
	}
}
