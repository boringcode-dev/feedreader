package web

import (
	"encoding/json"
	"html/template"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"feedreader/internal/config"
	"feedreader/internal/domain"
	"feedreader/internal/service"
)

type App struct {
	cfg        config.Config
	service    *service.FeedService
	templates  *template.Template
	staticRoot string
	mux        *http.ServeMux
}

type pageData struct {
	Cards               []domain.CardView
	Errors              []domain.ErrorView
	SourceFilters       []sourceFilter
	InitialVisibleCards int
	CurrentYear         int
}

type sourceFilter struct {
	Key   string
	Label string
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

func (a *App) Handler() http.Handler {
	return a.mux
}

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
	items, err := a.service.FeedItems(0, "")
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
		Cards:               service.BuildCards(items),
		Errors:              service.BuildErrors(snapshots),
		SourceFilters:       []sourceFilter{{Key: "all", Label: "All"}, {Key: "hackernews", Label: "HN"}, {Key: "github", Label: "GH"}, {Key: "huggingface", Label: "HF"}},
		InitialVisibleCards: 12,
		CurrentYear:         time.Now().UTC().Year(),
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
	requestedSource := r.URL.Query().Get("source")
	items, err := a.service.FeedItems(0, requestedSource)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	payloadItems := make([]map[string]any, 0, len(items))
	for _, item := range items {
		payloadItems = append(payloadItems, map[string]any{
			"source":       item.Source,
			"external_id":  item.ExternalID,
			"title":        item.Title,
			"url":          item.URL,
			"summary":      maybeString(item.Summary),
			"author":       maybeString(item.Author),
			"score":        maybeInt(item.Score),
			"comments_url": maybeString(item.CommentsURL),
			"published_at": maybeTime(item.PublishedAt),
			"source_rank":  item.SourceRank,
			"metadata":     item.Metadata,
		})
	}
	payload := map[string]any{
		"generated_at": time.Now().UTC().Format(time.RFC3339Nano),
		"items":        payloadItems,
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
	payload := map[string]any{
		"ok":       allOK,
		"outcomes": outcomes,
	}
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
