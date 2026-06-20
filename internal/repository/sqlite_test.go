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

	vectorItems, err := repo.ListFeedItems(10, 0, "", nil, "vector")
	if err != nil {
		t.Fatalf("search vector: %v", err)
	}
	if len(vectorItems) != 2 {
		t.Fatalf("expected 2 vector matches, got %d", len(vectorItems))
	}
	if vectorItems[0].ExternalID != "paper-1" || vectorItems[1].ExternalID != "owner/graph-scout" {
		t.Fatalf("unexpected vector ordering: %#v", []string{vectorItems[0].ExternalID, vectorItems[1].ExternalID})
	}

	selectedSourcesItems, err := repo.ListFeedItems(10, 0, "", []string{"github", "huggingface"}, "")
	if err != nil {
		t.Fatalf("selected sources: %v", err)
	}
	if len(selectedSourcesItems) != 3 {
		t.Fatalf("expected 3 selected-source items, got %d", len(selectedSourcesItems))
	}
	if selectedSourcesItems[0].ExternalID != "paper-1" || selectedSourcesItems[1].ExternalID != "owner/graph-scout" || selectedSourcesItems[2].ExternalID != "owner/search-agent" {
		t.Fatalf("unexpected selected-source ordering: %#v", []string{selectedSourcesItems[0].ExternalID, selectedSourcesItems[1].ExternalID, selectedSourcesItems[2].ExternalID})
	}

	githubVectorItems, err := repo.ListFeedItems(10, 0, "github", nil, "vector")
	if err != nil {
		t.Fatalf("search github vector: %v", err)
	}
	if len(githubVectorItems) != 1 || githubVectorItems[0].ExternalID != "owner/graph-scout" {
		t.Fatalf("unexpected github vector results: %#v", githubVectorItems)
	}

	authorItems, err := repo.ListFeedItems(10, 0, "", nil, "jane")
	if err != nil {
		t.Fatalf("search author: %v", err)
	}
	if len(authorItems) != 1 || authorItems[0].ExternalID != "paper-1" {
		t.Fatalf("unexpected author results: %#v", authorItems)
	}

	pagedItems, err := repo.ListFeedItems(1, 1, "", nil, "vector")
	if err != nil {
		t.Fatalf("search vector page 2: %v", err)
	}
	if len(pagedItems) != 1 || pagedItems[0].ExternalID != "owner/graph-scout" {
		t.Fatalf("unexpected paged results: %#v", pagedItems)
	}

	multiTermItems, err := repo.ListFeedItems(10, 0, "", nil, "vector research")
	if err != nil {
		t.Fatalf("search vector research: %v", err)
	}
	if len(multiTermItems) != 1 || multiTermItems[0].ExternalID != "paper-1" {
		t.Fatalf("unexpected multi-term results: %#v", multiTermItems)
	}
}

func TestSaveSnapshotPreservesInitialDatesAndUsesFirstSeenFallbackOrdering(t *testing.T) {
	database, err := dbpkg.Open(filepath.Join(t.TempDir(), "feedreader.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repo := NewSQLiteRepository(database)
	firstFetch := time.Date(2026, 6, 18, 10, 0, 0, 0, time.UTC)
	secondFetch := time.Date(2026, 6, 18, 11, 0, 0, 0, time.UTC)
	thirdFetch := time.Date(2026, 6, 18, 12, 0, 0, 0, time.UTC)
	publishedOriginal := time.Date(2026, 6, 18, 9, 0, 0, 0, time.UTC)
	publishedReplacement := time.Date(2026, 6, 18, 13, 0, 0, 0, time.UTC)

	if err := repo.SaveSnapshot("github", firstFetch, []domain.FeedItem{{
		Source:     "github",
		ExternalID: "item-a",
		Title:      "Item A",
		URL:        "https://github.com/example/item-a",
		SourceRank: 1,
		Metadata:   map[string]any{},
	}}); err != nil {
		t.Fatalf("save item-a first snapshot: %v", err)
	}

	if err := repo.SaveSnapshot("github", secondFetch, []domain.FeedItem{{
		Source:     "github",
		ExternalID: "item-b",
		Title:      "Item B",
		URL:        "https://github.com/example/item-b",
		SourceRank: 1,
		Metadata:   map[string]any{},
	}}); err != nil {
		t.Fatalf("save item-b snapshot: %v", err)
	}

	updatedTitle := "Item A updated"
	if err := repo.SaveSnapshot("github", thirdFetch, []domain.FeedItem{{
		Source:     "github",
		ExternalID: "item-a",
		Title:      updatedTitle,
		URL:        "https://github.com/example/item-a",
		SourceRank: 1,
		Metadata:   map[string]any{},
	}}); err != nil {
		t.Fatalf("save item-a second snapshot: %v", err)
	}

	if err := repo.SaveSnapshot("alphaxiv", firstFetch, []domain.FeedItem{{
		Source:      "alphaxiv",
		ExternalID:  "paper-1",
		Title:       "Paper 1",
		URL:         "https://www.alphaxiv.org/abs/paper-1",
		PublishedAt: &publishedOriginal,
		SourceRank:  1,
		Metadata:    map[string]any{},
	}}); err != nil {
		t.Fatalf("save paper-1 first snapshot: %v", err)
	}

	if err := repo.SaveSnapshot("alphaxiv", thirdFetch, []domain.FeedItem{{
		Source:      "alphaxiv",
		ExternalID:  "paper-1",
		Title:       "Paper 1 updated",
		URL:         "https://www.alphaxiv.org/abs/paper-1",
		PublishedAt: &publishedReplacement,
		SourceRank:  1,
		Metadata:    map[string]any{},
	}}); err != nil {
		t.Fatalf("save paper-1 second snapshot: %v", err)
	}

	items, err := repo.ListFeedItems(10, 0, "", []string{"github"}, "")
	if err != nil {
		t.Fatalf("list github items: %v", err)
	}
	if len(items) != 2 {
		t.Fatalf("expected 2 github items, got %d", len(items))
	}
	if items[0].ExternalID != "item-b" || items[1].ExternalID != "item-a" {
		t.Fatalf("expected first-seen fallback ordering to keep newer item-b ahead of refreshed item-a, got %#v", []string{items[0].ExternalID, items[1].ExternalID})
	}

	var publishedAt, firstSeenAt, updatedAt string
	if err := repo.db.QueryRow(
		`SELECT published_at, first_seen_at, updated_at FROM items WHERE source = ? AND external_id = ?`,
		"alphaxiv", "paper-1",
	).Scan(&publishedAt, &firstSeenAt, &updatedAt); err != nil {
		t.Fatalf("query preserved dates: %v", err)
	}

	wantPublished := publishedOriginal.UTC().Format(time.RFC3339Nano)
	wantFetched := firstFetch.UTC().Format(time.RFC3339Nano)
	if publishedAt != wantPublished {
		t.Fatalf("expected published_at %s, got %s", wantPublished, publishedAt)
	}
	if firstSeenAt != wantFetched {
		t.Fatalf("expected first_seen_at %s, got %s", wantFetched, firstSeenAt)
	}
	if updatedAt != wantFetched {
		t.Fatalf("expected updated_at to preserve initial fetched timestamp %s, got %s", wantFetched, updatedAt)
	}
}

func TestNormalizeGitHubItemsCanonicalizesLegacyRepoIdentity(t *testing.T) {
	database, err := dbpkg.Open(filepath.Join(t.TempDir(), "feedreader.db"))
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	repo := NewSQLiteRepository(database)
	fetchedAt := time.Date(2026, 6, 20, 8, 0, 0, 0, time.UTC)
	legacyExternalID := "\n\n\n\n\npalmier-io/\n\npalmier-pro"
	legacyTitle := "\n\n\n\n\npalmier-io/\n\npalmier-pro"

	if err := repo.SaveSnapshot("github", fetchedAt, []domain.FeedItem{{
		Source:     "github",
		ExternalID: legacyExternalID,
		Title:      legacyTitle,
		URL:        "https://github.com/palmier-io/palmier-pro",
		SourceRank: 1,
		Metadata:   map[string]any{},
	}}); err != nil {
		t.Fatalf("save legacy github snapshot: %v", err)
	}

	if err := repo.NormalizeGitHubItems(); err != nil {
		t.Fatalf("normalize github items: %v", err)
	}

	var count int
	if err := repo.db.QueryRow(`SELECT COUNT(*) FROM items WHERE source = 'github'`).Scan(&count); err != nil {
		t.Fatalf("count github items: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 github row after normalization, got %d", count)
	}

	var externalID, title string
	if err := repo.db.QueryRow(`SELECT external_id, title FROM items WHERE source = 'github' LIMIT 1`).Scan(&externalID, &title); err != nil {
		t.Fatalf("query normalized github row: %v", err)
	}
	if externalID != "palmier-io/palmier-pro" {
		t.Fatalf("unexpected normalized external_id: %q", externalID)
	}
	if title != "palmier-io/palmier-pro" {
		t.Fatalf("unexpected normalized title: %q", title)
	}
}
