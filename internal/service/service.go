package service

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"feedreader/internal/config"
	"feedreader/internal/domain"
	"feedreader/internal/repository"
	"feedreader/internal/sources"
)

type FeedService struct {
	cfg     config.Config
	repo    *repository.SQLiteRepository
	sources []sources.Source
	client  *http.Client
	mu      sync.Mutex
}

func New(cfg config.Config, repo *repository.SQLiteRepository) *FeedService {
	return &FeedService{
		cfg:     cfg,
		repo:    repo,
		sources: sources.Build(),
		client:  newHTTPClient(cfg),
	}
}

func newHTTPClient(cfg config.Config) *http.Client {
	return &http.Client{
		Timeout: time.Duration(cfg.RequestTimeoutSec * float64(time.Second)),
		Transport: &userAgentTransport{
			base:      http.DefaultTransport,
			userAgent: strings.TrimSpace(cfg.UserAgent),
		},
	}
}

type userAgentTransport struct {
	base      http.RoundTripper
	userAgent string
}

func (t *userAgentTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	transport := t.base
	if transport == nil {
		transport = http.DefaultTransport
	}
	if strings.TrimSpace(t.userAgent) == "" || req.Header.Get("User-Agent") != "" {
		return transport.RoundTrip(req)
	}
	clone := req.Clone(req.Context())
	clone.Header = req.Header.Clone()
	clone.Header.Set("User-Agent", t.userAgent)
	return transport.RoundTrip(clone)
}

func (s *FeedService) StartScheduler(ctx context.Context) {
	go func() {
		location := loadScheduleLocation()
		for {
			now := time.Now().In(location)
			next := nextScheduledRefresh(now, s.cfg.RefreshIntervalHours)
			wait := time.Until(next)
			if wait < 0 {
				wait = time.Second
			}
			timer := time.NewTimer(wait)
			select {
			case <-ctx.Done():
				timer.Stop()
				return
			case <-timer.C:
				_ = s.RefreshAll(ctx)
			}
		}
	}()
}

func (s *FeedService) RefreshAll(ctx context.Context) []domain.RefreshOutcome {
	s.mu.Lock()
	defer s.mu.Unlock()

	outcomes := make([]domain.RefreshOutcome, len(s.sources))
	var wg sync.WaitGroup
	for i, source := range s.sources {
		wg.Add(1)
		go func(idx int, source sources.Source) {
			defer wg.Done()
			outcomes[idx] = s.refreshOne(ctx, source)
		}(i, source)
	}
	wg.Wait()
	return outcomes
}

func (s *FeedService) refreshOne(ctx context.Context, source sources.Source) domain.RefreshOutcome {
	attemptedAt := time.Now().UTC()
	items, err := source.Fetch(ctx, s.client)
	if err != nil {
		_ = s.repo.RecordFailure(source.Key(), attemptedAt, err.Error())
		return domain.RefreshOutcome{Source: source.Key(), OK: false, Error: err.Error()}
	}
	if len(items) == 0 {
		err = fmt.Errorf("source returned zero items")
		_ = s.repo.RecordFailure(source.Key(), attemptedAt, err.Error())
		return domain.RefreshOutcome{Source: source.Key(), OK: false, Error: err.Error()}
	}
	if err := s.repo.SaveSnapshot(source.Key(), attemptedAt, items); err != nil {
		_ = s.repo.RecordFailure(source.Key(), attemptedAt, err.Error())
		return domain.RefreshOutcome{Source: source.Key(), OK: false, Error: err.Error()}
	}
	return domain.RefreshOutcome{Source: source.Key(), OK: true, ItemCount: len(items)}
}

func (s *FeedService) Dashboard(limit int) ([]domain.SourceSnapshot, error) {
	if limit <= 0 {
		limit = s.cfg.ItemsPerSource
	}
	states, err := s.repo.ListSourceStates()
	if err != nil {
		return nil, err
	}
	snapshots := make([]domain.SourceSnapshot, 0, len(s.sources))
	for _, source := range s.sources {
		state, ok := states[source.Key()]
		if !ok {
			state = domain.SyncState{Source: source.Key()}
		}
		items, err := s.repo.GetCurrentItems(source.Key(), limit)
		if err != nil {
			return nil, err
		}
		snapshots = append(snapshots, domain.SourceSnapshot{
			Source:        source.Key(),
			Label:         source.Label(),
			HomepageURL:   source.HomePageURL(),
			LastAttemptAt: state.LastAttemptAt,
			LastSuccessAt: state.LastSuccessAt,
			LastError:     state.LastError,
			ItemCount:     state.ItemCount,
			Items:         items,
		})
	}
	return snapshots, nil
}

func (s *FeedService) FeedItems(limit int, offset int, source string, sources []string, searchQuery string) ([]domain.FeedItem, bool, error) {
	fetchLimit := limit
	if fetchLimit > 0 {
		fetchLimit = limit + 1
	}
	items, err := s.repo.ListFeedItems(fetchLimit, offset, source, sources, searchQuery)
	if err != nil {
		return nil, false, err
	}
	hasNext := false
	if limit > 0 && len(items) > limit {
		hasNext = true
		items = items[:limit]
	}
	return items, hasNext, nil
}

func (s *FeedService) HealthPayload() (map[string]any, error) {
	snapshots, err := s.Dashboard(1)
	if err != nil {
		return nil, err
	}
	total, err := s.repo.CountTotalItems()
	if err != nil {
		return nil, err
	}
	sourcesPayload := make([]map[string]any, 0, len(snapshots))
	for _, snapshot := range snapshots {
		sourcesPayload = append(sourcesPayload, map[string]any{
			"source":          snapshot.Source,
			"last_attempt_at": toISO(snapshot.LastAttemptAt),
			"last_success_at": toISO(snapshot.LastSuccessAt),
			"last_error":      derefString(snapshot.LastError),
			"item_count":      snapshot.ItemCount,
		})
	}
	return map[string]any{
		"status":      "ok",
		"total_items": total,
		"sources":     sourcesPayload,
	}, nil
}

func BuildCards(items []domain.FeedItem, offset int) []domain.CardView {
	cards := make([]domain.CardView, 0, len(items))
	for i, item := range items {
		briefPrefix := cardBriefPrefix(item)
		briefSuffix := cardBriefSuffix(item)
		briefDateISO, briefDateKind := cardDateParts(item)
		cards = append(cards, domain.CardView{
			Source:        item.Source,
			Index:         offset + i + 1,
			Title:         item.Title,
			URL:           item.URL,
			Brief:         composeBrief(briefPrefix, briefDateISO, briefDateKind, briefSuffix),
			BriefPrefix:   briefPrefix,
			BriefSuffix:   briefSuffix,
			BriefDateISO:  briefDateISO,
			BriefDateKind: briefDateKind,
			Host:          hostLabel(item.URL),
		})
	}
	return cards
}

func BuildErrors(snapshots []domain.SourceSnapshot) []domain.ErrorView {
	out := []domain.ErrorView{}
	for _, snapshot := range snapshots {
		if snapshot.LastError != nil && strings.TrimSpace(*snapshot.LastError) != "" {
			out = append(out, domain.ErrorView{Source: snapshot.Source, Label: snapshot.Label, Error: *snapshot.LastError})
		}
	}
	return out
}

func cardBriefPrefix(item domain.FeedItem) *string {
	fragments := cardStatFragments(item)
	if len(fragments) == 0 {
		return nil
	}
	value := strings.Join(fragments, " · ")
	return &value
}

func cardBriefSuffix(item domain.FeedItem) *string {
	switch item.Source {
	case "hackernews":
		return nil
	case "github":
		if summary := normalizedSummary(item.Summary); summary != nil {
			return summary
		}
		if language, ok := metadataString(item.Metadata, "language"); ok {
			value := "Trending " + language + " repository on GitHub"
			return &value
		}
		value := "Trending repository on GitHub"
		return &value
	case "huggingface":
		if summary := normalizedSummary(item.Summary); summary != nil {
			return summary
		}
		if item.Author != nil && strings.TrimSpace(*item.Author) != "" {
			value := strings.TrimSpace(*item.Author)
			return &value
		}
		value := "Trending paper on Hugging Face"
		return &value
	case "alphaxiv":
		if summary := normalizedSummary(item.Summary); summary != nil {
			return summary
		}
		if item.Author != nil && strings.TrimSpace(*item.Author) != "" {
			value := strings.TrimSpace(*item.Author)
			return &value
		}
		value := "Trending paper on alphaXiv"
		return &value
	default:
		return normalizedSummary(item.Summary)
	}
}

func cardStatFragments(item domain.FeedItem) []string {
	fragments := []string{}
	switch item.Source {
	case "hackernews":
		if item.Score != nil {
			fragments = append(fragments, fmtSprintf("%d points", *item.Score))
		}
		if comments, ok := metadataInt(item.Metadata, "comments_count"); ok {
			fragments = append(fragments, fmtSprintf("%d comments", comments))
		}
	case "github":
		if stars, ok := metadataInt(item.Metadata, "total_stars"); ok {
			fragments = append(fragments, fmtSprintf("%s stars", formatCount(stars)))
		}
		if today, ok := metadataInt(item.Metadata, "stars_today"); ok {
			fragments = append(fragments, fmtSprintf("%s today", formatCount(today)))
		}
		if forks, ok := metadataInt(item.Metadata, "forks"); ok {
			fragments = append(fragments, fmtSprintf("%s forks", formatCount(forks)))
		}
	case "huggingface":
		if item.Score != nil {
			fragments = append(fragments, fmtSprintf("%s upvotes", formatCount(*item.Score)))
		}
	case "alphaxiv":
		if item.Score != nil {
			fragments = append(fragments, fmtSprintf("%s likes", formatCount(*item.Score)))
		}
	}
	return fragments
}

func cardDateLabel(item domain.FeedItem) string {
	if item.PublishedAt != nil {
		return "Published " + item.PublishedAt.UTC().Format("Jan 2, 2006")
	}
	if item.FetchedAt != nil {
		return "Fetched " + item.FetchedAt.UTC().Format("Jan 2, 2006")
	}
	return ""
}

func cardDateParts(item domain.FeedItem) (*string, string) {
	if item.PublishedAt != nil {
		value := item.PublishedAt.UTC().Format(time.RFC3339Nano)
		return &value, "published"
	}
	if item.FetchedAt != nil {
		value := item.FetchedAt.UTC().Format(time.RFC3339Nano)
		return &value, "fetched"
	}
	return nil, ""
}

func composeBrief(prefix *string, dateISO *string, dateKind string, suffix *string) *string {
	parts := []string{}
	if prefix != nil && strings.TrimSpace(*prefix) != "" {
		parts = append(parts, strings.TrimSpace(*prefix))
	}
	if fallback := cardDateLabelFromParts(dateISO, dateKind); fallback != "" {
		parts = append(parts, fallback)
	}
	joined := strings.Join(parts, " · ")
	if suffix != nil && strings.TrimSpace(*suffix) != "" {
		text := strings.TrimSpace(*suffix)
		if joined != "" {
			value := joined + " - " + text
			return &value
		}
		return &text
	}
	if joined == "" {
		return nil
	}
	return &joined
}

func cardDateLabelFromParts(dateISO *string, dateKind string) string {
	if dateISO == nil || strings.TrimSpace(*dateISO) == "" {
		return ""
	}
	parsed, err := time.Parse(time.RFC3339Nano, *dateISO)
	if err != nil {
		return ""
	}
	verb := ""
	switch dateKind {
	case "published":
		verb = "Published"
	case "fetched":
		verb = "Fetched"
	default:
		return ""
	}
	return verb + " " + parsed.UTC().Format("Jan 2, 2006")
}

func normalizedSummary(summary *string) *string {
	if summary == nil || strings.TrimSpace(*summary) == "" {
		return nil
	}
	value := strings.Join(strings.Fields(*summary), " ")
	return &value
}

func hostLabel(rawURL string) string {
	trimmed := strings.TrimSpace(rawURL)
	trimmed = strings.TrimPrefix(trimmed, "https://")
	trimmed = strings.TrimPrefix(trimmed, "http://")
	parts := strings.SplitN(trimmed, "/", 2)
	host := strings.TrimPrefix(strings.ToLower(parts[0]), "www.")
	if host == "" {
		return rawURL
	}
	return host
}

func toISO(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}

func derefString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func metadataInt(metadata map[string]any, key string) (int, bool) {
	value, ok := metadata[key]
	if !ok {
		return 0, false
	}
	switch typed := value.(type) {
	case float64:
		return int(typed), true
	case int:
		return typed, true
	case int64:
		return int(typed), true
	default:
		return 0, false
	}
}

func metadataString(metadata map[string]any, key string) (string, bool) {
	value, ok := metadata[key]
	if !ok {
		return "", false
	}
	text, ok := value.(string)
	if !ok || strings.TrimSpace(text) == "" {
		return "", false
	}
	return text, true
}

func formatCount(value int) string {
	if value == 0 {
		return "0"
	}
	negative := value < 0
	if negative {
		value = -value
	}
	digits := fmtSprintf("%d", value)
	parts := []string{}
	for len(digits) > 3 {
		parts = append([]string{digits[len(digits)-3:]}, parts...)
		digits = digits[:len(digits)-3]
	}
	parts = append([]string{digits}, parts...)
	joined := strings.Join(parts, ",")
	if negative {
		return "-" + joined
	}
	return joined
}

func loadScheduleLocation() *time.Location {
	location, err := time.LoadLocation("Asia/Ho_Chi_Minh")
	if err == nil {
		return location
	}
	return time.FixedZone("UTC+7", 7*60*60)
}

func nextScheduledRefresh(now time.Time, intervalHours int) time.Time {
	if intervalHours < 1 {
		intervalHours = 1
	}
	location := now.Location()
	base := time.Date(now.Year(), now.Month(), now.Day(), now.Hour(), 0, 0, 0, location)
	hoursUntilNext := intervalHours - (now.Hour() % intervalHours)
	return base.Add(time.Duration(hoursUntilNext) * time.Hour)
}

func fmtSprintf(format string, values ...any) string {
	return fmt.Sprintf(format, values...)
}
