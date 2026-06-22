# Deployment (Docker)

This is the only supported deployment for this repo. A Cloudflare Workers
port lives in a separate repo, `boringcode-dev/feedreader-edge`.

## Build

```bash
docker build -t feedreader .
```

CI publishes multi-arch images (`linux/amd64`, `linux/arm64`) to
`ghcr.io/boringcode-dev/feedreader` on `v*.*.*` tag pushes
([.github/workflows/cd.yml](../.github/workflows/cd.yml)).

## Run

```bash
docker run --rm -p 8080:8080 -v $(pwd)/data:/data feedreader
```

The `/data` volume holds the SQLite database (`FEEDREADER_DB_PATH`, default
`/data/feedreader.db` inside the container). Losing this volume loses all
fetched history; sources are re-fetched from scratch on the next refresh.

## Configuration

See the env var table in [README.md](../README.md#configuration) — this doc
intentionally doesn't duplicate it.

## Release flow

1. Merge to `main` — CI runs `gofmt -l` + `go test ./...`.
2. [release-please](../.github/workflows/release-please.yml) opens a release
   PR based on Conventional Commit messages; merging it tags a version and
   updates `CHANGELOG.md`.
3. The tag push triggers `cd.yml`, which builds and pushes the image, then
   appends container-pull instructions to the GitHub release notes.
