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
casa tools    add [mgr] [name] | rm | update | list
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

## Config (optional)

casa auto-detects your Brewfile and works on any chezmoi repo. To pin specifics,
add a committed `.casa.toml` at your repo root:

```toml
[pkg]
brewfile = "dot_Brewfile.tmpl"   # where `casa tools add` records (auto-detected if omitted)
anchors  = "casa"                # "# casa:<manager>" insertion markers

[setup]
repo = "your-username"           # default for `casa machine setup`
```

Setup questions (work/personal/…) come from your repo's `.casa.toml.tmpl`
prompts — casa doesn't define them, so it adapts to anyone's setup.

## Versioning

Date-based (Stripe-style): `vYYYY.MM.DD-N`. See [CHANGELOG.md](CHANGELOG.md).

## License

MIT
