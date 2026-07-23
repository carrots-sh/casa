---
name: casa-files
description: Manage files with casa — the machine manager that keeps dotfiles, tools, and secrets in one git repo with two verbs, push and pull. Covers casa files add/edit/list/remove/storage/drift: storage kinds (plain, template, encrypted) and add's heuristics, source-name mapping (dot_zshrc, .tmpl, encrypted_*.age), the agent-safe edit flow (edit source, targeted apply, casa push), drift review (keep/restore/skip), questionnaire template data, and .casaignore gating. Use when tracking, editing, or reconciling dotfiles in a casa repo. Skip for tools/secrets/machine-setup work or non-casa repos.
license: MIT
---

# casa files — dotfiles from one repo

casa manages your machines — files, tools, and secrets — from one git repo, with two verbs: push and pull. This skill covers the **files** part: shell rc files, git config, editor settings, anything under `$HOME`, stored in the repo as plain files, per-machine templates, or age-encrypted secrets.

Under the hood, chezmoi renders and applies the files — every casa files operation maps to a native chezmoi command, and the repo stays a valid chezmoi repo, so the user can leave casa at any time and keep everything. When automating, prefer `casa` commands; drop to `chezmoi --source "$repo" ...` only for the specific non-interactive forms shown below.

## Orient first

```bash
casa status            # to push / to pull / drift / outdated — the one-line health check
casa machine info      # repo path + managed file count
casa files list        # every managed file with storage badges (plain output, pipeable)
```

The repo lives at `~/.local/share/casa` by default (override: `$CASA_SOURCE`; an existing `~/.local/share/chezmoi` is picked up automatically). Resolve it programmatically:

```bash
repo="${CASA_SOURCE:-$HOME/.local/share/casa}"
[ -d "$repo/.git" ] || repo=$(chezmoi source-path)   # legacy chezmoi layouts only
```

`casa files list` output looks like:

```
~/.zshrc
~/.gitconfig  (template)
~/.config/secret.env  (encrypted)
~/.ssh/config  (template, encrypted)
```

No badge means plain. Badges are derived from the source filename, so they always match what's in the repo.

## Storage kinds

Each managed file has one of four storage modes:

| Mode | Meaning | chezmoi equivalent |
| --- | --- | --- |
| plain | same bytes on every machine | `chezmoi add` |
| template | differs per machine — rendered from machine data | `chezmoi add --autotemplate` |
| encrypted | secret, sealed in the repo with the user's age key | `chezmoi add --encrypt` |
| encrypted template | secret and per-machine | `chezmoi add --encrypt --template` |

### How `add` picks a default

`casa files add <path>` always asks the storage question, but the preselected answer follows two heuristics, checked in order:

1. **Sensitive-looking name → encrypted.** Filename contains one of `.env`, `.pem`, `.key`, `credential`, `secret`, `id_rsa`, `id_ed25519`, `token`.
2. **Contains machine data → template.** File content mentions any of this machine's template data values (string data values of 4+ chars, plus `.chezmoi.hostname` / `.chezmoi.username`).

Otherwise the default is plain. The heuristics only move the cursor; the user confirms.

## Source-name mapping

Managed files live in the repo under chezmoi source-state names. You need this to find the file to edit:

| Target ($HOME path) | Source (repo path) |
| --- | --- |
| `~/.zshrc` (plain) | `dot_zshrc` |
| `~/.gitconfig` (template) | `dot_gitconfig.tmpl` |
| `~/.netrc` (encrypted) | `encrypted_private_dot_netrc.age` |
| `~/.config/starship.toml` | `dot_config/starship.toml` |

Rules: leading `.` becomes `dot_`, templates get a `.tmpl` suffix, encrypted files get an `encrypted_` prefix and `.age` suffix, and `private_` / `executable_` prefixes carry file permissions. Never guess — resolve exactly:

```bash
chezmoi --source "$repo" source-path ~/.gitconfig
# → /Users/x/.local/share/casa/dot_gitconfig.tmpl
```

## Adding files

```bash
casa files add ~/.zshrc      # start managing this file (asks the storage question)
casa files add               # adopt picker: well-known unmanaged dotfiles + "type a path"
```

Both forms are **interactive** (storage select, then an offer to push). The no-arg picker suggests common dotfiles found in `$HOME` that aren't managed yet (`.zshrc`, `.gitconfig`, `.tmux.conf`, `.config/starship.toml`, `.config/nvim/init.lua`, ...).

**Agent non-interactive path** — use the chezmoi equivalent from the table, pinned to the repo, then push:

```bash
chezmoi --source "$repo" add ~/.zshrc                    # plain
chezmoi --source "$repo" add --autotemplate ~/.gitconfig # template
casa push "casa: track .zshrc"
```

Only use `--encrypt` if `casa secrets keys` / the user confirms an age key exists; encryption without a key fails.

## Editing

### Agent-safe flow (non-interactive)

`casa edit` opens `$EDITOR` — it hangs without a tty. In automation, edit the **source** file and apply just that target:

1. Resolve the source: `chezmoi --source "$repo" source-path ~/.zshrc`
2. Edit that file with your file tools. In a `.tmpl` source, remember it renders as a template — literal `{{` must be escaped, and machine values may appear as `{{ .email }}` rather than text.
3. Targeted apply, then verify and push:

```bash
chezmoi --source "$repo" apply ~/.zshrc     # render source → $HOME
chezmoi --source "$repo" status             # empty = clean
casa push "casa: edit .zshrc"
```

Do **not** edit the file in `$HOME` and stop there — that creates drift, not a repo change. If you did edit the target, record it with `chezmoi --source "$repo" add <target>` then push.

Encrypted files are the exception: the source is an opaque `.age` blob. Never hand-edit it, and never decrypt secrets into logs or transcripts. Leave those to the interactive flow (`casa edit <name>` or `casa secrets edit`), which decrypts, validates, and re-seals with the same key.

### Interactive flow (users)

```bash
casa files edit zshrc
casa edit zshrc              # top-level shortcut, same thing
```

Fuzzy-matches the name, opens the source in `$EDITOR`, applies on save (`chezmoi edit --apply`), handles encrypted files transparently, then offers to push.

## Name matching

`edit`, `storage`, and `remove` (when the argument isn't an existing path) resolve `[name]` the same way:

- exact match on a managed path wins;
- a single case-insensitive substring hit opens directly (`casa edit zsh` → `~/.zshrc`);
- several hits open a pre-filtered picker (interactive);
- no argument opens the full picker.

In automation, pass a string specific enough for a unique hit, or the full target path.

## Changing storage

```bash
casa files storage gitconfig     # interactive multi-select
```

Storage is not fixed at add time. The command shows the file's current attributes preselected — `template`, `encrypted`, `private`, `executable` — and applies the toggles via `chezmoi chattr`. Everything off = plain. Non-interactive equivalent:

```bash
chezmoi --source "$repo" chattr -- +template ~/.gitconfig
casa push "casa: change storage of .gitconfig"
```

## Removing

```bash
casa files remove ~/.vimrc
```

Stops managing the file (`chezmoi forget --force`) but **keeps it on disk untouched** — this is not a delete. Accepts a path, a name to match, or nothing (picker). Legacy aliases: `configs` = `files`, `track` = `add`, `untrack` and `rm` = `remove`.

## Drift

Drift = managed files whose on-disk state differs from what the repo would render — usually edits made outside casa. `casa status` shows the count; `casa files drift` reviews it.

```bash
casa files drift     # interactive: pick a file, see the diff, choose a side
```

For each drifted file casa shows the diff (`-` lines are the local change) and offers exactly three choices:

- **keep** — my local version wins: record it in the repo (`chezmoi add`), then casa offers to push;
- **restore** — the repo wins: overwrite the local change (`chezmoi apply --force`);
- **skip** — decide later; nothing changes.

Pending run scripts (chezmoi `R` status rows) are not file drift — they just run on the next `casa pull`; drift reports them but has nothing to keep or restore.

**Agent non-interactive equivalents** — inspect, then resolve per file. Never bulk-resolve drift without showing the user the diffs; keep and restore both destroy one side.

```bash
chezmoi --source "$repo" status                     # two-letter code + target per line
chezmoi --source "$repo" diff --no-pager ~/.zshrc   # show the divergence
chezmoi --source "$repo" add ~/.zshrc               # KEEP local → then casa push
chezmoi --source "$repo" apply --force ~/.zshrc     # RESTORE repo version
```

Note `casa pull` surfaces drift too: it pushes your changes first, has you review drift, upgrades packages, then applies.

## Templates and machine data

Template sources use chezmoi template syntax (Go text/template + sprig); casa never reimplements rendering. `--autotemplate` at add time replaces occurrences of your data values with references, so a `.gitconfig` containing your email is stored as:

```gitconfig
[user]
    email = {{ .email }}
```

Data values come from the repo's setup questionnaire, `.casa.toml.tmpl` — the questions casa asks in its own UI during `casa machine setup`, answered per machine. Built-ins like `{{ .chezmoi.os }}` and `{{ .chezmoi.hostname }}` are always available. Inspect what a template can use:

```bash
chezmoi --source "$repo" data --format json
```

- `casa machine question` — add a questionnaire key (text / yes-no / choice / multi / number); it's asked immediately and becomes `{{ .key }}` in every template.
- `casa machine answers [name]` — re-ask and re-render answers on this machine.
- `chezmoi --source "$repo" cat ~/.gitconfig` — preview a rendered target without applying (avoid on encrypted files).

## .casaignore — what renders where

`.casaignore` (committed at the repo root) gates which files exist per machine. Same syntax as chezmoi's ignore file, and it is itself a template, so entries can key off OS or questionnaire answers:

```
README.md

{{ if ne .chezmoi.os "darwin" }}
.config/karabiner/**
{{ end }}

{{ if not .work }}
.config/work/**
{{ end }}
```

Ignored paths are neither applied nor reported as drift on that machine. Edit `.casaignore` directly, then `casa push`; a full `casa pull` (or `chezmoi --source "$repo" apply`) picks it up.

## Guardrails

- **Interactive commands hang without a tty**: bare `casa`, no-arg `add`/`edit`/`storage`/`remove`, and `files drift`. In automation use only the argumented forms and chezmoi equivalents above.
- **Commit only casa names.** The repo carries `.casa.toml.tmpl`, `.casaignore`, `.casadata/`; the `.chezmoi*` names are gitignored symlinks casa regenerates before every call. Never commit `.chezmoi*` paths or `run_*` scripts — run scripts are generated from the installed casa binary.
- **No raw `git` writes in the repo** — `casa push [msg]` commits and pushes (auto-message if omitted); `casa machine undo` reverts. Read-only `git log`/`diff` is fine.
- **Every change ends in `casa push`** — an unpushed repo edit helps no other machine.
- **Never expose secret plaintext**: no `chezmoi cat`/`decrypt` of encrypted targets into logs.
- macOS and Linux only.
