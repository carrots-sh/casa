# Changelog

casa uses **date-based versioning**: the version is the release date, tagged
`vYYYY.MMDD.N` (year . zero-padded month+day . same-day counter), e.g.
`v2026.0621.0`. Entries below are keyed by version date, newest first.

## 2026.0621.0

- Initial date-versioned release.
- `add` / `remove` / `update` across brew, cask, tap, go, uv, npm, cargo.
- Keeps the chezmoi-managed Brewfile in sync via `# casa:<manager>` anchors.
- `remove` lists every recorded package across all managers in one picker.
