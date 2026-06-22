@AGENTS.md

## Claude-specific notes

- Don't read or modify files under `data/` (gitignored local SQLite DBs).
- Prefer `go test ./internal/<package>/...` over a full `go test ./...` run
  when iterating on a single package.
