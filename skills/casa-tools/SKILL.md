---
name: casa-tools
description: Operate casa's package manifest (.casadata/packages.toml) — the one TOML file that declares every tool on a machine (brew, cask, taps, go, uv, npm, bun, cargo, distro packages, self-installing sh tools) and converges on apply, installing what is listed and UNINSTALLING what was removed. Use when adding, removing, updating, or importing packages in a casa-managed repo, editing packages.toml, trusting taps, or debugging why a package appeared or vanished on casa pull. Skip for dotfiles or secrets work (use the casa skill) and for repos not managed by casa.
license: MIT
---

# casa tools — the package manifest

casa manages your machines — files, tools, and secrets — from one git repo, with two verbs: push and pull. This skill covers the tools third in depth: the package manifest, what applying it does, and the exact commands to drive it.

One manifest declares every package on every machine. Applying converges instead of accumulates: what is listed gets installed, what you removed gets **uninstalled**. There is no Brewfile, ever — never create one.

## Where the manifest lives

- Primary: `.casadata/packages.toml` in the chezmoi source dir (find it with `casa cd` or `chezmoi source-path`).
- Fallback: `.chezmoidata/packages.toml`, only in repos that already had a real (non-symlink) `.chezmoidata` directory.

Under the hood, chezmoi renders and applies files, and it reads the manifest as template data through a gitignored `.chezmoidata` symlink that casa maintains. Commit only `.casadata/packages.toml`; never commit the symlink.

The file is plain TOML and safe to hand-edit: casa edits it line by line, so comments and ordering survive. But a hand-edit is live — the next apply (`casa pull`) really uninstalls whatever you deleted.

## Every section

All under one `[packages]` table. String-list sections, in display order:

| Section | Installed via | Notes |
| --- | --- | --- |
| `taps` | `brew tap` | Homebrew taps, standard `user/repo` form |
| `taps_trusted` | `brew tap` | Taps brew bundle may manage without prompting; `casa tools trust` moves taps here. Renders as `tap "…", trusted: true` AND is synced into brew's machine trust store (`brew trust`) |
| `brew` | Homebrew | Cross-platform CLI tools — installs on macOS and Linux |
| `brew_darwin` | Homebrew | macOS-only formulae (e.g. `mas`) |
| `cask` | Homebrew cask | macOS apps and fonts; only rendered on macOS |
| `go` | `go install` (via brew bundle's `go` directive) | Full module paths: `golang.org/x/tools/gopls` |
| `uv` | `uv tool install` | Python CLI tools |
| `npm` | `npm install -g` | |
| `bun` | `bun add -g` | Special-cased on apply — see below |
| `cargo` | `cargo install` | |
| `system` | apt / dnf / pacman — whichever the machine has | Linux distro packages brew can't own (login shells, `xclip`). Ignored on macOS. Hand-managed |
| `extra` | verbatim | Raw Brewfile lines passed through as-is — for anything needing arguments: custom tap URLs, `link: false`, `mas`/`vscode` directives. Hand-managed; casa's add/remove never touch these |
| `extra_darwin` | verbatim | Same, but only rendered on macOS |

Plus `[[packages.sh]]` blocks for tools that ship their own installer:

```toml
[[packages.sh]]
bin = "herdr"                                       # binary name — install detection via command -v
install = "curl -fsSL https://herdr.dev/install.sh | sh"
update = "herdr self-update"   # optional; omit if the tool updates itself
os = "darwin"                  # optional: "darwin" | "linux"; omit = all platforms
creates = "$HOME/.herdr"       # optional: path guard for installers with no binary
```

`bin` and `install` are required. `creates` replaces the `command -v` guard for installers that leave a directory instead of a binary (Oh My Zsh → `creates = "$HOME/.oh-my-zsh"`). A block can also express a one-time post-install step: `bin = "cargo"` + `install = "rustup default stable"` runs only while `cargo` is missing.

Realistic manifest excerpt:

```toml
[packages]

taps = [
  "oven-sh/bun",
]

taps_trusted = [
  "carrots-sh/tap",
]

brew = [
  "ripgrep",
  "jq",
  "showwin/speedtest/speedtest",   # tap-scoped formulae need the FULL path
]

brew_darwin = [
  "mas",
]

cask = [
  "ghostty",
  "font-jetbrains-mono",
]

go = [
  "golang.org/x/tools/gopls",
]

uv = [
  "ruff",
]

system = [
  "zsh",
]

extra = [
  'brew "ruby", link: false',
]

extra_darwin = [
  'mas "Xcode", id: 497799835',
]
```

Note the quoting: `extra`/`extra_darwin` entries are single-quoted TOML strings containing double-quoted Brewfile syntax.

## What applying does

The manifest is data; repos carry only data. Before every chezmoi call, casa generates two `run_onchange` scripts in the source dir from templates embedded in the installed casa binary — **gitignored, never committed** (never commit them, never hand-edit them; they are regenerated):

- `run_onchange_after_10-packages.sh.tmpl` — everything brew bundle can manage
- `run_onchange_after_20-sh-tools.sh.tmpl` — the `[[packages.sh]]` tools

They re-run on apply exactly when their rendered content changes, i.e. when the relevant part of the manifest changes. On apply (part of `casa pull`):

1. **Linux system packages** first: `system` entries go to apt-get / dnf / pacman, whichever exists, with sudo if needed. Skipped entirely on macOS.
2. **Trusted taps** are asserted in brew's machine trust store: `brew trust "<tap>"` per `taps_trusted` entry (errors from older brews ignored).
3. **The Brewfile pipe**: the manifest renders as a Brewfile in memory — `tap`, `brew`, `cask` (macOS only), `extra`/`extra_darwin` verbatim, plus `go`, `uv`, `npm`, `cargo` directives (brew bundle drives those language managers itself) — and is piped straight into `brew bundle --file=-`. **No Brewfile ever touches disk.**
4. **Cleanup**: `brew bundle cleanup --force --file=-` uninstalls anything installed but no longer declared. Removing a manifest entry + apply = uninstall. This is the intended way to uninstall by hand-editing.
5. **bun re-assert loop**: brew bundle can't manage bun globals, so the script runs idempotent `bun add -g` per entry (only when `bun` is on PATH). No cleanup diffing for bun — removing a bun entry by hand also needs a manual `bun remove -g` (`casa tools remove` does both).
6. **sh tools**: each `[[packages.sh]]` installer runs behind its guard — `! command -v <bin>` or `[ ! -e <creates> ]` — so re-runs are free. `os`-tagged blocks only run on that platform.

Safeguards baked into the packages script:

- **Empty-manifest guard**: an empty manifest renders an empty Brewfile, and `cleanup --force` against that would uninstall everything — the script refuses and skips instead.
- **Missing brew**: prints a notice and exits 0 rather than failing the apply.
- **Failing entries** (dead formula, network blip) warn and continue; the rest of the setup proceeds, and the script retries whenever the list next changes.

## Commands

```
casa tools add [manager] [name]   # install + record
casa tools add sh                 # tool with its own curl|sh installer (interactive)
casa tools add cmd ["command"]    # paste an install one-liner; casa detects the manager
casa tools remove                 # multi-select uninstall (interactive)
casa tools update                 # upgrade outdated picks (interactive)
casa tools list                   # print everything recorded (plain, pipeable)
casa tools import                 # record installed-but-unrecorded drift (interactive)
casa tools trust                  # pick trusted taps (interactive)
```

Legacy alias: `rm` = `remove`. Every command that changes the manifest ends by offering to commit and push.

### Non-interactive forms (use these as an agent)

Fully non-interactive when both manager and name are given:

```bash
casa tools add brew ripgrep
casa tools add brew showwin/speedtest/speedtest    # tap-scoped: FULL path required
casa tools add cask ghostty
casa tools add go golang.org/x/tools/gopls
casa tools add uv ruff
casa tools add npm typescript
casa tools add bun prettier
casa tools add cargo cargo-watch
casa tools add tap oven-sh/bun
casa tools add cmd "go install golang.org/x/tools/gopls@latest"
```

`add` installs first, then records; expect `installing <name> via <mgr>...` then `✓ recorded: <section> "<name>"`, then a save/push offer (`CASA_YES=1` auto-answers yes to confirmation prompts; it does NOT satisfy pickers or text prompts). `add cmd` records without reinstalling when the tool is already present; the plain `add <manager> <name>` form always runs the installer (harmless re-install for brew, a real rebuild for go/cargo). Manager names: `brew`, `cask`, `tap`, `go`, `uv`, `npm`, `bun`, `cargo` (`tap` records into `taps`; `brew` on macOS records into `brew`, not `brew_darwin` — move macOS-only entries there by hand if the user also runs Linux).

`add cmd` parses README one-liners (leading `sudo` stripped): `go install <path>@<ver>` (version stripped) → `go`; `cargo install` → `cargo`; `npm`/`pnpm`/`yarn` global → `npm`; `bun add|install|i -g` → `bun`; `uv tool install` → `uv`; `brew install [--cask]` → `brew`/`cask`; `brew tap` → `taps`; `curl`/`wget` piped to `sh`/`bash` → routes to the sh-installer flow (which then prompts).

### Interactive-only (do NOT script these)

`casa tools add` with no args (search picker), `add sh`, `remove`, `update`, `import`, `trust`, and the first-use manifest bootstrap prompt all use TUI selectors. As an agent, prefer editing the manifest directly for removals and bulk changes, then let the next `casa pull` converge — or tell the user to run the interactive command.

For agent-driven removals: edit `.casadata/packages.toml`, delete the entry (or the whole `[[packages.sh]]` block), and run `casa pull`. Cleanup uninstalls it. Exception: bun entries also need `bun remove -g <pkg>`.

### First use

The first `add`/`import` on a repo without a manifest offers (interactively) to bootstrap: writes a commented skeleton, then offers to seed it — from a legacy Brewfile setup if one exists (imports its packages, deletes `dot_Brewfile*`/`Brewfile*` sources and old brew-bundle run scripts, unmanages `~/.Brewfile`; the old rendered file must go or its cleanup fights the manifest), otherwise from everything already installed on the machine. Declining still installs, just doesn't record.

## Import and drift

`casa tools import` finds packages installed on this machine that the manifest doesn't record, multi-selects with everything preselected. Best-effort per manager: managers not on PATH contribute nothing. Everything installing via the same manager counts as recorded — sibling sections (`brew_darwin`, `taps_trusted`) and directives parsed out of `extra`/`extra_darwin` lines, with tap-scoped paths also matching under their short name — so a hand-managed `brew "ruby", link: false` never shows up as unrecorded `ruby`. The interactive menu shows drift as an "(N to record)" hint on import.

## Trust model

Third-party tap formulae trigger "tap formula is not trusted" prompts under brew bundle. Two layers:

1. **Manifest**: `casa tools trust` moves taps between `taps` and `taps_trusted`; trusted taps render as `tap "…", trusted: true`, covering bundle runs on every machine. Taps declared as raw `extra` lines (the only way to express a custom clone URL) get `, trusted: true` appended inside the single-quoted line instead.
2. **Machine**: brew keeps its own trust store that the Brewfile directive never touches — plain `brew install`/`brew upgrade` warn from it. `casa tools trust` also converges this machine's store via `brew trust`/`brew untrust`; the apply script re-asserts `brew trust` per `taps_trusted` entry so every machine goes quiet. Older brews without `brew trust` just error, ignored.

## Update semantics

`casa tools update` lists outdated and upgrades picks:

- Per package for brew, cask, npm (real outdated detection).
- sh tools with an `update` command are **always** listed, labeled "self-update (freshness unknown)" — self-updaters can't report freshness.
- uv and cargo only upgrade in bulk (`uv tool upgrade --all`, all cargo packages); their rows appear only when something else has updates.
- Selecting everything collapses brew+cask into one `brew upgrade`.

`casa pull` also upgrades packages as part of its converge (push yours first → review drift → upgrade → apply).

## Guardrails

- **Never create a Brewfile.** The manifest is the only package record; the Brewfile exists only as an in-memory render piped to `brew bundle --file=-`.
- **Hand-edits are real**: deleting a manifest entry uninstalls it on the next `casa pull`. Confirm before removing entries the user might still want.
- **`extra`/`extra_darwin`/`system` are hand-managed**: casa's add/remove never touch them. Edit them directly; keep the single-quote-outside, double-quote-inside TOML form for extras.
- **Never commit** the generated `run_onchange_after_*` scripts or the `.chezmoidata` mirror symlink — both gitignored, both regenerated by casa.
- **Tap-scoped formulae need full paths** in `brew` (`showwin/speedtest/speedtest`), and the tap itself belongs in `taps`/`taps_trusted`.
- After manifest edits, land them with `casa push` (or accept the offered save) so other machines pick them up on their next `casa pull`.
