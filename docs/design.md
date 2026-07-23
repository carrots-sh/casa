# Design

casa manages your machines — files, tools, and secrets — from one git repo, with two verbs: push and pull. This page explains the architecture and the reasoning behind its main decisions.

## Never reimplement the engine

Under the hood, [chezmoi](https://www.chezmoi.io/) renders and applies your files. casa shells out to it for every state operation and never reimplements its behavior. `internal/chez` is a thin wrapper: each function builds a `chezmoi --source <dir> ...` command and streams it to the terminal. The user's repo stays the source of truth — and stays a valid chezmoi repo, so anything chezmoi can do remains available by dropping to `chezmoi` directly, and you can leave casa at any time and keep everything.

The same rule applies to packages. casa does not track install state or compute diffs itself — it renders the manifest to Brewfile-format text and pipes it to `brew bundle --file=-`, then `brew bundle cleanup --force --file=-`. No Brewfile ever exists on disk. Homebrew does the diffing: whatever is installed but no longer declared gets uninstalled. casa's job is to keep one manifest honest, not to be a package manager.

The source directory defaults to `~/.local/share/casa` (override with `CASA_SOURCE`). For backward compatibility, if that directory is not a repo but chezmoi already has a configured source, casa uses it — an existing chezmoi setup works unchanged.

## One manifest as data, generated scripts as behavior

Tool state lives in a single committed file, `.casadata/packages.toml`. The scripts that act on it — package install, sh-tool install, key restore — are **not** repo content. casa writes them into the source directory gitignored, regenerated from templates embedded in the casa binary before every chezmoi call (`internal/app/generated.go`).

This split exists because committed scripts go stale. A script committed a year ago encodes the behavior of the casa that wrote it; every machine that clones the repo replays that old behavior, and fixing a bug means touching every repo. With generated scripts, behavior always matches the *installed* casa version: upgrade casa, and the next `apply` runs the current logic on every machine. Repos carry only data.

The generated scripts are chezmoi `run_onchange` scripts, so they re-run on `apply` exactly when the rendered package list changes. The package script includes an empty-manifest guard — `brew bundle cleanup --force` against an empty package list would uninstall everything, so an empty render skips the run entirely.

See [tools](tools.md) for the manifest format.

## casa names with self-healing mirrors

Repos commit chezmoi's special files under casa names: `.casa.toml.tmpl` (the setup questionnaire), `.casaignore`, `.casadata/`, and so on. chezmoi hardcodes its own names, so casa maintains gitignored symlinks (`.chezmoi.toml.tmpl` → `.casa.toml.tmpl`, etc.) and recreates any missing link before every chezmoi invocation (`internal/chez/mirrors.go`). The check is a handful of `Lstat` calls, so it is free to run every time.

The mirror is one-way and non-destructive: a repo that uses chezmoi names directly is left untouched, and a user's real chezmoi-named file is never overwritten by a link.

The trade-off is deliberate: the repo reads as a casa repo, at the cost of one indirection that casa repairs automatically. The known consequence is covered under [limitations](#limitations).

## Registry-free keys

casa's age key management has no registry, no config file, and no repo state. A key **is** a private identity file: `~/.config/casa/keys/<name>.txt`. Everything else is derived:

- **Names** are filenames. Listing keys is reading the directory.
- **Recipients** are derived on demand with `age-keygen -y <identity>`.
- **The default key** is a local `.default` marker file, falling back to the first key alphabetically.

Nothing key-related is committed to a repo (passphrase-sealed backups under `.casa/keys/` are opt-in and useless without the passphrase). The generated `[age]` config block is fully generic — it globs the keys directory and derives the recipient at `init` time — so the same repo works on any machine with any set of keys.

The same philosophy applies to which key encrypted a file: casa does not record it. When you edit a secret, casa **probes** — it tries each held key with `age --decrypt` until one opens the file — and re-seals with that same key. Probing instead of tracking means there is no mapping to drift out of sync with reality.

The trade-off: you can only encrypt to keys you hold, because the recipient is derived from the private identity file. casa cannot encrypt "for the laptop" from a machine that doesn't have the laptop's key. In practice keys are shared across machines via passphrase-sealed repo backups or Doppler, so machines hold the keys they need. See [secrets](secrets.md).

## Package managers: a small interface, optional capabilities

`internal/pm` defines one interface every manager implements:

```go
type Manager interface {
    Name() string
    Install(pkg string) error
    Uninstall(pkg string) error
    Installed() []string // best-effort, for `tools import`
}
```

Capabilities that not every manager has are optional interfaces, asserted only where they are used:

- `Searcher` — a usable CLI search (brew, cask, npm, cargo). Managers without one (go, uv tools) take a package name directly.
- `Outdater` — per-package outdated reporting and upgrade (brew, cask, npm).
- `BulkUpgrader` — managers that can only upgrade everything at once (uv, cargo, bun).

The UI reflects reality instead of papering over it: `SearchAll` fans out to every `Searcher` in parallel; the update flow offers per-package choices only where the manager supports them. Adding a manager means implementing four methods plus whichever capabilities its CLI genuinely has.

## Testing: e2e first

casa is mostly glue between a terminal UI and external CLIs, so the highest-value test is the whole thing running for real. `scripts/e2e.sh` exercises every casa action against a sandboxed `HOME` with a real chezmoi, a real git remote (a local bare repo), and real age encryption. Interactive forms are driven through a pty with `expect(1)`, asserting on the actual rendered screens.

The sandbox is strict: `HOME`, `CASA_SOURCE`, `EDITOR`, `PATH`, and git config all point into a temp dir, and brew is deliberately kept off `PATH` (with `CASA_PLAIN_PATH=1` so casa doesn't re-add real manager directories) — the suite can never touch your real dotfiles or packages.

Unit tests cover the pure logic (manifest parsing, prompt parsing, drift computation), with `chez.SetExec` as the one seam for faking chezmoi. The release gate runs everything:

```console
$ make ship   # fmt + lint + modernize check + unit tests + pty e2e + install
```

Releases are semver (`vMAJOR.MINOR.PATCH`), cut via a GitHub Actions `workflow_dispatch` with a bump input.

## Limitations

Honest edges of the current design:

- **Bare chezmoi on a fresh clone.** The chezmoi-named symlinks are gitignored, so a fresh `git clone` followed by bare `chezmoi init`/`apply` won't see the questionnaire or ignore file until casa runs once (any casa command recreates the links), or you create the three symlinks by hand. Cloning through `casa machine setup` avoids this entirely.
- **No cleanup diffing for bun.** `brew bundle` cannot manage bun globals, so the generated script just re-asserts `bun add -g` for each listed package (idempotent and fast). Removing a bun entry by hand-editing the manifest does not uninstall it; `casa`'s remove flow runs `bun remove -g` for you.
- **sh tools are invisible to drift detection.** Tools declared in `[[packages.sh]]` blocks install behind `command -v` guards; there is no manager to enumerate what's installed, so `tools import` cannot discover an sh-installed tool that was never recorded. Record them when you install them.
