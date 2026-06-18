package sources

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"

	"feedreader/internal/domain"
)

type HuggingFacePapersSource struct{}

func (HuggingFacePapersSource) Key() string         { return "huggingface" }
func (HuggingFacePapersSource) Label() string       { return "Hugging Face Papers Trending" }
func (HuggingFacePapersSource) HomePageURL() string { return "https://huggingface.co/papers/trending" }

func (s HuggingFacePapersSource) Fetch(ctx context.Context, client *http.Client) ([]domain.FeedItem, error) {
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
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseHuggingFacePapers(string(body))
}

type dailyProps struct {
	DailyPapers []struct {
		ID          string `json:"id"`
		Title       string `json:"title"`
		Summary     string `json:"summary"`
		Upvotes     *int   `json:"upvotes"`
		PublishedAt string `json:"publishedAt"`
		SubmittedBy struct {
			Fullname string `json:"fullname"`
			Name     string `json:"name"`
		} `json:"submittedBy"`
		Paper struct {
			ID          string `json:"id"`
			Title       string `json:"title"`
			Summary     string `json:"summary"`
			PublishedAt string `json:"publishedAt"`
			Upvotes     *int   `json:"upvotes"`
			NumComments *int   `json:"numComments"`
			Authors     []struct {
				Name string `json:"name"`
			} `json:"authors"`
		} `json:"paper"`
	} `json:"dailyPapers"`
}

func parseHuggingFacePapers(payload string) ([]domain.FeedItem, error) {
	re := regexp.MustCompile(`data-target="DailyPapers"\s+data-props="([^"]+)"`)
	match := re.FindStringSubmatch(payload)
	if len(match) != 2 {
		return nil, fmt.Errorf("unable to locate DailyPapers payload")
	}
	var props dailyProps
	if err := json.Unmarshal([]byte(html.UnescapeString(match[1])), &props); err != nil {
		return nil, err
	}
	items := make([]domain.FeedItem, 0, len(props.DailyPapers))
	for i, entry := range props.DailyPapers {
		paperID := firstNonEmpty(entry.Paper.ID, entry.ID)
		if paperID == "" {
			continue
		}
		authors := []string{}
		for _, author := range entry.Paper.Authors {
			if strings.TrimSpace(author.Name) != "" {
				authors = append(authors, strings.TrimSpace(author.Name))
			}
		}
		metadata := map[string]any{}
		if entry.Paper.NumComments != nil {
			metadata["comments_count"] = *entry.Paper.NumComments
		}
		if len(authors) > 0 {
			if len(authors) > 6 {
				metadata["authors"] = authors[:6]
			} else {
				metadata["authors"] = authors
			}
		}
		submittedBy := strings.TrimSpace(firstNonEmpty(entry.SubmittedBy.Fullname, entry.SubmittedBy.Name))
		if submittedBy != "" {
			metadata["submitted_by"] = submittedBy
		}
		items = append(items, domain.FeedItem{
			Source:      "huggingface",
			ExternalID:  paperID,
			Title:       strings.TrimSpace(firstNonEmpty(entry.Title, entry.Paper.Title, paperID)),
			URL:         "https://huggingface.co/papers/" + paperID,
			Summary:     cleanString(firstNonEmpty(entry.Summary, entry.Paper.Summary)),
			Author:      cleanString(authorSummary(authors)),
			Score:       firstInt(entry.Upvotes, entry.Paper.Upvotes),
			PublishedAt: parseISOTime(firstNonEmpty(entry.PublishedAt, entry.Paper.PublishedAt)),
			SourceRank:  i + 1,
			Metadata:    metadata,
		})
	}
	return items, nil
}

func parseISOTime(value string) *time.Time {
	if strings.TrimSpace(value) == "" {
		return nil
	}
	parsed, err := time.Parse(time.RFC3339, strings.ReplaceAll(value, "Z", "+00:00"))
	if err != nil {
		return nil
	}
	utc := parsed.UTC()
	return &utc
}

func authorSummary(authors []string) string {
	if len(authors) == 0 {
		return ""
	}
	if len(authors) <= 3 {
		return strings.Join(authors, ", ")
	}
	return strings.Join(authors[:3], ", ") + fmt.Sprintf(" +%d", len(authors)-3)
}

func firstInt(values ...*int) *int {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
