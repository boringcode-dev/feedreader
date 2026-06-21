# Changelog

All notable changes to this project are documented in this file.

## [1.1.0](https://github.com/boringcode-dev/feedreader/compare/v1.0.0...v1.1.0) (2026-06-21)


### Features

* refresh feeds hourly and repurpose header refresh ([#3](https://github.com/boringcode-dev/feedreader/issues/3)) ([292005d](https://github.com/boringcode-dev/feedreader/commit/292005d16aaa13468f4c78a0da3e2d999c79c6ee))

## v1.0.0 - 2026-06-21

Initial public release of `feedreader`.

### Added

- Initial `feedreader` application with combined feed aggregation and self-hosted Go service runtime.
- Incremental backend loading for feed items.
- Feed search across items.
- Configurable source selection in the UI.
- Source icons across the feed experience.
- alphaXiv as a feed source.
- Toast-based loading states.
- Offline PWA cache and reconnect toasts.
- Offline connection indicator.
- Reader settings dialog.
- Browser-localized feed card dates.
- GitHub Actions CI for formatting and tests.
- GitHub Actions CD for multi-arch GHCR image publishing on release tags.

### Changed

- Rewrote the README for the open source release.
- Switched the Hugging Face integration to trending papers.
- Enriched feed cards with source-specific metrics.
- Rebranded the app UI and icons around the `feedreader` identity.
- Moved source settings access into the header.
- Updated local development and test instructions.
- Parameterized the Docker build for `linux/amd64` and `linux/arm64` image publishing.

### Fixed

- Refreshed app icons and footer branding.
- Preserved explicit alphaXiv source selection.
- Preserved stable feed ordering timestamps.
- Persisted visited links and refreshed the list in place.
- Brightened GitHub icons in dark mode.
- Simplified filter loading and search close behavior.
- Improved iOS search open and empty close behavior.
- Normalized GitHub repository identity handling.
- Applied density settings correctly in the settings dialog.
- Centered settings choice cards in compact mode.
- Busted stylesheet cache for settings alignment fixes.
