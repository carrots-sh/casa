# Changelog

casa uses **semver**: `vMAJOR.MINOR.PATCH`, newest first. (Releases before
v0.1.0 used date-based versions, `vYYYY.MM.DD-N`; those tags and releases were
retired when the scheme changed — their entries remain below for history.)

## 0.1.0

First semver release — everything since the last date-based one:

- **One manifest, no Brewfile**: every tool is recorded in `.casadata/packages.toml`;
  on apply it renders straight into `brew bundle --file=-` (install + cleanup).
  First `tools add` bootstraps the manifest, imports a legacy Brewfile, or scans
  the machine (`tools import`). Self-installing tools (`curl … | sh`) are
  first-class `[[packages.sh]]` entries with optional self-update commands.
- **Templates & setup questions**: track asks how to store files (plain /
  template / encrypted / encrypted template), `configs storage` converts later,
  `machine answers` / `machine question` manage the setup questionnaire in
  casa's own UI, encrypted templates validate on edit.
- **casa names everywhere**: repos commit `.casa.toml.tmpl`, `.casaignore`,
  `.casadata/` — casa self-heals the gitignored chezmoi-name symlinks before
  every chezmoi call, and fresh-machine setup clones first so the questionnaire
  is found.
- **Self-updater**: `casa upgrade` pulls the latest GitHub release in place.
- **UX**: paths display as `~/…` everywhere; the menu clusters commands by
  domain (configs / tools / secrets / machine / casa) with urgency moving the
  cursor, not the rows.
- **Tooling**: `make ship` = fmt + lint + modernize + tests + pty-driven e2e
  suite + install.

## 2026.06.22-0

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
