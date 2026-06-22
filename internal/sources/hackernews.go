package sources

import (
	"context"
	"encoding/json"
	"html"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"feedreader/internal/domain"
)

const hackerNewsFrontPageAPI = "https://hn.algolia.com/api/v1/search?tags=front_page"

type HackerNewsSource struct{}

func (HackerNewsSource) Key() string         { return "hackernews" }
func (HackerNewsSource) Label() string       { return "Hacker News" }
func (HackerNewsSource) HomePageURL() string { return "https://news.ycombinator.com/" }

func (s HackerNewsSource) Fetch(ctx context.Context, client *http.Client) ([]domain.FeedItem, error) {
	resp, err := getWithRetry(ctx, client, hackerNewsFrontPageAPI)
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

type hnFrontPage struct {
	Hits []hnStory `json:"hits"`
}

type hnStory struct {
	ObjectID    string `json:"objectID"`
	StoryID     int    `json:"story_id"`
	Title       string `json:"title"`
	StoryTitle  string `json:"story_title"`
	URL         string `json:"url"`
	StoryURL    string `json:"story_url"`
	StoryText   string `json:"story_text"`
	CommentText string `json:"comment_text"`
	Author      string `json:"author"`
	Points      *int   `json:"points"`
	NumComments *int   `json:"num_comments"`
	CreatedAt   string `json:"created_at"`
}

func parseHackerNews(payload []byte) ([]domain.FeedItem, error) {
	var rss hnFrontPage
	if err := json.Unmarshal(payload, &rss); err != nil {
		return nil, err
	}
	items := make([]domain.FeedItem, 0, len(rss.Hits))
	for idx, node := range rss.Hits {
		externalID := strings.TrimSpace(node.ObjectID)
		if externalID == "" && node.StoryID > 0 {
			externalID = strconv.Itoa(node.StoryID)
		}
		if externalID == "" {
			continue
		}
		commentsURL := "https://news.ycombinator.com/item?id=" + externalID
		metadata := map[string]any{}
		if node.NumComments != nil {
			metadata["comments_count"] = *node.NumComments
		}
		items = append(items, domain.FeedItem{
			Source:      "hackernews",
			ExternalID:  externalID,
			Title:       strings.TrimSpace(firstNonEmpty(node.Title, node.StoryTitle, externalID)),
			URL:         strings.TrimSpace(firstNonEmpty(node.URL, node.StoryURL, commentsURL)),
			Summary:     cleanString(extractHNSummary(firstNonEmpty(node.StoryText, node.CommentText))),
			Author:      cleanString(strings.TrimSpace(node.Author)),
			Score:       node.Points,
			CommentsURL: cleanString(commentsURL),
			PublishedAt: parseHackerNewsTime(node.CreatedAt),
			SourceRank:  idx + 1,
			Metadata:    metadata,
		})
	}
	return items, nil
}

func parseHackerNewsTime(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.TrimSpace(value))
	if err != nil {
		return nil
	}
	utc := parsed.UTC()
	return &utc
}

func extractHNSummary(description string) string {
	cleaned := regexp.MustCompile(`<a [^>]+>|</a>|<[^>]+>`).ReplaceAllString(description, " ")
	cleaned = html.UnescapeString(cleaned)
	cleaned = regexp.MustCompile(`\s+`).ReplaceAllString(cleaned, " ")
	return strings.TrimSpace(cleaned)
}
