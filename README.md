# casa

An easy, **interactive** front-end for [chezmoi](https://chezmoi.io). Manage your
dotfiles, secrets, and tools from one friendly menu — nothing to memorize.

casa never reimplements chezmoi; it shells out to it, so your dotfiles repo stays
the single source of truth and works with or without casa.

## Install

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
casa            # → "Machine" → "Set up this machine", answer the questions
casa            # from then on: pick what you want, follow the prompts
```

## Typed commands (optional — for scripts & muscle memory)

Everything in the menu is also a namespaced command:

```
casa tools    add [mgr] [name] | rm | update | list
casa configs  edit [name] | track [path] | untrack [path] | list
casa secrets  add [path] | edit | list
casa machine  setup [repo] | sync | save [msg] | status | context | doctor | info
```

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

Contexts (work/personal/…) come from your repo's `.chezmoi.toml.tmpl` prompts —
casa doesn't define them, so it adapts to anyone's setup.

## Versioning

Date-based (Stripe-style): `vYYYY.MM.DD-N`. See [CHANGELOG.md](CHANGELOG.md).

## License

MIT
