package sources

import (
	"strings"
	"testing"
)

func TestParseGitHubTrendingNormalizesRepoPathFromHref(t *testing.T) {
	payload := `
	<article class="Box-row">
	  <h2>
	    <a href="/palmier-io/palmier-pro">
	      palmier-io/
	      palmier-pro
	    </a>
	  </h2>
	  <p>macOS video editor built for AI</p>
	  <span itemprop="programmingLanguage">Swift</span>
	  <a href="/palmier-io/palmier-pro/stargazers">2,142</a>
	  <a href="/palmier-io/palmier-pro/forks">207</a>
	  <span>756 stars today</span>
	</article>`

	items, err := parseGitHubTrending(strings.NewReader(payload))
	if err != nil {
		t.Fatalf("parse github payload: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}

	item := items[0]
	if item.ExternalID != "palmier-io/palmier-pro" {
		t.Fatalf("unexpected external id: %q", item.ExternalID)
	}
	if item.Title != "palmier-io/palmier-pro" {
		t.Fatalf("unexpected title: %q", item.Title)
	}
	if item.URL != "https://github.com/palmier-io/palmier-pro" {
		t.Fatalf("unexpected url: %q", item.URL)
	}
	if item.Summary == nil || *item.Summary != "macOS video editor built for AI" {
		t.Fatalf("unexpected summary: %#v", item.Summary)
	}
	if item.Score == nil || *item.Score != 756 {
		t.Fatalf("unexpected score: %#v", item.Score)
	}
	if got, ok := item.Metadata["total_stars"].(int); !ok || got != 2142 {
		t.Fatalf("unexpected total stars: %#v", item.Metadata["total_stars"])
	}
	if got, ok := item.Metadata["forks"].(int); !ok || got != 207 {
		t.Fatalf("unexpected forks: %#v", item.Metadata["forks"])
	}
	if got, ok := item.Metadata["language"].(string); !ok || got != "Swift" {
		t.Fatalf("unexpected language: %#v", item.Metadata["language"])
	}
}
