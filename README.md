# feedreader

<p align="center">
  <strong>A tiny, fast, self-hosted feed reader for engineering and research signals.</strong>
</p>

<p align="center">
  Server-rendered UI · SQLite storage · Scheduled refresh · Docker-friendly · Private by default
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.24-00ADD8?logo=go&logoColor=white" alt="Go 1.24" />
  <img src="https://img.shields.io/badge/SQLite-3-003B57?logo=sqlite&logoColor=white" alt="SQLite" />
  <img src="https://img.shields.io/badge/Docker-ready-2496ED?logo=docker&logoColor=white" alt="Docker ready" />
  <img src="https://img.shields.io/badge/Frontend-server--rendered-111827" alt="Server rendered UI" />
</p>

---

## Screenshot

![feedreader home screen](docs/assets/feedreader-home.png)

---

## Features

- **Multi-source feed aggregation**
  - Hacker News
  - GitHub Trending
  - Hugging Face Papers
- **Persistent local storage** with SQLite
- **Incremental fetch model** that keeps older items in the database
- **Scheduled refresh** every 3 hours on wall-clock boundaries in UTC+7
- **Manual refresh** from the UI
- **Responsive, minimalist UI** with source filters and dark/light mode
- **PWA-ready assets** including manifest, service worker, and touch icons
- **Docker deployment** with reverse-proxy-friendly HTTP service

---

## Why feedreader?

`feedreader` is designed for people who want a small, understandable, self-hosted reader instead of a large feed platform.

It optimizes for:

- simple operations
- low memory usage
- straightforward data ownership
- easy extension when adding more sources

---

## Tech stack

### Backend
- Go
- `net/http`
- `html/template`
- `modernc.org/sqlite`
- `goquery`

### Frontend
- Server-rendered HTML
- Vanilla JavaScript
- Plain CSS

### Storage
- SQLite

### Deployment
- Docker
- Reverse proxy compatible

---

## Architecture

At a high level:

1. source adapters fetch upstream content
2. items are upserted into SQLite by `(source, external_id)`
3. the web app reads stored items ordered by article date descending
4. the scheduler refreshes on 3-hour clock boundaries

Key properties:

- old items are retained in the database
- fetch failures do not wipe existing data
- sources without a native article date fall back to fetch/update time for ordering

---

## Project structure

```text
cmd/feedreader/         CLI entrypoint
internal/config/        configuration loading
internal/db/            SQLite bootstrap and pragmas
internal/domain/        domain models
internal/repository/    persistence layer
internal/service/       refresh orchestration and scheduler
internal/sources/       upstream source adapters
internal/web/           HTTP handlers and page rendering
web/templates/          HTML templates
web/static/             CSS, JS, icons, PWA assets
```

---

## Getting started

### Prerequisites

- Go 1.24+ for local builds
- or Docker for containerized usage

### Run locally

```bash
go run ./cmd/feedreader serve --host 0.0.0.0 --port 8080
```

Then open:

- `http://127.0.0.1:8080`

### Manual refresh

```bash
go run ./cmd/feedreader fetch
```

### Docker build

```bash
docker build -t feedreader .
```

### Docker run

```bash
docker run --rm -p 8080:8080 -v $(pwd)/data:/data feedreader
```

Then open:

- `http://127.0.0.1:8080`

---

## Configuration

Environment variables:

| Variable | Default | Description |
|---|---:|---|
| `FEEDREADER_DB_PATH` | `./data/feedreader.db` | SQLite database path |
| `FEEDREADER_REFRESH_INTERVAL_HOURS` | `3` | Refresh interval setting used by the scheduler |
| `FEEDREADER_ITEMS_PER_SOURCE` | `20` | Per-source item count used in source dashboard/health contexts |
| `FEEDREADER_REQUEST_TIMEOUT_SECONDS` | `20` | Upstream request timeout |
| `FEEDREADER_USER_AGENT` | `feedreader/0.1` | Outbound fetch user agent |
| `FEEDREADER_HOST` | `0.0.0.0` | HTTP bind host |
| `FEEDREADER_PORT` | `8080` | HTTP bind port |

---

## Scheduling

The scheduler runs **inside the app process**.

Behavior:

- aligned to **UTC+7** (`Asia/Ho_Chi_Minh`)
- runs on the next **3-hour wall-clock boundary**
- does **not** perform an immediate refresh just because the container starts

Manual refresh is also available through the UI and CLI.

---

## API

### `GET /healthz`
Returns service health and per-source refresh status.

### `GET /api/items`
Returns the flattened feed item list.

Query params:
- `source` — optional source filter (`hackernews`, `github`, `huggingface`)

### `POST /api/refresh`
Triggers an immediate refresh and returns source-level outcomes.

---

## Data model

The service stores a cumulative feed history.

Each fetch:

- upserts items by `(source, external_id)`
- updates refresh state in `sync_state`
- preserves older items already in the database

The UI/API render items from the full stored set, ordered by article date descending.

---

## Roadmap

Potential next improvements:

- more sources (blogs, changelogs, newsletters, papers)
- server-side pagination
- source weighting and ranking controls
- source-specific parsing tests with fixtures
- export/import support

---

## Contributing

Contributions are welcome.

A good contribution flow:

1. fork the repository
2. create a branch
3. make changes
4. run formatting and tests
5. open a pull request

Example local verification:

```bash
gofmt -w $(find . -name "*.go")
go test ./...
```

---

## Repository hygiene

The SQLite runtime data directory is intentionally ignored:

```gitignore
data/
```

This keeps the repository focused on source code and assets.
