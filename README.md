# feedreader

Go port of the private feed reader.

## Runtime
- server-rendered HTML
- SQLite cumulative feed store
- wall-clock scheduler every 3 hours in UTC+7 (`Asia/Ho_Chi_Minh`)
- no immediate refresh on container start
- Docker deployment behind shared nginx

## Commands
```bash
/feedreader serve --host 0.0.0.0 --port 8080
/feedreader fetch
/feedreader healthcheck
```

## Data model
- old items are retained in SQLite
- new fetches upsert by `(source, external_id)`
- UI/API list items ordered by article date descending
- sources without article dates fall back to fetch/update time for ordering
