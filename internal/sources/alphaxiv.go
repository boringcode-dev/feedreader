package sources

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"path"
	"regexp"
	"strings"
	"time"

	"feedreader/internal/domain"
	"github.com/PuerkitoBio/goquery"
)

type AlphaXivSource struct{}

func (AlphaXivSource) Key() string         { return "alphaxiv" }
func (AlphaXivSource) Label() string       { return "alphaXiv" }
func (AlphaXivSource) HomePageURL() string { return "https://www.alphaxiv.org/" }

func (s AlphaXivSource) Fetch(ctx context.Context, client *http.Client) ([]domain.FeedItem, error) {
	resp, err := getWithRetry(ctx, client, s.HomePageURL())
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return nil, &httpError{StatusCode: resp.StatusCode, Body: string(body)}
	}
	return parseAlphaXivExplore(resp.Body)
}

func parseAlphaXivExplore(reader io.Reader) ([]domain.FeedItem, error) {
	doc, err := goquery.NewDocumentFromReader(reader)
	if err != nil {
		return nil, err
	}

	items := []domain.FeedItem{}
	seen := map[string]struct{}{}
	doc.Find(`a[href^="/abs/"]`).Each(func(_ int, link *goquery.Selection) {
		title := strings.Join(strings.Fields(link.Text()), " ")
		if title == "" || strings.EqualFold(title, "View blog") {
			return
		}
		href, ok := link.Attr("href")
		if !ok || strings.TrimSpace(href) == "" {
			return
		}
		absoluteURL := resolveAlphaXivURL(href)
		externalID := alphaXivExternalID(absoluteURL)
		if externalID == "" {
			return
		}
		if _, exists := seen[externalID]; exists {
			return
		}
		card := alphaXivCardContainer(link)
		if card == nil {
			return
		}

		authors := alphaXivAuthors(card)
		metadata := map[string]any{}
		if len(authors) > 0 {
			metadata["authors"] = authors
		}
		tags := alphaXivTags(card)
		if len(tags) > 0 {
			metadata["tags"] = tags
		}
		items = append(items, domain.FeedItem{
			Source:      "alphaxiv",
			ExternalID:  externalID,
			Title:       title,
			URL:         absoluteURL,
			Summary:     alphaXivSummary(card),
			Author:      cleanString(strings.Join(authors, ", ")),
			Score:       alphaXivScore(card),
			PublishedAt: alphaXivPublishedAt(card),
			SourceRank:  len(items) + 1,
			Metadata:    metadata,
		})
		seen[externalID] = struct{}{}
	})
	return items, nil
}

func alphaXivCardContainer(link *goquery.Selection) *goquery.Selection {
	for node := link.Parent(); node != nil && node.Length() > 0; node = node.Parent() {
		if node.Find("svg.lucide-thumbs-up").Length() > 0 || node.Find(`a[href^="/audio/"]`).Length() > 0 || node.Find(`a[href^="/replicate/"]`).Length() > 0 {
			return node
		}
	}
	return nil
}

func alphaXivExternalID(rawURL string) string {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return ""
	}
	if !strings.HasPrefix(parsed.Path, "/abs/") {
		return ""
	}
	id := path.Base(strings.TrimSuffix(parsed.Path, "/"))
	if id == "." || id == "/" {
		return ""
	}
	return strings.TrimSpace(id)
}

func alphaXivSummary(card *goquery.Selection) *string {
	best := ""
	card.Find("p").Each(func(_ int, p *goquery.Selection) {
		text := strings.Join(strings.Fields(p.Text()), " ")
		if len(text) > len(best) {
			best = text
		}
	})
	return cleanString(best)
}

func alphaXivAuthors(card *goquery.Selection) []string {
	seen := map[string]struct{}{}
	authors := []string{}
	appendAuthor := func(name string) {
		name = strings.Join(strings.Fields(name), " ")
		if name == "" || strings.HasPrefix(name, "#") || strings.EqualFold(name, "View blog") || len(name) > 80 {
			return
		}
		if _, exists := seen[name]; exists {
			return
		}
		seen[name] = struct{}{}
		authors = append(authors, name)
	}
	card.Find(`[aria-haspopup="dialog"]`).Each(func(_ int, node *goquery.Selection) {
		appendAuthor(node.Text())
	})
	if len(authors) > 0 {
		return authors
	}
	card.Find(`div.font-normal, span.font-normal`).Each(func(_ int, node *goquery.Selection) {
		appendAuthor(node.Text())
	})
	return authors
}

func alphaXivScore(card *goquery.Selection) *int {
	var score *int
	card.Find("button").EachWithBreak(func(_ int, button *goquery.Selection) bool {
		if button.Find("svg.lucide-thumbs-up").Length() == 0 {
			return true
		}
		score = parseDigits(button.Text())
		return false
	})
	return score
}

func alphaXivTags(card *goquery.Selection) []string {
	seen := map[string]struct{}{}
	tags := []string{}
	card.Find("a").Each(func(_ int, link *goquery.Selection) {
		href, ok := link.Attr("href")
		if !ok {
			return
		}
		if !strings.Contains(href, "?subcategories=") && !strings.Contains(href, "?categories=") && !strings.Contains(href, "?custom-categories=") {
			return
		}
		tag := strings.TrimPrefix(strings.Join(strings.Fields(link.Text()), " "), "#")
		if tag == "" {
			return
		}
		if _, exists := seen[tag]; exists {
			return
		}
		seen[tag] = struct{}{}
		tags = append(tags, tag)
	})
	return tags
}

var alphaXivDatePattern = regexp.MustCompile(`\b\d{1,2}\s+[A-Z][a-z]{2}\s+\d{4}\b`)

func alphaXivPublishedAt(card *goquery.Selection) *time.Time {
	match := ""
	card.Find("span").EachWithBreak(func(_ int, node *goquery.Selection) bool {
		text := strings.Join(strings.Fields(node.Text()), " ")
		match = alphaXivDatePattern.FindString(text)
		return match == ""
	})
	if match == "" {
		normalized := strings.Join(strings.Fields(card.Text()), " ")
		match = alphaXivDatePattern.FindString(normalized)
	}
	if match == "" {
		return nil
	}
	parsed, err := time.Parse("2 Jan 2006", match)
	if err != nil {
		return nil
	}
	utc := parsed.UTC()
	return &utc
}

func resolveAlphaXivURL(rawPath string) string {
	base, _ := url.Parse("https://www.alphaxiv.org")
	rel, err := url.Parse(strings.TrimSpace(rawPath))
	if err != nil {
		return rawPath
	}
	return base.ResolveReference(rel).String()
}
