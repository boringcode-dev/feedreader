package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"feedreader/internal/config"
	"feedreader/internal/domain"
	"feedreader/internal/service"
)

const pageSize = 12

type App struct {
	cfg        config.Config
	service    *service.FeedService
	templates  *template.Template
	staticRoot string
	mux        *http.ServeMux
}

type pageData struct {
	Cards         []domain.CardView
	Errors        []domain.ErrorView
	SourceFilters []sourceFilter
	CurrentSource string
	SearchQuery   string
	SearchOpen    bool
	EmptyMessage  string
	PageSize      int
	HasNext       bool
	CurrentYear   int
}

type sourceFilter struct {
	Key      string
	Label    string
	IconPath string
	Active   bool
}

func New(cfg config.Config, svc *service.FeedService, baseDir string) (*App, error) {
	tmpl, err := template.ParseFiles(filepath.Join(baseDir, "web", "templates", "index.html"))
	if err != nil {
		return nil, err
	}
	app := &App{
		cfg:        cfg,
		service:    svc,
		templates:  tmpl,
		staticRoot: filepath.Join(baseDir, "web", "static"),
		mux:        http.NewServeMux(),
	}
	app.routes()
	return app, nil
}

func (a *App) Handler() http.Handler { return a.mux }

func (a *App) routes() {
	staticFS := http.FileServer(http.Dir(a.staticRoot))
	a.mux.Handle("/static/", http.StripPrefix("/static/", staticFS))
	a.mux.HandleFunc("/", a.home)
	a.mux.HandleFunc("/healthz", a.healthz)
	a.mux.HandleFunc("/api/items", a.itemsAPI)
	a.mux.HandleFunc("/api/refresh", a.refresh)
	a.mux.HandleFunc("/site.webmanifest", a.staticFile("site.webmanifest", "application/manifest+json"))
	a.mux.HandleFunc("/service-worker.js", a.staticFile("service-worker.js", "application/javascript"))
	a.mux.HandleFunc("/favicon.svg", a.staticFile("favicon.svg", "image/svg+xml"))
	a.mux.HandleFunc("/apple-touch-icon.png", a.staticFile("apple-touch-icon.png", "image/png"))
	a.mux.HandleFunc("/icon-192.png", a.staticFile("icon-192.png", "image/png"))
	a.mux.HandleFunc("/icon-512.png", a.staticFile("icon-512.png", "image/png"))
}

func (a *App) home(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}
	source := normalizeSource(r.URL.Query().Get("source"))
	searchQuery := normalizeSearchQuery(r.URL.Query().Get("q"))
	querySource := source
	if querySource == "all" {
		querySource = ""
	}
	items, hasNext, err := a.service.FeedItems(pageSize, 0, querySource, nil, searchQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	snapshots, err := a.service.Dashboard(1)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	data := pageData{
		Cards:         service.BuildCards(items, 0),
		Errors:        service.BuildErrors(snapshots),
		SourceFilters: buildSourceFilters(source),
		CurrentSource: source,
		SearchQuery:   searchQuery,
		SearchOpen:    searchQuery != "",
		EmptyMessage:  buildEmptyMessage(source, searchQuery),
		PageSize:      pageSize,
		HasNext:       hasNext,
		CurrentYear:   time.Now().UTC().Year(),
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := a.templates.ExecuteTemplate(w, "index.html", data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (a *App) healthz(w http.ResponseWriter, r *http.Request) {
	payload, err := a.service.HealthPayload()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (a *App) itemsAPI(w http.ResponseWriter, r *http.Request) {
	source := normalizeSource(r.URL.Query().Get("source"))
	searchQuery := normalizeSearchQuery(r.URL.Query().Get("q"))
	selectedSources := normalizeSourceList(r.URL.Query().Get("sources"))
	limit := parsePositiveInt(r.URL.Query().Get("limit"), pageSize)
	if limit > 100 {
		limit = 100
	}
	offset := parseNonNegativeInt(r.URL.Query().Get("offset"), 0)
	querySource := source
	if querySource == "all" {
		querySource = ""
	}
	items, hasNext, err := a.service.FeedItems(limit, offset, querySource, selectedSources, searchQuery)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	cards := service.BuildCards(items, offset)
	payloadCards := make([]map[string]any, 0, len(cards))
	for _, card := range cards {
		payloadCards = append(payloadCards, map[string]any{
			"source": card.Source,
			"index":  card.Index,
			"title":  card.Title,
			"url":    card.URL,
			"brief":  maybeString(card.Brief),
			"host":   card.Host,
		})
	}
	payload := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339Nano),
		"source":       source,
		"sources":      selectedSources,
		"query":        searchQuery,
		"offset":       offset,
		"limit":        limit,
		"has_next":     hasNext,
		"items":        payloadCards,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(payload)
}

func (a *App) refresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.Header().Set("Allow", http.MethodPost)
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	outcomes := a.service.RefreshAll(r.Context())
	allOK := true
	for _, outcome := range outcomes {
		if !outcome.OK {
			allOK = false
			break
		}
	}
	payload := map[string]any{"ok": allOK, "outcomes": outcomes}
	w.Header().Set("Content-Type", "application/json")
	if !allOK {
		w.WriteHeader(http.StatusBadGateway)
	}
	_ = json.NewEncoder(w).Encode(payload)
}

func (a *App) staticFile(name, contentType string) http.HandlerFunc {
	path := filepath.Join(a.staticRoot, name)
	return func(w http.ResponseWriter, r *http.Request) {
		if contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
		file, err := os.Open(path)
		if err != nil {
			http.NotFound(w, r)
			return
		}
		defer file.Close()
		http.ServeContent(w, r, name, time.Now(), file)
	}
}

func buildSourceFilters(current string) []sourceFilter {
	defs := []sourceFilter{
		{Key: "all", Label: "All enabled sources"},
		{Key: "hackernews", Label: "Hacker News", IconPath: "/static/source-icons/hackernews.svg"},
		{Key: "github", Label: "GitHub Trending", IconPath: "/static/source-icons/github.svg"},
		{Key: "huggingface", Label: "Hugging Face Papers Trending", IconPath: "/static/source-icons/huggingface.svg"},
		{Key: "alphaxiv", Label: "alphaXiv", IconPath: "/static/source-icons/alphaxiv.png"},
	}
	filters := make([]sourceFilter, 0, len(defs))
	for _, item := range defs {
		filters = append(filters, sourceFilter{Key: item.Key, Label: item.Label, IconPath: item.IconPath, Active: item.Key == current})
	}
	return filters
}

func normalizeSource(raw string) string {
	switch strings.TrimSpace(raw) {
	case "", "all":
		return "all"
	case "hackernews", "github", "huggingface", "alphaxiv":
		return raw
	default:
		return "all"
	}
}

func normalizeSourceList(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	seen := map[string]struct{}{}
	out := make([]string, 0, len(parts))
	for _, part := range parts {
		normalized := normalizeSource(part)
		if normalized == "all" {
			continue
		}
		if _, exists := seen[normalized]; exists {
			continue
		}
		seen[normalized] = struct{}{}
		out = append(out, normalized)
	}
	return out
}

func normalizeSearchQuery(raw string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(raw)), " ")
}

func buildEmptyMessage(source, searchQuery string) string {
	if searchQuery != "" {
		if source != "" && source != "all" {
			return "No matches found in " + sourceLabel(source) + ". Try a different query."
		}
		return "No matches found. Try a different query."
	}
	if source != "" && source != "all" {
		return "No items found in " + sourceLabel(source) + " right now."
	}
	return "No items yet. The scheduler will populate the feed automatically."
}

func sourceLabel(source string) string {
	switch source {
	case "hackernews":
		return "Hacker News"
	case "github":
		return "GitHub Trending"
	case "huggingface":
		return "Hugging Face Papers Trending"
	case "alphaxiv":
		return "alphaXiv"
	default:
		return "this source"
	}
}

func parsePositiveInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 1 {
		return fallback
	}
	return parsed
}

func parseNonNegativeInt(raw string, fallback int) int {
	if strings.TrimSpace(raw) == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(raw)
	if err != nil || parsed < 0 {
		return fallback
	}
	return parsed
}

func sourceURL(source string) string {
	values := url.Values{}
	if source != "" && source != "all" {
		values.Set("source", source)
	}
	encoded := values.Encode()
	if encoded == "" {
		return "/"
	}
	return "/?" + encoded
}

func maybeString(value *string) any {
	if value == nil {
		return nil
	}
	return *value
}

func maybeInt(value *int) any {
	if value == nil {
		return nil
	}
	return *value
}

func maybeTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.UTC().Format(time.RFC3339Nano)
}
