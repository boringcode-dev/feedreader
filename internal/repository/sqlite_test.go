package repository

import (
	"path/filepath"
	"testing"
	"time"

	dbpkg "feedreader/internal/db"
	"feedreader/internal/domain"
)

func TestListFeedItemsSearchAndPagination(t *testing.T) {
	database, err := dbpkg.Open(filepath.Join(t.TempDir(), "feedreader.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repo := NewSQLiteRepository(database)
	githubFetchedAt := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	hfFetchedAt := time.Date(2026, 6, 18, 11, 0, 0, 0, time.UTC)
	publishedOne := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	publishedTwo := time.Date(2026, 6, 18, 11, 30, 0, 0, time.UTC)
	publishedThree := time.Date(2026, 6, 18, 12, 30, 0, 0, time.UTC)

	summaryOne := "Vector search for local feeds"
	summaryTwo := "Agent workflows for private feeds"
	titleThree := "Vector Agents for Research"
	authorThree := "Jane Doe"

	if err := repo.SaveSnapshot("github", githubFetchedAt, []domain.FeedItem{
		{
			Source:      "github",
			ExternalID:  "owner/graph-scout",
			Title:       "Graph Scout",
			URL:         "https://github.com/owner/graph-scout",
			Summary:     &summaryOne,
			PublishedAt: &publishedOne,
			SourceRank:  1,
			Metadata:    map[string]any{"language": "Go"},
		},
		{
			Source:      "github",
			ExternalID:  "owner/search-agent",
			Title:       "Search Agent",
			URL:         "https://github.com/owner/search-agent",
			Summary:     &summaryTwo,
			PublishedAt: &publishedTwo,
			SourceRank:  2,
			Metadata:    map[string]any{"language": "TypeScript"},
		},
	}); err != nil {
		t.Fatalf("save github snapshot: %v", err)
	}

	if err := repo.SaveSnapshot("huggingface", hfFetchedAt, []domain.FeedItem{
		{
			Source:      "huggingface",
			ExternalID:  "paper-1",
			Title:       titleThree,
			URL:         "https://huggingface.co/papers/paper-1",
			Author:      &authorThree,
			PublishedAt: &publishedThree,
			SourceRank:  1,
			Metadata:    map[string]any{"authors": []string{"Jane Doe"}},
		},
	}); err != nil {
		t.Fatalf("save huggingface snapshot: %v", err)
	}

	vectorItems, err := repo.ListFeedItems(10, 0, "", "vector")
	if err != nil {
		t.Fatalf("search vector: %v", err)
	}
	if len(vectorItems) != 2 {
		t.Fatalf("expected 2 vector matches, got %d", len(vectorItems))
	}
	if vectorItems[0].ExternalID != "paper-1" || vectorItems[1].ExternalID != "owner/graph-scout" {
		t.Fatalf("unexpected vector ordering: %#v", []string{vectorItems[0].ExternalID, vectorItems[1].ExternalID})
	}

	githubVectorItems, err := repo.ListFeedItems(10, 0, "github", "vector")
	if err != nil {
		t.Fatalf("search github vector: %v", err)
	}
	if len(githubVectorItems) != 1 || githubVectorItems[0].ExternalID != "owner/graph-scout" {
		t.Fatalf("unexpected github vector results: %#v", githubVectorItems)
	}

	authorItems, err := repo.ListFeedItems(10, 0, "", "jane")
	if err != nil {
		t.Fatalf("search author: %v", err)
	}
	if len(authorItems) != 1 || authorItems[0].ExternalID != "paper-1" {
		t.Fatalf("unexpected author results: %#v", authorItems)
	}

	pagedItems, err := repo.ListFeedItems(1, 1, "", "vector")
	if err != nil {
		t.Fatalf("search vector page 2: %v", err)
	}
	if len(pagedItems) != 1 || pagedItems[0].ExternalID != "owner/graph-scout" {
		t.Fatalf("unexpected paged results: %#v", pagedItems)
	}

	multiTermItems, err := repo.ListFeedItems(10, 0, "", "vector research")
	if err != nil {
		t.Fatalf("search vector research: %v", err)
	}
	if len(multiTermItems) != 1 || multiTermItems[0].ExternalID != "paper-1" {
		t.Fatalf("unexpected multi-term results: %#v", multiTermItems)
	}
}
