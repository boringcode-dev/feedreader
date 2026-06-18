package service

import (
	"strings"
	"testing"

	"feedreader/internal/domain"
)

func TestBuildCardsIncludesHackerNewsCountsWithNormalizedSeparator(t *testing.T) {
	points := 50
	summary := "ignored"
	items := []domain.FeedItem{{
		Source:     "hackernews",
		Title:      "story",
		URL:        "https://example.com/story",
		Summary:    &summary,
		Score:      &points,
		SourceRank: 1,
		Metadata: map[string]any{
			"comments_count": 24,
		},
	}}

	cards := BuildCards(items, 0)
	if len(cards) != 1 || cards[0].Brief == nil {
		t.Fatalf("expected one card with brief, got %#v", cards)
	}
	brief := *cards[0].Brief
	if brief != "50 points · 24 comments" {
		t.Fatalf("unexpected brief: %q", brief)
	}
}

func TestBuildCardsIncludesGitHubCountsInBrief(t *testing.T) {
	summary := "High-performance code intelligence MCP server."
	items := []domain.FeedItem{{
		Source:     "github",
		Title:      "owner/repo",
		URL:        "https://github.com/owner/repo",
		Summary:    &summary,
		SourceRank: 1,
		Metadata: map[string]any{
			"total_stars": 4774,
			"stars_today": 718,
			"forks":       321,
		},
	}}

	cards := BuildCards(items, 0)
	if len(cards) != 1 || cards[0].Brief == nil {
		t.Fatalf("expected one card with brief, got %#v", cards)
	}
	brief := *cards[0].Brief
	for _, want := range []string{"4,774 stars", "718 today", "321 forks", summary} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief %q missing %q", brief, want)
		}
	}
}

func TestBuildCardsIncludesHuggingFaceUpvotesInBrief(t *testing.T) {
	summary := "Passive models for long video understanding."
	author := "Jane Doe, John Smith"
	upvotes := 111
	items := []domain.FeedItem{{
		Source:     "huggingface",
		Title:      "paper",
		URL:        "https://huggingface.co/papers/123",
		Summary:    &summary,
		Author:     &author,
		Score:      &upvotes,
		SourceRank: 1,
	}}

	cards := BuildCards(items, 0)
	if len(cards) != 1 || cards[0].Brief == nil {
		t.Fatalf("expected one card with brief, got %#v", cards)
	}
	brief := *cards[0].Brief
	for _, want := range []string{"111 upvotes", summary} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief %q missing %q", brief, want)
		}
	}
}

func TestBuildCardsIncludesAlphaXivLikesInBrief(t *testing.T) {
	summary := "A social reading layer for research papers."
	author := "Jane Doe, John Smith"
	likes := 86
	items := []domain.FeedItem{{
		Source:     "alphaxiv",
		Title:      "paper",
		URL:        "https://www.alphaxiv.org/abs/2606.15956",
		Summary:    &summary,
		Author:     &author,
		Score:      &likes,
		SourceRank: 1,
	}}

	cards := BuildCards(items, 0)
	if len(cards) != 1 || cards[0].Brief == nil {
		t.Fatalf("expected one card with brief, got %#v", cards)
	}
	brief := *cards[0].Brief
	for _, want := range []string{"86 likes", summary} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief %q missing %q", brief, want)
		}
	}
}
