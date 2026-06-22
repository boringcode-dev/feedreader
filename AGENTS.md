# AGENTS.md

## Project overview

`feedreader` is a small Go service that aggregates Hacker News, GitHub Trending,
Hugging Face Papers Trending, and alphaXiv into a private, server-rendered feed
reader backed by SQLite. It ships as a Docker container.

## Build & run

```bash
go run ./cmd/feedreader serve --host 127.0.0.1 --port 8080
```

Configuration is env-var driven — see [internal/config/config.go](internal/config/config.go)
for the full list and defaults rather than duplicating it here.

## Test

```bash
gofmt -l $(git ls-files '*.go')   # must print nothing
go test ./...
```

Both checks run in CI ([.github/workflows/ci.yml](.github/workflows/ci.yml)) on every PR.

## Code style

`gofmt` is the only formatter. No additional linter is configured.

## Architecture

See [docs/ARCHITECTURE.md](docs/ARCHITECTURE.md) for the package-by-package layout.

## Adding a new feed source

Implement the `sources.Source` interface and register it in `sources.Build()`
in [internal/sources/sources.go](internal/sources/sources.go).

## Security considerations

There is no authentication on the HTTP API surface — this is designed for
private/personal deployment behind your own network or reverse proxy. Do not
add public write endpoints beyond the existing `POST /api/refresh`.

## Commit / PR conventions

Conventional Commits (`feat:`, `fix:`, `chore:`, ...) — releases are automated
via [release-please](.github/workflows/release-please.yml) based on commit
messages.
