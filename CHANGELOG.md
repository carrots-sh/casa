# Changelog

casa uses **semver**: `vMAJOR.MINOR.PATCH`, newest first. (Releases before
v0.1.0 used date-based versions, `vYYYY.MM.DD-N`; those tags and releases were
retired when the scheme changed — their entries remain below for history.)

## 0.7.0

- The menu is action-first: clusters are act / see / change / undo / casa —
  the top-level question is "what do I want to do", not "which kind of
  thing". edit is type-smart (an encrypted pick routes through the secret
  flow: template validation + same-key re-seal) and list shows everything
  (files, tools, secrets) in one filterable pager. No action needs a second
  type pick. CLI commands unchanged.

## 0.6.1

- Menu clusters are frequency-ordered: daily verb first, list second in
  every cluster, occasional actions middle, destructive/rare last.

## 0.6.0

- Lists open inside the TUI: the menu.s list actions (configs, tools,
  secrets) show their output in a scrollable, type-to-filter view — same
  controls as every picker, esc/enter back to the menu — instead of raw
  terminal output. CLI list commands still print plainly for pipes.
- Menu list entries are one word ("list") — the cluster already says what of.

## 0.5.5

- Long action output (e.g. tools list) no longer renders duplicated leading
  chunks in some terminals: casa resets synchronized-output mode and the
  scroll region before printing, in case the TUI left them dangling.

## 0.5.4

- The "enter to go back" pause no longer echoes stray input: raw mode, only
  enter continues (ctrl+c still quits), everything else is swallowed silently.

## 0.5.3

- Internal: the generated script and template bodies are embedded files
  (internal/*/embedded/) instead of Go string constants — byte-identical
  output, no behavior change.

## 0.5.2

- Key restore on a new machine is skippable: the restore script asks
  "Restore age key ... [Y/n]" before prompting for the passphrase; declining
  (or a wrong passphrase, or no terminal) skips with a warning instead of
  aborting the whole apply. Only files that key guards fail to decrypt;
  restore later via casa secrets keys.

## 0.5.1

- Key backups in the repo are ASCII-armored (base64 text) instead of raw
  binary — displays and diffs sanely. Existing binary backups stay valid;
  re-run the backup action to convert.

## 0.5.0

- **Run scripts are casa-generated, never repo content**: the packages,
  sh-tools, and key-restore scripts no longer live in your repo — casa writes
  them into the source dir (gitignored, like the chezmoi-name mirrors) and
  refreshes them from the installed casa's embedded templates before every
  chezmoi call. Behavior always matches your casa version; script staleness
  across repos is impossible; repos carry only data. Existing committed
  copies: `git rm --cached` them and casa takes over.
- **Drift fix**: hand-managed `extra`/`extra_darwin` lines now count as
  recorded, so `tools import` can't re-add them as plain entries (a
  duplicate like `brew "ruby"` vs `brew "ruby", link: false` broke
  brew bundle at the link step).

## 0.4.0

- **Fresh machines are fully self-sufficient**: `machine setup` offers to
  install Homebrew after the questionnaire (declining just skips packages),
  and every casa-side file in a repo is now reproducible from nothing —
  manifest + run scripts, the `[age]` block, and key restore.
- **Key backup in the repo** (`secrets keys` → *backup to repo*): the private
  identity is passphrase-encrypted to `.casa/keys/<name>.key.age` (never a
  chezmoi target, safe to commit) and a generated `run_once` script restores
  backups into `~/.config/casa/keys` on new machines before anything needs
  decrypting. Deleting a key also deletes its backup, so a fresh apply never
  prompts for a dead key.

## 0.3.0

- **Registry-free keys**: a key IS a private identity file in
  `~/.config/casa/keys/<name>.txt`. No key names, paths, or recipients ever
  enter a repo — names are filenames, recipients derive from the files
  (`age-keygen -y`), the default is a local `.default` marker, and the config
  template's `[age]` block is one generic snippet (glob + derive at init).
  Replaces 0.2.0's committed `.casadata/keys.toml`; a legacy `~/key.txt`
  moves into the keys dir as `main` on first use. Trade-off: a machine can
  only encrypt to keys it holds.

## 0.2.0

- **Multi-key encryption** (`secrets keys`): create keys, adopt a legacy
  `~/key.txt`, pick a default, choose the key per secret, delete with orphan
  detection (files only that key opens are re-encrypted with a survivor
  first), and back private identities up to doppler. Public recipients live
  committed in `.casadata/keys.toml`; the config template's `[age]` block is
  generated with stat-filtered identities so machines holding only some keys
  still decrypt what they can. Editing a secret re-seals with the same key
  that sealed it.
- **Smart add**: paste any install command (`go install …`, `cargo install`,
  `npm -g`, `uv tool`, `brew`, `bun add -g`, `curl … | sh`) into the add flow —
  casa detects the manager, installs only if missing, and records it. Install
  directly in your terminal and the menu shows an `(N to record)` drift hint;
  `tools import` records exactly what's missing.
- **bun** is a full manager: add/remove/import/update -g, paste detection,
  and an idempotent `bun add -g` loop on apply (brew bundle can't drive bun).
- **Trusted taps** (`tools trust`): pick which taps brew bundle may manage
  without prompting (`tap "…", trusted: true`).
- **PATH self-healing**: manager bin dirs (go, cargo, bun, uv, brew incl.
  linuxbrew and keg-only rustup) are prepended when missing, so casa works
  under a minimal environment; `doctor` leads with a deps table.
- **Fresh machine**: plain `casa` bootstraps chezmoi itself (brew, or
  get.chezmoi.io) before setup — verified in an isolated Linux container.
- **UX**: clean-screen menu with grouped commands, consistent controls
  (tab select · enter submit · esc/← back), fzf-style path completion,
  `~/`-style paths everywhere, single-pick track.
- **Internals**: managers behind a `pm.Manager` interface (one file each),
  chezmoi and prompt seams for tests, one-topic files, `make ship`
  (fmt + lint + modernize + tests + pty e2e + install).

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
