# casa

An easy, **interactive** front-end for [chezmoi](https://chezmoi.io). Manage your
dotfiles, secrets, and tools from one friendly menu — nothing to memorize.

casa never reimplements chezmoi; it shells out to it, so your dotfiles repo stays
the single source of truth and works with or without casa.

## Install

One line on a brand-new machine — installs Homebrew if needed, installs casa, and
(optionally) sets everything up from your dotfiles:

```bash
curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <your-github-username>
```

Or just the binary:

```bash
brew install carrots-sh/tap/casa
```

## Use it

Just run:

```bash
casa
```

You get a status-aware menu — it tells you what needs doing (unsaved changes,
updates available, machine behind your repo) and walks you through everything:

```
casa · your-machine

> Configs  · edit your dotfiles
  Tools    · install, remove, update      (5 updates)
  Secrets  · encrypted files
  Sync     · pull latest onto this machine
  Save     · publish your changes         (2 to save)
  Status   · full overview
  Machine  · contexts, info, health
  Quit
```

Every step is a list or a prompt — pick, confirm, done. Encrypted files are
handled transparently; nothing destructive happens without a confirmation.

### First five minutes

```bash
brew install carrots-sh/tap/casa
casa machine setup <your-github-username>   # or: casa → machine → set up this machine
casa                                        # from then on: pick what you want
```

`setup` takes a github **username** (→ `<user>/dotfiles`), a `user/repo`, or a full
URL. It prefers **SSH**, falls back to **HTTPS**, and errors only if both fail.

casa stores your dotfiles in **`~/.local/share/casa`** by default (override with
`$CASA_SOURCE`). Existing `~/.local/share/chezmoi` setups keep working unchanged.

## Typed commands (optional — for scripts & muscle memory)

Everything in the menu is also a namespaced command:

```
casa tools    add [mgr] [name] | rm | update | list | import
casa configs  edit [name] | track [path] | storage [name] | untrack [path] | list
casa secrets  add [path] | edit | list
casa machine  setup [repo] | sync | save [msg] | status | answers [name] | question | doctor | info
```

## Templates & setup questions

Tracking a file asks how to store it: **plain**, **template** (differs per
machine — casa auto-fills your data values into `{{ .var }}` references),
**encrypted**, or **encrypted template**. Change your mind later with
`casa configs storage` — it converts in place.

Your repo's setup questionnaire lives in **`.casa.toml.tmpl`** at the repo root
(chezmoi's `.chezmoi.toml.tmpl` works too — casa reads casa-named special files
first and mirrors them so plain chezmoi keeps working). `casa machine setup`
asks its questions in casa's UI, `casa machine answers` changes any answer later
without re-asking the rest, and `casa machine question` adds a new question —
answerable with text, yes/no, a number, or (multi-)choice — ready to use as
`{{ .key }}` in any template.

## Tools: one manifest, no Brewfile

Everything you install through casa is recorded in **`.casadata/packages.toml`**
— a single hand-editable file that chezmoi loads as template data. On every
`chezmoi apply`, run scripts render it straight into `brew bundle --file=-`
(install + cleanup) and run any self-installing tools; no Brewfile ever exists
on disk, and the repo keeps working without casa.

First `casa tools add` on a repo without a manifest offers to set everything up:
it creates the manifest + run scripts, then either **imports your existing
Brewfile** (and retires it) or **scans this machine** (`brew`, `cask`, taps,
`go`, `uv`, `npm`, `cargo`) so migration is one keypress. `casa tools import`
re-scans any time.

Tools that ship their own installer (`curl … | sh`) are first-class: pick *sh*
when adding, give it the one-liner and binary name, and every machine installs
it on apply — with an optional self-update command for `casa tools update`.

## Config (optional)

casa works on any chezmoi repo. To pin specifics, add a committed `.casa.toml`
at your repo root:

```toml
[pkg]
manifest = ".casadata/packages.toml"  # where `casa tools add` records (this is the default)

[setup]
repo = "your-username"                # default for `casa machine setup`
```

Setup questions (work/personal/…) come from your repo's `.casa.toml.tmpl`
prompts — casa doesn't define them, so it adapts to anyone's setup.

## casa names, chezmoi machinery

Special files in a casa repo carry casa names — `.casa.toml.tmpl`,
`.casaignore`, `.casadata/` — and casa maintains gitignored symlinks to the
names chezmoi hardcodes (`.chezmoi.toml.tmpl`, `.chezmoiignore`,
`.chezmoidata`), recreating them automatically before any chezmoi call. Repos
that use chezmoi names directly work unchanged. To use a casa repo with bare
chezmoi (no casa installed), create the three symlinks by hand once per clone.

## Versioning

Semver: `vMAJOR.MINOR.PATCH`. See [CHANGELOG.md](CHANGELOG.md).

## License

MIT
