package sources

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"feedreader/internal/domain"
	"github.com/PuerkitoBio/goquery"
)

type GitHubTrendingSource struct{}

func (GitHubTrendingSource) Key() string         { return "github" }
func (GitHubTrendingSource) Label() string       { return "GitHub Trending" }
func (GitHubTrendingSource) HomePageURL() string { return "https://github.com/trending" }

func (s GitHubTrendingSource) Fetch(ctx context.Context, client *http.Client) ([]domain.FeedItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.HomePageURL(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &httpError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return parseGitHubTrending(resp.Body)
}

func parseGitHubTrending(reader io.Reader) ([]domain.FeedItem, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}
	items := []domain.FeedItem{}
	doc.Find("article.Box-row").Each(func(i int, article *goquery.Selection) {
		repoLink := article.Find("h2 a").First()
		if repoLink.Length() == 0 {
			return
		}
		href, _ := repoLink.Attr("href")
		repoPath := normalizeGitHubRepoPath(href, repoLink.Text())
		if repoPath == "" {
			return
		}
		description := cleanString(article.Find("p").First().Text())
		language := cleanString(article.Find(`[itemprop="programmingLanguage"]`).First().Text())
		stars := parseDigits(article.Find(`a[href$="/stargazers"]`).First().Text())
		forks := parseDigits(article.Find(`a[href$="/forks"]`).First().Text())
		articleText := strings.Join(strings.Fields(article.Text()), " ")
		starsToday := extractInt(articleText, `(\d[\d,]*)\s+stars today`)
		metadata := map[string]any{}
		if language != nil {
			metadata["language"] = *language
		}
		if stars != nil {
			metadata["total_stars"] = *stars
		}
		if forks != nil {
			metadata["forks"] = *forks
		}
		if starsToday != nil {
			metadata["stars_today"] = *starsToday
		}
		items = append(items, domain.FeedItem{
			Source:     "github",
			ExternalID: strings.ToLower(repoPath),
			Title:      repoPath,
			URL:        resolveGitHubURL(href),
			Summary:    description,
			Score:      starsToday,
			SourceRank: i + 1,
			Metadata:   metadata,
		})
	})
	return items, nil
}

func resolveGitHubURL(path string) string {
	base, _ := url.Parse("https://github.com")
	rel, _ := url.Parse(path)
	return base.ResolveReference(rel).String()
}

func normalizeGitHubRepoPath(href string, linkText string) string {
	if parsed, err := url.Parse(strings.TrimSpace(href)); err == nil {
		path := strings.Trim(parsed.Path, "/")
		parts := strings.Split(path, "/")
		if len(parts) >= 2 && parts[0] != "" && parts[1] != "" {
			return parts[0] + "/" + parts[1]
		}
	}
	return strings.Trim(strings.Join(strings.Fields(linkText), ""), "/")
}

func parseDigits(value string) *int {
	cleaned := regexp.MustCompile(`[^\d]`).ReplaceAllString(value, "")
	if cleaned == "" {
		return nil
	}
	return extractInt(cleaned, `(\d+)`)
}
