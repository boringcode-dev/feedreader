package service

import (
	"strings"
	"testing"
	"time"

	"feedreader/internal/domain"
)

func TestBuildCardsIncludesHackerNewsPublishedDateParts(t *testing.T) {
	points := 50
	summary := "ignored"
	publishedAt := time.Date(2026, time.June, 20, 0, 0, 0, 0, time.UTC)
	items := []domain.FeedItem{{
		Source:      "hackernews",
		Title:       "story",
		URL:         "https://example.com/story",
		Summary:     &summary,
		Score:       &points,
		PublishedAt: &publishedAt,
		SourceRank:  1,
		Metadata: map[string]any{
			"comments_count": 24,
		},
	}}

	cards := BuildCards(items, 0)
	if len(cards) != 1 || cards[0].Brief == nil {
		t.Fatalf("expected one card with brief, got %#v", cards)
	}
	if cards[0].BriefDateKind != "published" || cards[0].BriefDateISO == nil {
		t.Fatalf("expected published date parts, got %#v", cards[0])
	}
	if cards[0].BriefPrefix == nil || *cards[0].BriefPrefix != "50 points · 24 comments" {
		t.Fatalf("unexpected brief prefix: %#v", cards[0].BriefPrefix)
	}
	if cards[0].BriefSuffix != nil {
		t.Fatalf("expected no brief suffix, got %#v", cards[0].BriefSuffix)
	}
	brief := *cards[0].Brief
	if brief != "50 points · 24 comments · Published Jun 20, 2026" {
		t.Fatalf("unexpected brief: %q", brief)
	}
}

func TestBuildCardsIncludesGitHubFetchedDateParts(t *testing.T) {
	summary := "High-performance code intelligence MCP server."
	fetchedAt := time.Date(2026, time.June, 18, 10, 0, 0, 0, time.UTC)
	items := []domain.FeedItem{{
		Source:     "github",
		Title:      "owner/repo",
		URL:        "https://github.com/owner/repo",
		Summary:    &summary,
		FetchedAt:  &fetchedAt,
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
	if cards[0].BriefDateKind != "fetched" || cards[0].BriefDateISO == nil {
		t.Fatalf("expected fetched date parts, got %#v", cards[0])
	}
	if cards[0].BriefPrefix == nil {
		t.Fatalf("expected brief prefix, got %#v", cards[0])
	}
	for _, want := range []string{"4,774 stars", "718 today", "321 forks"} {
		if !strings.Contains(*cards[0].BriefPrefix, want) {
			t.Fatalf("brief prefix %q missing %q", *cards[0].BriefPrefix, want)
		}
	}
	if cards[0].BriefSuffix == nil || *cards[0].BriefSuffix != summary {
		t.Fatalf("unexpected brief suffix: %#v", cards[0].BriefSuffix)
	}
	brief := *cards[0].Brief
	for _, want := range []string{"4,774 stars", "718 today", "321 forks", "Fetched Jun 18, 2026", summary} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief %q missing %q", brief, want)
		}
	}
}

func TestBuildCardsIncludesHuggingFacePublishedDateParts(t *testing.T) {
	summary := "Passive models for long video understanding."
	author := "Jane Doe, John Smith"
	upvotes := 111
	publishedAt := time.Date(2026, time.June, 17, 0, 0, 0, 0, time.UTC)
	items := []domain.FeedItem{{
		Source:      "huggingface",
		Title:       "paper",
		URL:         "https://huggingface.co/papers/123",
		Summary:     &summary,
		Author:      &author,
		Score:       &upvotes,
		PublishedAt: &publishedAt,
		SourceRank:  1,
	}}

	cards := BuildCards(items, 0)
	if len(cards) != 1 || cards[0].Brief == nil {
		t.Fatalf("expected one card with brief, got %#v", cards)
	}
	if cards[0].BriefDateKind != "published" || cards[0].BriefDateISO == nil {
		t.Fatalf("expected published date parts, got %#v", cards[0])
	}
	if cards[0].BriefPrefix == nil {
		t.Fatalf("expected brief prefix, got %#v", cards[0])
	}
	for _, want := range []string{"111 upvotes"} {
		if !strings.Contains(*cards[0].BriefPrefix, want) {
			t.Fatalf("brief prefix %q missing %q", *cards[0].BriefPrefix, want)
		}
	}
	if cards[0].BriefSuffix == nil || *cards[0].BriefSuffix != summary {
		t.Fatalf("unexpected brief suffix: %#v", cards[0].BriefSuffix)
	}
	brief := *cards[0].Brief
	for _, want := range []string{"111 upvotes", "Published Jun 17, 2026", summary} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief %q missing %q", brief, want)
		}
	}
}

func TestBuildCardsIncludesAlphaXivPublishedDateParts(t *testing.T) {
	summary := "A social reading layer for research papers."
	author := "Jane Doe, John Smith"
	likes := 86
	publishedAt := time.Date(2026, time.June, 14, 0, 0, 0, 0, time.UTC)
	items := []domain.FeedItem{{
		Source:      "alphaxiv",
		Title:       "paper",
		URL:         "https://www.alphaxiv.org/abs/2606.15956",
		Summary:     &summary,
		Author:      &author,
		Score:       &likes,
		PublishedAt: &publishedAt,
		SourceRank:  1,
	}}

	cards := BuildCards(items, 0)
	if len(cards) != 1 || cards[0].Brief == nil {
		t.Fatalf("expected one card with brief, got %#v", cards)
	}
	if cards[0].BriefDateKind != "published" || cards[0].BriefDateISO == nil {
		t.Fatalf("expected published date parts, got %#v", cards[0])
	}
	if cards[0].BriefPrefix == nil {
		t.Fatalf("expected brief prefix, got %#v", cards[0])
	}
	for _, want := range []string{"86 likes"} {
		if !strings.Contains(*cards[0].BriefPrefix, want) {
			t.Fatalf("brief prefix %q missing %q", *cards[0].BriefPrefix, want)
		}
	}
	if cards[0].BriefSuffix == nil || *cards[0].BriefSuffix != summary {
		t.Fatalf("unexpected brief suffix: %#v", cards[0].BriefSuffix)
	}
	brief := *cards[0].Brief
	for _, want := range []string{"86 likes", "Published Jun 14, 2026", summary} {
		if !strings.Contains(brief, want) {
			t.Fatalf("brief %q missing %q", brief, want)
		}
	}
}

func TestBuildCardsWithoutStatsStillCarriesDateParts(t *testing.T) {
	summary := "A plain article summary"
	publishedAt := time.Date(2026, time.June, 12, 0, 0, 0, 0, time.UTC)
	items := []domain.FeedItem{{
		Source:      "custom",
		Title:       "plain",
		URL:         "https://example.com/plain",
		Summary:     &summary,
		PublishedAt: &publishedAt,
	}}

	cards := BuildCards(items, 0)
	if len(cards) != 1 || cards[0].Brief == nil {
		t.Fatalf("expected one card with brief, got %#v", cards)
	}
	if cards[0].BriefPrefix != nil {
		t.Fatalf("expected no brief prefix, got %#v", cards[0].BriefPrefix)
	}
	if cards[0].BriefDateKind != "published" || cards[0].BriefDateISO == nil {
		t.Fatalf("expected published date parts, got %#v", cards[0])
	}
	if cards[0].BriefSuffix == nil || *cards[0].BriefSuffix != summary {
		t.Fatalf("expected brief suffix %q, got %#v", summary, cards[0].BriefSuffix)
	}
	if *cards[0].Brief != "Published Jun 12, 2026 - A plain article summary" {
		t.Fatalf("unexpected brief: %q", *cards[0].Brief)
	}
}
