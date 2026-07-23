# casa

[![Release](https://img.shields.io/github/v/release/carrots-sh/casa)](https://github.com/carrots-sh/casa/releases)
[![License](https://img.shields.io/github/license/carrots-sh/casa)](LICENSE)

An interactive front-end for [chezmoi](https://chezmoi.io). Your dotfiles,
tools, and secrets — one friendly menu, nothing to memorize.

casa never reimplements chezmoi; it shells out to it, so your dotfiles repo
stays a plain chezmoi repo you own.

## Highlights

- 🏠 **One menu for everything**: edit configs, install tools, manage secrets,
  sync machines — grouped, filterable, single keypress each.
- 📦 **One package manifest**: everything you install lands in
  `.casadata/packages.toml`; every machine converges on it via `chezmoi apply`
  (brew, casks, taps, `go`, `uv`, `npm`, `bun`, `cargo`, and `curl | sh` installers).
- 📋 **Paste any install command**: `go install …`, `cargo install …`,
  `bun add -g …`, `curl … | sh` — casa detects the manager and records it.
  Install directly in your terminal and casa notices the drift.
- 🔐 **Encryption with zero key metadata in the repo**: keys are files in
  `~/.config/casa/keys/`; recipients are derived, never stored. Multiple keys,
  per-file choice, safe deletion with orphan re-encryption, passphrase-sealed
  repo backups, doppler support.
- 🧰 **Repos carry data, casa carries behavior**: the run scripts that install
  packages and restore keys are generated from the casa you're running —
  they never live in (or go stale in) your repo.
- 🖥️ **Fresh machines from nothing**: one command installs chezmoi, Homebrew,
  your key, and every tool — a passphrase is all you carry.
- ⌨️ **Consistent controls**: type to filter, `tab` select, `enter` submit,
  `esc`/`←` back. Paths shown as `~/…` everywhere.

## Installation

One line on a brand-new machine (installs Homebrew if needed, installs casa,
and sets everything up from your dotfiles):

```bash
curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <your-github-username>
```

Or just the binary:

```bash
# Homebrew
brew install carrots-sh/tap/casa

# Go
go install github.com/carrots-sh/casa/cmd/casa@latest
```

Release binaries for macOS and Linux (arm64/amd64, plus `.deb`/`.rpm`) are on
the [releases page](https://github.com/carrots-sh/casa/releases). casa updates
itself with `casa upgrade`.

## Quickstart

```bash
casa
```

On a machine with nothing set up, that single command walks you through
everything: installing chezmoi, cloning your repo (or starting one), your
setup questions, Homebrew, and key restore. On a working machine it opens the
menu:

```
casa · your-machine

files     edit         · pick + edit a file — encrypted handled
          list         · managed files
          add          · start managing a file
          storage      · how a file is stored
          remove       · stop managing a file
tools     add          · install a tool (search or paste)
          list         · recorded tools
          update       · upgrade outdated tools   (3 updates)
          import       · record what's installed here
secrets   edit · list · add · keys · remove
machine   save         · publish your changes     (2 to save)
          sync         · update this machine
          status · answers · question · undo · setup · doctor · info
casa      upgrade · quit
```

One verb vocabulary everywhere: `add`, `edit`, `remove`, `list` mean the same
thing in every cluster — no synonym verbs to learn (tracking a file is just
`files add`). `edit` handles encrypted files transparently.

Everything in the menu is also a typed command:

```
casa files    edit [name] | add [path] | storage [name] | remove [path] | list
casa tools    add [mgr] [name] | add sh | add cmd ["command"] | remove | update | list | import | trust
casa secrets  add [path] | edit [name] | remove | keys | list
casa machine  setup [repo] | sync | save [msg] | status | answers [name] | question | doctor | info
```

(`configs`/`track`/`untrack`/`rm` still work as legacy aliases.)

## Features

### Tools

Every tool you install through casa is recorded in one hand-editable manifest;
`chezmoi apply` converges any machine onto it (installs what's missing,
removes what you deleted). Search across managers, paste a full install
command, or record a `curl | sh` installer with its own update command:

```bash
casa tools add                       # search brew/cask/npm/cargo, or paste a command
casa tools add cmd "go install golang.org/x/tools/gopls@latest"
casa tools import                    # record things you installed directly
```

See [docs/tools.md](docs/tools.md).

### Secrets & keys

Files are encrypted with [age](https://age-encryption.org); private keys live
in `~/.config/casa/keys/` and nothing about them — names, paths, recipients —
ever enters a repo. Create keys, pick which one seals each secret, delete keys
safely (casa finds files only that key can open and re-encrypts them first),
and carry keys between machines via a passphrase-sealed repo backup, `scp`,
or doppler.

```bash
casa secrets add ~/.aws/credentials
casa secrets keys                    # create · default · backup · delete · doppler
```

See [docs/secrets.md](docs/secrets.md).

### Configs & templates

Track a file and casa asks how to store it — plain, template (per-machine
values auto-substituted), encrypted, or both — with sensible defaults
detected from the file itself. Your repo's setup questions are asked in casa's
UI and become `{{ .variables }}` usable in any template:

```bash
casa configs track ~/.gitconfig      # heuristics suggest template (it has your email)
casa machine question                # add a setup question, use {{ .key }} anywhere
casa machine answers                 # change an answer, re-render this machine
```

See [docs/configs.md](docs/configs.md).

### Machines

```bash
casa machine setup <user>            # provision from <user>/dotfiles
casa pull                            # upgrade packages, pull, apply
casa push                            # commit + push (auto-written message)
casa machine doctor                  # deps table + chezmoi doctor
```

See [docs/machine.md](docs/machine.md) and
[docs/getting-started.md](docs/getting-started.md) for the fresh-VPS
walkthrough.

## Agent skill

Teach your coding agent to drive casa (Claude Code, Cursor, Codex, and
~50 others auto-detected):

```bash
bunx skills add carrots-sh/casa
```

The skill ([skills/casa/SKILL.md](skills/casa/SKILL.md)) covers the
repo-as-source-of-truth model, the agent-safe command forms, and the
guardrails (no Brewfiles, no chezmoi-named commits, secrets stay human).

## Documentation

Full documentation lives in [docs/](docs/):
[getting started](docs/getting-started.md) ·
[installation](docs/installation.md) ·
[tools](docs/tools.md) ·
[secrets & keys](docs/secrets.md) ·
[configs](docs/configs.md) ·
[machine](docs/machine.md) ·
[repo layout](docs/repo-layout.md) ·
[CLI reference](docs/reference.md) ·
[design](docs/design.md)

## Design

- Repos commit **casa-named** special files (`.casa.toml.tmpl`, `.casaignore`,
  `.casadata/`); casa maintains gitignored symlinks to the names chezmoi
  expects, self-healing before every chezmoi call.
- Repos carry **only data**. The run scripts that turn the manifest into
  installs are generated from the installed casa's templates — behavior
  always matches your casa version.
- Your repo remains a working chezmoi repo: files, templates, and encryption
  are all native chezmoi. See [docs/design.md](docs/design.md) for the
  trade-offs, honestly stated.

## Versioning

Semver: `vMAJOR.MINOR.PATCH`. See [CHANGELOG.md](CHANGELOG.md).

## License

MIT
