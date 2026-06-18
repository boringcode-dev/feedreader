# feedreader implementation notes

This note tracks the concrete UI/API/runtime behavior of the current app so future changes can preserve contract-level expectations instead of only visual appearance.

## Current product surface

### Sources
- Hacker News
- GitHub Trending
- Hugging Face Papers Trending

### Storage and fetch
- SQLite is the only persistent store.
- Feed items are upserted by `(source, external_id)`.
- Older items are retained.
- Fetch failures must not wipe the last good dataset.
- Refresh scheduling is aligned to 3-hour wall-clock boundaries in UTC+7.

## Card summary contract

The source adapters already capture raw metadata, but the visible card summaries are built later in the service layer. Current rendered metrics:

- Hacker News:
  - points
  - comments
  - separator style: `·`
- GitHub Trending:
  - total stars
  - stars today
  - forks
- Hugging Face Papers Trending:
  - upvotes

This formatting lives in `internal/service/service.go` rather than in the source adapters.

## Search contract

### UX
- Search expands inline in the header.
- Clicking the search icon focuses the input.
- The search input renders at `16px` to reduce iOS Safari auto-zoom risk.
- Typing is debounced before calling `/api/items`.
- Closing search clears the query and resets the feed.

### API
- `GET /api/items`
- supported query params:
  - `source`
  - `sources`
  - `q`
  - `limit`
  - `offset`

Search matches across:
- title
- summary
- author
- URL
- stored metadata JSON

## Source configuration contract

### Persistence
- Selected visible sources are stored in browser local storage under:
  - `feedreader.sources`

### Dialog behavior
- Configure button appears before the source filters.
- Configure dialog exposes 3 checkboxes:
  - Hacker News
  - GitHub Trending
  - Hugging Face Papers Trending

### Filter bar behavior
- If all 3 sources are enabled:
  - visible filters are `All`, `HN`, `GH`, `HF`
- If 2 sources are enabled:
  - visible filters are `All` plus the enabled sources
- If exactly 1 source is enabled:
  - only that source filter is shown
- If the current active filter becomes invalid after saving source config:
  - fall back to the first visible source/filter

### Aggregate semantics
- `All` means **all enabled sources**, not all backend sources.
- The client expresses that through the `sources=` query param when needed.
- Backend support for that lives in `internal/web/web.go`, `internal/service/service.go`, and `internal/repository/sqlite.go`.

## Client architecture notes

The filter bar is rendered from client state, not just shown/hidden from the initial DOM.

Important consequence:
- do not rely on a static NodeList captured before re-render for filter clicks
- use delegated events on the stable filter-nav parent

## Mobile/responsive notes

- Search-active mobile layout hides the filter cluster so the search row can expand.
- Search input remains 16px on mobile.
- Header controls share a common control-height CSS variable.

## Documentation assets

- README screenshot asset:
  - `docs/assets/feedreader-home.png`

When UI changes materially, replace the screenshot and update README + this note together.
