# Changelog

casa uses **date-based versioning**: the version is the release date, tagged
`vYYYY.MM.DD-N` (date, then a same-day counter that's always present), e.g.
`v2026.06.21-0`, then `v2026.06.21-1` for a second release the same day.
Entries below are keyed by version date, newest first.

## 2026.06.21-0

- Initial date-versioned release.
- `add` / `remove` / `update` across brew, cask, tap, go, uv, npm, cargo.
- Keeps the chezmoi-managed Brewfile in sync via `# casa:<manager>` anchors.
- `remove` lists every recorded package across all managers in one picker.
