# Tools

casa records every tool you install in a single manifest,
`.casadata/packages.toml`, committed in your dotfiles repo. On every machine,
applying your dotfiles installs what the manifest lists and uninstalls what you
removed — the manifest is the single source of truth for what's on a machine.

```bash
casa tools add [manager] [name]   # install a tool and record it
casa tools add sh                 # a tool that ships its own installer (curl | sh)
casa tools add cmd ["command"]    # paste any install command — casa detects the manager
casa tools rm                     # uninstall tool(s) — pick across all managers
casa tools update                 # upgrade outdated tools — one, many, or all
casa tools list                   # list recorded tools
casa tools import                 # seed the manifest from what's installed here
casa tools trust                  # pick which taps update without prompting
```

Every command that changes the manifest ends by offering to commit and push
(see [Machine](machine.md) for `casa save`).

## The manifest

The manifest lives at `.casadata/packages.toml` in your source directory.
chezmoi reads it through the `.chezmoidata` mirror symlink that casa maintains
(see [Repository layout](repo-layout.md)); repos that already have a real `.chezmoidata`
directory keep using `.chezmoidata/packages.toml` instead.

It is plain TOML: one array per package manager, plus `[[packages.sh]]` blocks
for tools that ship their own installer. Edit it by hand freely — casa edits it
line by line, so your comments and ordering survive.

```toml
[packages]

# Homebrew taps
taps = [
  "oven-sh/bun",
]

# Taps whose formulae brew bundle may manage without prompting
# (casa tools trust moves taps here)
taps_trusted = [
  "carrots-sh/tap",
]

# CLI tools — cross-platform (macOS + Linux via Homebrew)
brew = [
  "ripgrep",
  "jq",
]

# macOS-only formulae
brew_darwin = [
  "mas",
]

# macOS apps + fonts (casks)
cask = [
  "ghostty",
  "font-jetbrains-mono",
]

# go install
go = [
  "golang.org/x/tools/gopls",
]

# uv tool install
uv = [
  "ruff",
]

# npm install -g
npm = [
  "typescript",
]

# bun add -g
bun = [
  "prettier",
]

# cargo install
cargo = [
  "cargo-watch",
]

# Linux distro packages (apt/dnf/pacman — whichever the machine has) for
# things brew can't own: login shells, clipboard tools, ... Ignored on macOS.
system = [
  "zsh",
]

# Raw Brewfile lines passed through verbatim — for anything with extra
# arguments (custom tap URLs, link: false, trusted: true, mas/vscode
# directives, ...). Hand-managed; casa's add/remove don't touch these.
extra = [
  'brew "ruby", link: false',
]

# Same, but only rendered on macOS.
extra_darwin = [
  'mas "Xcode", id: 497799835',
]

# Tools that ship their own installer are recorded as blocks
[[packages.sh]]
bin = "herdr"
install = "curl -fsSL https://herdr.dev/install.sh | sh"
update = "herdr self-update"  # optional; omit if the tool updates itself
os = "darwin"                 # optional: darwin | linux
creates = "$HOME/.herdr"      # optional: path guard for installers with no binary
```

Section notes:

- `brew` entries install on both macOS and Linux (via Homebrew);
  `brew_darwin`, `cask`, and `extra_darwin` only render on macOS.
- `extra` and `extra_darwin` are raw Brewfile lines passed through verbatim.
  Use them for anything needing arguments casa's flat lists can't express.
  casa's `add`/`rm` never touch them, but they still count as recorded for
  drift detection, so a hand-managed `brew "ruby", link: false` never shows up
  in `casa tools import` as an unrecorded `ruby`.
- `system` entries install via the machine's distro package manager (apt, dnf,
  or pacman — detected at apply) and are ignored on macOS. Use them for the few
  things Homebrew can't own on Linux, like a login shell (`zsh`) or `xclip`.
  Hand-managed, like `extra`.
- Each `[[packages.sh]]` block needs `bin` (how casa detects the tool is
  installed) and `install` (the one-liner); `update` and `os` are optional.
  For installers that don't put a binary on PATH (Oh My Zsh installs a
  directory), set `creates` to the path the installer leaves behind — casa
  guards on that instead of `command -v`. A block can also express a
  post-install step: `bin = "cargo"` + `install = "rustup default stable"`
  initializes a toolchain only while `cargo` is missing.

## What happens on apply

The manifest is data, not a script. Before every chezmoi call, casa generates
two `run_onchange` scripts in the source directory — gitignored, never
committed — from templates embedded in the installed casa binary:

- `run_onchange_after_10-packages.sh.tmpl` — everything brew bundle can manage
- `run_onchange_after_20-sh-tools.sh.tmpl` — the `[[packages.sh]]` tools

Because they are `run_onchange` scripts, they re-run on apply exactly when the
rendered package list changes; because they are regenerated rather than
committed, install behavior always matches the casa version you have installed.

The packages script renders the manifest as a Brewfile and pipes it straight
into `brew bundle --file=-` — no Brewfile ever exists on disk. Taps, `brew`,
`cask`, `go`, `uv`, `npm`, and `cargo` entries all become Brewfile directives
(brew bundle drives the language package managers itself). Then
`brew bundle cleanup --force --file=-` uninstalls anything installed but no
longer declared: removing an entry from the manifest and applying is how you
uninstall by hand.

Two safeguards apply:

- **Empty-manifest guard.** An empty manifest renders an empty Brewfile, and
  `cleanup --force` against an empty Brewfile would uninstall everything. The
  script refuses to run against an empty list and skips instead.
- **Missing brew.** If `brew` isn't on `PATH`, the script prints a notice and
  exits cleanly rather than failing the apply.

`bun` globals are the exception — brew bundle can't manage them, so the script
re-asserts the list with an idempotent `bun add -g` loop (only when `bun` is
installed). There is no cleanup diffing for bun: removing a bun entry by hand
also needs a manual `bun remove -g`, which `casa tools rm` does for you.

The sh-tools script runs each `[[packages.sh]]` installer behind a
`command -v` guard, so re-runs are free and the script is idempotent. Blocks
with an `os` tag only run on that platform.

## Adding tools

### Search

With no arguments, `casa tools add` prompts for a query and searches every
manager with a usable search — brew, cask, npm, and cargo — in parallel:

```bash
casa tools add
```

Pick a result and casa installs it and records it in the matching manifest
section. The result list always ends with two extra rows — "sh" for tools with
their own installer and "command" for pasting an install command — so every
flow is reachable from plain `add`. Pasting an install command directly into
the search prompt also works: casa detects it and routes to the command flow.

### Direct

Name the manager (and optionally the package) to skip the search:

```bash
casa tools add brew ripgrep
casa tools add go golang.org/x/tools/gopls
```

Managers without a search — `go`, `uv`, `tap` — prompt for the package
directly.

### Paste an install command

`casa tools add cmd` takes the install one-liner from a project's README,
detects the manager and package, installs if needed, and records the result:

```bash
casa tools add cmd "go install golang.org/x/tools/gopls@latest"
```

Detected forms (a leading `sudo` is stripped):

| Command | Recorded as |
| --- | --- |
| `go install <path>@<ver>` / `go get <path>` | `go` (version suffix stripped) |
| `cargo install <pkg>` | `cargo` |
| `npm` / `pnpm` / `yarn` with `-g` / `--global` / `global` | `npm` |
| `bun add\|install\|i -g <pkg>` | `bun` |
| `uv tool install <pkg>` | `uv` |
| `brew install <pkg>` | `brew` |
| `brew install --cask <pkg>` | `cask` |
| `brew tap <tap>` | `taps` |
| `curl …` / `wget …` / anything piped to `sh` or `bash` | routed to the sh-installer flow |

If the package is already installed, casa records it without reinstalling.

### Tools with their own installer

`casa tools add sh` records a tool that ships as a `curl | sh` installer:

```bash
casa tools add sh
```

casa asks for the install one-liner, the binary name (guessed from the
installer URL, e.g. `https://herdr.dev/install.sh` → `herdr`), an optional
self-update command, and which platforms it runs on (all, macOS only, or Linux
only). If the binary is already on `PATH`, casa records it without re-running
the installer; otherwise it confirms and runs the one-liner, and warns if the
binary still isn't on `PATH` afterwards (the name is probably wrong). The
result is a `[[packages.sh]]` block, so every machine gets the tool on apply.

### First use

The first `add` (or `import`) on a repo without a manifest offers to set up
package management: casa writes a commented skeleton manifest, then offers to
seed it. If the repo has a legacy Brewfile setup (`dot_Brewfile.tmpl`,
`Brewfile`, …), casa imports its packages, deletes the Brewfile sources and
any old `brew bundle` run script, and unmanages `~/.Brewfile` — the old
rendered file must go, or its cleanup step would fight the manifest.
Otherwise casa offers to import everything already installed on the machine.
Declining still installs the tool; it just isn't recorded.

## Removing tools

`casa tools rm` shows one flat multi-select of everything recorded across all
managers. For each pick, casa uninstalls via the owning manager and removes
the manifest entry (if the uninstall errors, the entry is removed anyway). For
sh tools, casa removes the `[[packages.sh]]` block and — since the installer,
not casa, put the binary there — asks before deleting the binary itself.

## Updating tools

`casa tools update` lists what's outdated and upgrades your picks:

- **Per package** for brew, cask, and npm — each outdated package is its own
  row.
- **Self-update commands** for sh tools that declared an `update` command.
- **In bulk** for uv (`uv tool upgrade --all`) and cargo — these managers only
  upgrade everything at once, and their rows appear whenever something else
  has updates.

Selecting everything collapses the brew and cask rows into a single
`brew upgrade`. If nothing is outdated, `update` says so and exits.
`casa sync` also upgrades packages before pulling and applying — see
[Machine](machine.md).

## Import and drift

`casa tools import` finds drift: packages installed on this machine that the
manifest doesn't record. It multi-selects over the drift with everything
preselected, so pressing enter records it all:

```bash
casa tools import
```

Drift detection is per manager and best-effort — managers not installed on the
machine contribute nothing. Everything that installs through the same manager
counts as recorded, including the `brew_darwin` and `taps_trusted` sibling
sections and directives parsed out of `extra`/`extra_darwin` lines (tapped
formulae also match under their short name). The interactive menu surfaces
drift as an "(N to record)" hint on the import entry.

## Trusted taps

Formulae from third-party taps produce "tap formula is not trusted" prompts
under brew bundle. `casa tools trust` multi-selects which of your recorded
taps to trust; trusted taps move to the `taps_trusted` section and render as
`tap "…", trusted: true`, so their formulae install and clean up without
prompting. Untrusting a tap moves it back to `taps`.

## Listing

`casa tools list` prints everything recorded, grouped by section. sh tools
whose binary isn't on `PATH` are marked `(not installed here)` — recorded in
the repo, but not yet applied on this machine.
