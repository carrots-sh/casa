# Changelog

casa uses **semver**: `vMAJOR.MINOR.PATCH`, newest first. (Releases before
v0.1.0 used date-based versions, `vYYYY.MM.DD-N`; those tags and releases were
retired when the scheme changed — their entries remain below for history.)

## 0.17.0

- casa has its own identity now: **make any machine feel like home** — a
  machine manager for files, tools, and secrets, not "easy chezmoi". The
  help header, Homebrew formula, README, and all docs carry it; chezmoi is
  credited as the engine under the hood with the no-lock-in guarantee.
- Agent skills grew into a suite: `casa` (overview + router), `casa-files`,
  `casa-tools`, `casa-secrets`, `casa-machine` — detailed, source-verified,
  installable with `bunx skills add carrots-sh/casa`.
- `rm` works as a remove alias in every cluster (files/secrets too, not
  just tools).

## 0.16.1

- The menu got its own fzf-style widget: sections render once per cluster
  (no repeated gutter labels), and when you filter, each visible group still
  shows which section it belongs to — the gutter is computed per visible
  row, not baked into the text. Sizes itself to the terminal.

## 0.16.0

- `tools trust` covers every tap now: custom-URL taps declared as raw
  `extra` lines (like sst/tap) appear in the picker, and toggling them
  rewrites the line's `trusted: true` flag in place.
- Trust reaches brew's machine-level store too: the trust picker syncs
  `brew trust`/`brew untrust` for this machine, and the generated packages
  script converges trusted taps on every apply — so plain `brew upgrade`
  stops warning, not just bundle runs, on every machine.
- Update menu: self-updating sh tools (bun) are labeled
  "self-update (freshness unknown)" — they always appear because casa
  can't ask a self-updater whether it's outdated.

## 0.15.1

- Menu rows carry their section name on every row, not just the cluster's
  first — filtering keeps each action's context (filtering "add" shows which
  add is files/tools/secrets), and typing a section name filters to it.

## 0.15.0

- The actions ARE the directions now: `push` and `pull` replace save and
  sync everywhere — menu, commands, status (`to push: N` / `to pull: N`),
  and all output copy. `casa save` / `casa sync` remain as legacy aliases.

## 0.14.1

- The direction is in the copy now: the menu shows save as
  "push · your changes → repo" and sync as "pull · repo → this machine
  (pushes yours first)", and `casa push` / `casa pull` work as aliases.

## 0.14.0

- `casa sync` is bidirectional now: unsaved local changes are shown and
  offered as a push first, and files changed outside casa get a
  keep-or-restore review before the pull applies over them — the direction
  of every difference is an explicit choice, never an accident.

## 0.13.2

- A failing install no longer aborts the whole apply (found on a real fresh
  VPS: one dead formula killed the entire setup). brew bundle failures,
  sh-tool installer failures, and bun failures are warnings now — everything
  else keeps installing, and fixed entries retry on the next apply.
- The Homebrew formula depends on `age`, so key restore works on the very
  first apply instead of warning that age isn't installed yet.

## 0.13.1

- Generated run scripts renumbered contiguously now that repos need no
  hand-written ones: `00-casa-keys` (before) → `10-packages` → `20-sh-tools`
  (after). casa sweeps the old-numbered files and their gitignore entries
  from existing repos automatically — nothing runs twice, nothing to do.

## 0.13.0

- Repos need **zero hand-written run scripts** now:
  - `system = [...]` manifest section — Linux distro packages (apt/dnf/pacman,
    detected at apply) for things brew can't own: login shells, xclip, ...
    Ignored on macOS.
  - `[[packages.sh]]` blocks accept `creates = "$HOME/.oh-my-zsh"` — a path
    guard for installers that don't put a binary on PATH.
  - `install.sh` and `casa machine setup` install Homebrew's Linux
    prerequisites (build tools, curl, file, git) before Homebrew itself.

## 0.12.0

- `casa cd` opens a subshell inside your dotfiles repo, like `chezmoi cd` —
  exit to return to where you were.

## 0.11.0

- npm-registry search results install via **bun** by default: bun pulls
  from the same registry, so hits appear as bun rows (installed with
  `bun add -g`, recorded under `bun`). npm search only surfaces when bun
  isn't installed; npm remains available as an explicit manager pick.

## 0.10.1

- Status and the drift flow agree: "local drift" counts only reviewable
  files (what `files drift` shows); pending run scripts get their own
  status line and a note in the drift screen — they just run on the next
  sync, nothing to keep or restore.

## 0.10.0

- **fzf-style multiselects**: in remove, update, import, trust, and every
  other multi-pick, just type to filter — no `/` needed — space toggles the
  highlighted row, and selections persist while you type and erase across
  multiple searches. ctrl+a toggles everything visible; the footer counts
  what's selected.

## 0.9.2

- Key copy is consistent: space is the advertised select/toggle key
  everywhere (tab and x still work); titles no longer repeat key
  instructions — the help line is the single source of truth.

## 0.9.1

- Drift diffs print colored straight to the terminal (chezmoi's diff, no
  pager, no line-cursor buffer); pending run scripts no longer appear as
  "drifted files"; restoring the repo version forces past chezmoi's own
  overwrite re-prompt.

## 0.9.0

- **Drift review** (`files drift`, also in the menu with a background
  `(N drifted)` hint): walk the files whose on-disk state differs from the
  repo, view each diff inside the TUI pager, then keep your local version
  (records it + offers save) or restore the repo's — per file, skippable.

## 0.8.2

- Paste detection recognizes command-substitution installers
  (`sh -c "$(curl ...)"`, incl. env prefixes) — not just the piped
  `curl ... | sh` shape.

## 0.8.1

- The add search offers one escape row instead of two: "command · paste an
  install command" covers curl-pipe installers too (it routes them into the
  sh flow), so the redundant sh row is gone. casa tools add sh still exists.

## 0.8.0

- Noun clusters with ONE verb vocabulary: files / tools / secrets / machine,
  and add / edit / remove / list mean the same thing in every cluster — the
  synonym verbs are gone (track → files add, untrack → files remove,
  encrypt → secrets add). New: secrets remove. No "everything" views; each
  cluster lists its own. edit still handles encrypted files transparently.
  CLI: casa files … is the namespace (configs/track/untrack/rm remain
  as legacy aliases); tools remove replaces tools rm.

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
