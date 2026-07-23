# casa

casa is an interactive front-end for [chezmoi](https://chezmoi.io). It manages
your dotfiles, secrets, and installed tools from a single status-aware menu —
pick, confirm, done — and every menu action is also a typed command for scripts
and muscle memory. casa never reimplements chezmoi: it shells out to it for
every operation, so chezmoi remains the engine and your dotfiles repo remains
the single source of truth.

## The promise

Your repo stays a plain chezmoi repo.

- **Files work with or without casa.** Everything casa commits — dotfiles,
  templates, encrypted files, the `.casa.toml.tmpl` questionnaire, the
  `.casadata/packages.toml` manifest — is ordinary chezmoi content. casa
  maintains gitignored symlinks from the casa-named special files to the names
  chezmoi hardcodes, recreated before every chezmoi call; to use a casa repo
  with bare chezmoi, create those three symlinks by hand once per clone.
- **casa behavior needs casa.** Package installation on apply, the interactive
  questionnaire, key backup and restore, and the menus are provided by the casa
  binary — the run scripts that drive them are generated fresh from the
  installed casa version and never committed. Without casa you keep all your
  files; you lose the automation.

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
| [Repository layout](repo-layout.md) | How casa maps onto chezmoi: casa-named special files, self-healing symlinks, generated run scripts, and using your repo without casa. |
| [Reference](reference.md) | Every command, environment variable, keyboard control, and the `.casa.toml` config file. |
| [Design](design.md) | The architecture and the reasoning behind casa's main decisions. |

## Quickstart

On a brand-new machine, one line installs Homebrew if needed, installs casa,
and offers to set everything up from your dotfiles:

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

casa stores your dotfiles in `~/.local/share/casa` by default (override with
`$CASA_SOURCE`); an existing `~/.local/share/chezmoi` setup keeps working
unchanged. Run `casa help` for the full command listing.
