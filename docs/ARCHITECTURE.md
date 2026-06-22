# Architecture

## Packages

| Package | Responsibility |
| --- | --- |
| [`cmd/feedreader`](../cmd/feedreader) | CLI entrypoint: `serve`, `fetch`, `healthcheck` subcommands. |
| [`internal/config`](../internal/config) | Env-var configuration loading and defaults. |
| [`internal/db`](../internal/db) | SQLite bootstrap: schema DDL, WAL/pragma setup. |
| [`internal/domain`](../internal/domain) | Plain data types shared across packages (`FeedItem`, `SyncState`, `CardView`, ...). |
| [`internal/repository`](../internal/repository) | Persistence: upserts, sync-state tracking, feed queries. |
| [`internal/service`](../internal/service) | Refresh orchestration, scheduler, card-building/display logic. |
| [`internal/sources`](../internal/sources) | Upstream source adapters (Hacker News, GitHub Trending, Hugging Face Papers, alphaXiv). |
| [`internal/web`](../internal/web) | HTTP routes, SSR page rendering, JSON APIs. |

## Data model and refresh behavior

- Items are upserted by `(source, external_id)`. A refresh never deletes
  existing rows — a failed fetch just records the failure in `sync_state` and
  leaves prior data in place.
- The original `published_at` is preserved across re-fetches via
  `coalesce(items.published_at, excluded.published_at)` — a source that later
  starts reporting a different date for the same item doesn't reorder it.
- `internal/repository/sqlite.go`'s `ListFeedItems` sorts and paginates **in
  application memory**, not in SQL. This is intentional: total item count
  across all 4 sources is small (a few hundred rows), so the simplicity of one
  in-memory comparator outweighs the complexity of expressing the same
  fallback-ordering (published date, else first-seen date, else source rank)
  in SQL.
- The scheduler in `internal/service/service.go` wakes on N-hour wall-clock
  boundaries in `Asia/Ho_Chi_Minh` (UTC+7, no DST) — default hourly via
  `FEEDREADER_REFRESH_INTERVAL_HOURS`. It does not refresh immediately on
  startup.

## Frontend

Server-rendered HTML (`web/templates/index.html`) plus vanilla JS
(`web/static/app.js`) — no frontend build step. The service worker
(`web/static/service-worker.js`) caches the app shell and visited
`/api/items` responses for offline reuse.
