# casa

make any machine feel like home.

casa manages your machines — files, tools, and secrets — from one git repo,
with two verbs: push and pull. One repo holds your dotfiles, one manifest
declares every package, and your secrets travel age-encrypted with keys that
never touch a repo. Every action lives in a status-aware interactive menu —
pick, confirm, done — and every menu action is also a typed command for
scripts and muscle memory.

## The promise

casa converges instead of accumulates, and never locks you in.

- **One repo, three things.** Dotfiles and templates, a single
  `.casadata/packages.toml` manifest covering brew, casks, taps, distro
  packages, go, uv, npm, bun, cargo, and self-installing tools, and
  age-encrypted secrets whose keys live in `~/.config/casa/keys` — plain
  files on your machine, never in a repo.
- **Pull actually converges.** `casa pull` pushes your changes first, reviews
  drift, upgrades packages, and applies — and removing an entry from the
  manifest uninstalls it on the next apply. Run scripts (package install,
  sh-tools, key restore) are generated fresh from the installed casa binary,
  gitignored, never committed. Repos carry only data.
- **No lock-in.** Under the hood, [chezmoi](https://chezmoi.io) renders and
  applies your files — and your repo stays a valid chezmoi repo, so you can
  leave casa at any time and keep everything.

See [Repository layout](repo-layout.md) for the details of this arrangement.

## Documentation

| Page | Description |
| --- | --- |
| [Installation](installation.md) | Install casa with the bootstrap one-liner, Homebrew, release binaries, or `go install`, and keep it updated with `casa upgrade`. |
| [Getting started](getting-started.md) | Three starting points: from nothing, provisioning a new machine, or adopting casa in an existing chezmoi repo. |
| [Configs](configs.md) | Track, edit, and untrack dotfiles; choose plain, template, or encrypted storage and convert between them. |
| [Tools](tools.md) | Record every installed tool in one `.casadata/packages.toml` manifest — Homebrew, casks, taps, go, uv, npm, bun, cargo, and self-installing `curl \| sh` tools. |
| [Secrets](secrets.md) | Encrypt files with age, manage keys without a registry, back keys up to your repo, and sync them through Doppler. |
| [Machine](machine.md) | Provision a new machine from your repo, pull, push with auto commit messages, change setup answers, undo, and run health checks. |
| [Repository layout](repo-layout.md) | How casa's repo format maps onto its chezmoi engine: casa-named special files, self-healing symlinks, generated run scripts, and using your repo without casa. |
| [Reference](reference.md) | Every command, environment variable, keyboard control, and the `.casa.toml` config file. |
| [Design](design.md) | The architecture and the reasoning behind casa's main decisions. |

## Quickstart

A fresh machine is one curl away from being yours. One line installs Homebrew
if needed, installs casa, and offers to set everything up from your repo:

```bash
curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <your-github-username>
```

Or install just the binary and set up explicitly:

```bash
brew install carrots-sh/tap/casa
casa machine setup <your-github-username>
```

`setup` accepts a GitHub username (resolved to `<user>/dotfiles`), a
`user/repo`, or a full URL. It clones your repo, asks your repo's setup
questions in casa's UI, offers to install your recorded tools, and restores
your encryption key if you backed one up.

From then on, just run:

```bash
casa
```

The menu tells you what needs doing — unsaved changes, available updates, a
machine behind its repo — and walks you through it. The common actions are
also top-level shortcuts:

```bash
casa edit [name]     # pick and edit a config
casa push [msg]      # your changes → repo
casa pull            # repo → this machine, pushing yours first
casa status          # what's changed, behind, or outdated
```

casa runs on macOS and Linux. It stores your dotfiles in
`~/.local/share/casa` by default (override with `$CASA_SOURCE`); an existing
`~/.local/share/chezmoi` setup keeps working unchanged. Run `casa help` for
the full command listing.
