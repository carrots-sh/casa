# Changelog

casa uses **date-based versioning**: the version is the release date, tagged
`vYYYY.MM.DD-N` (date, then a same-day counter that's always present), e.g.
`v2026.06.21-0`, then `v2026.06.21-1` for a second release the same day.
Entries below are keyed by version date, newest first.

## 2026.06.21-5

UX round — make dotfiles dead-simple anywhere:

- **One-line bootstrap** `install.sh`: Homebrew + casa + optional setup in one command.
- **Fresh-machine auto-start**: running `casa` with nothing set up jumps straight to setup.
- **Auto commit messages**: `save` summarizes changed files (no typing).
- **Adopt picker**: `configs track` (no arg) offers common unmanaged dotfiles to manage.
- **Contexts checklist**: `machine context` toggles on/off contexts instead of re-asking everything.
- **Sensitive-file detection**: tracking a `.env`/`*.key`/credential offers to encrypt it.
- **`machine undo`**: revert the last saved change and re-apply.
- **Diff preview**: `save` shows a `--stat` of what will be committed.

## 2026.06.21-4

- **Branded source dir**: casa stores dotfiles in `~/.local/share/casa` by default
  (override with `$CASA_SOURCE`); existing `~/.local/share/chezmoi` setups keep working.
- **Smarter `machine setup`**: pass a github username (→ `<user>/dotfiles`), a
  `user/repo`, or a full URL. Prefers SSH, falls back to HTTPS, errors only if both fail.
- Light-mode theme fix and all-lowercase UI.

## 2026.06.21-1

- **Interactive-first**: `casa` opens a status-aware menu — nothing to memorize.
- Now a full, generic **chezmoi front-end**: `configs`, `secrets`, `machine`
  (setup / sync / save / status / context / doctor / info) alongside `tools`.
- Namespaced commands (`casa tools|configs|secrets|machine …`); the old flat
  `casa add/remove/update` are gone.
- Config-driven via optional `.casa.toml`; works on any chezmoi repo.

## 2026.06.21-0

- Initial date-versioned release.
- `add` / `remove` / `update` across brew, cask, tap, go, uv, npm, cargo.
- Keeps the chezmoi-managed Brewfile in sync via `# casa:<manager>` anchors.
- `remove` lists every recorded package across all managers in one picker.
