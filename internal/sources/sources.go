package sources

import (
	"context"
	"net/http"

	"feedreader/internal/domain"
)

type Source interface {
	Key() string
	Label() string
	HomePageURL() string
	Fetch(ctx context.Context, client *http.Client) ([]domain.FeedItem, error)
}

func Build() []Source {
	return []Source{
		HackerNewsSource{},
		GitHubTrendingSource{},
		HuggingFacePapersSource{},
	}
}
