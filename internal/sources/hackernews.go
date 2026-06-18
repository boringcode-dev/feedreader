package sources

import (
	"context"
	"encoding/xml"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"feedreader/internal/domain"
)

type HackerNewsSource struct{}

func (HackerNewsSource) Key() string         { return "hackernews" }
func (HackerNewsSource) Label() string       { return "Hacker News" }
func (HackerNewsSource) HomePageURL() string { return "https://news.ycombinator.com/" }

func (s HackerNewsSource) Fetch(ctx context.Context, client *http.Client) ([]domain.FeedItem, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://hnrss.org/frontpage", nil)
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseHackerNews(body)
}

type hnRSS struct {
	Channel struct {
		Items []hnItem `xml:"item"`
	} `xml:"channel"`
}

type hnItem struct {
	Title       string `xml:"title"`
	Description string `xml:"description"`
	PubDate     string `xml:"pubDate"`
	Link        string `xml:"link"`
	Comments    string `xml:"comments"`
	Guid        string `xml:"guid"`
	Creator     string `xml:"creator"`
}

func parseHackerNews(payload []byte) ([]domain.FeedItem, error) {
	var rss hnRSS
	if err := xml.Unmarshal(payload, &rss); err != nil {
		return nil, err
	}
	items := make([]domain.FeedItem, 0, len(rss.Channel.Items))
	for idx, node := range rss.Channel.Items {
		var publishedAt *time.Time
		if node.PubDate != "" {
			if parsed, err := time.Parse(time.RFC1123Z, node.PubDate); err == nil {
				t := parsed.UTC()
				publishedAt = &t
			}
		}
		score := extractInt(node.Description, `Points:\s*(\d+)`)
		commentsCount := extractInt(node.Description, `# Comments:\s*(\d+)`)
		metadata := map[string]any{}
		if commentsCount != nil {
			metadata["comments_count"] = *commentsCount
		}
		items = append(items, domain.FeedItem{
			Source:      "hackernews",
			ExternalID:  extractStoryID(firstNonEmpty(node.Comments, node.Guid, node.Link)),
			Title:       strings.TrimSpace(node.Title),
			URL:         strings.TrimSpace(node.Link),
			Summary:     cleanString(extractHNSummary(node.Description)),
			Author:      cleanString(strings.TrimSpace(node.Creator)),
			Score:       score,
			CommentsURL: cleanString(strings.TrimSpace(node.Comments)),
			PublishedAt: publishedAt,
			SourceRank:  idx + 1,
			Metadata:    metadata,
		})
	}
	return items, nil
}

func extractStoryID(value string) string {
	re := regexp.MustCompile(`id=(\d+)`)
	if match := re.FindStringSubmatch(value); len(match) == 2 {
		return match[1]
	}
	return strings.TrimSpace(value)
}

func extractHNSummary(description string) string {
	head := strings.SplitN(description, "<hr>", 2)[0]
	replacer := regexp.MustCompile(`<a [^>]+>|</a>|<[^>]+>`)
	cleaned := replacer.ReplaceAllString(head, " ")
	cleaned = html.UnescapeString(cleaned)
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`Comments URL:\s*\S+`),
		regexp.MustCompile(`Article URL:\s*\S+`),
		regexp.MustCompile(`Points:\s*\d+`),
		regexp.MustCompile(`# Comments:\s*\d+`),
		regexp.MustCompile(`\s+`),
	}
	for _, pattern := range patterns {
		cleaned = pattern.ReplaceAllString(cleaned, " ")
	}
	return strings.TrimSpace(cleaned)
}

func extractInt(value, pattern string) *int {
	re := regexp.MustCompile(pattern)
	match := re.FindStringSubmatch(value)
	if len(match) != 2 {
		return nil
	}
	parsed := strings.ReplaceAll(match[1], ",", "")
	if parsed == "" {
		return nil
	}
	out, err := strconv.Atoi(parsed)
	if err != nil {
		return nil
	}
	return &out
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func cleanString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}
