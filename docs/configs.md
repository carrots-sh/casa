# Configs

Configs are the files casa manages for you: shell rc files, git config, editor settings, anything under `$HOME`. casa is a front-end for chezmoi, so every operation here maps to a native chezmoi command — `add`, `edit`, `chattr`, `forget` — with casa choosing sensible defaults and handling encryption transparently.

Paths are shown with `~/` everywhere.

```console
$ casa configs list
~/.zshrc
~/.gitconfig  (template)
~/.config/secret.env  (encrypted)
```

## Tracking a file

```console
$ casa configs track ~/.zshrc
```

With a path, casa starts managing that file directly. Without one, it opens an adopt picker:

- **Suggestions** — well-known dotfiles present in `$HOME` that aren't managed yet (`.zshrc`, `.zprofile`, `.bashrc`, `.bash_profile`, `.profile`, `.gitconfig`, `.gitignore`, `.vimrc`, `.tmux.conf`, `.inputrc`, `.config/starship.toml`, `.config/nvim/init.lua`, `.config/ghostty/config`, `.aliases`, `.functions`, `.curlrc`, `.editorconfig`).
- **`another file · type a path`** — a path prompt with as-you-type completion for anything not on the list.

### Choosing a storage mode

For each file, casa asks how it should be stored:

| Mode | Meaning | chezmoi equivalent |
| --- | --- | --- |
| `plain` | Same on every machine | `chezmoi add` |
| `template` | Differs per machine (auto-fills your data) | `chezmoi add --autotemplate` |
| `encrypted` | Secret, sealed in the repo | `chezmoi add --encrypt` |
| `encrypted template` | Secret and per-machine | `chezmoi add --encrypt --template` |

The default follows two heuristics, checked in order:

1. **Sensitive-looking name → `encrypted`.** The filename contains one of `.env`, `.pem`, `.key`, `credential`, `secret`, `id_rsa`, `id_ed25519`, or `token`.
2. **Contains machine data → `template`.** The file's content mentions one of this machine's template data values (any string data value of 4+ characters, plus `.chezmoi.hostname` and `.chezmoi.username`) — a good candidate for per-machine substitution.

Otherwise the default is `plain`. The heuristics only move the cursor; you always confirm the choice.

Encrypted files are sealed with your age key. See [secrets](secrets.md).

## Templates

Choosing `template` runs `chezmoi add --autotemplate`: chezmoi scans the file and replaces occurrences of your data values with template references. If your `.gitconfig` contains your email, the stored source becomes:

```gitconfig
[user]
    email = {{ .email }}
```

On every machine, `apply` renders the template with that machine's data, so one source file produces the right output everywhere. You can use `{{ .var }}` for any data value in any templated file — chezmoi does all the rendering, casa never reimplements template semantics.

### Setup questions as template data

Data values come from your repo's setup questionnaire (`.casa.toml.tmpl`). Add a question with:

```console
$ casa machine question
```

You pick a data key, the question text, and an answer kind (text, yes/no, one of a list, several of a list, or number). casa writes the corresponding `prompt*Once` call into the questionnaire, asks it on this machine immediately, and the key becomes available as `{{ .key }}` in every template. New machines are asked during setup; change an answer later with `casa machine answers`. See [machine](machine.md).

## Changing storage

```console
$ casa configs storage gitconfig
```

Storage is not fixed at track time. `storage` shows the file's current attributes preselected in a multi-select — `template`, `encrypted`, `private`, `executable` — and applies whatever you toggle via `chezmoi chattr`. Turning everything off stores the file plain.

Attributes are read from the chezmoi source filename (the `encrypted_`/`private_`/`executable_` prefixes and `.tmpl` suffix), so what you see always matches what's in the repo.

## Editing

```console
$ casa configs edit zshrc
$ casa edit zshrc          # top-level shortcut
```

`edit` opens the managed file's source in your editor and applies the result (`chezmoi edit --apply`). Encrypted files are handled transparently: decrypted for editing, re-encrypted on save. After an edit, casa offers to commit and push.

## Untracking

```console
$ casa configs untrack ~/.vimrc
```

`untrack` stops managing a file (`chezmoi forget --force`) but keeps it on disk untouched. Pass a path, a name to match against managed files, or nothing to pick from a list.

## Listing

```console
$ casa configs list
```

Lists every managed file with a storage badge — `(template)`, `(encrypted)`, or `(template, encrypted)` — derived from its source filename. Plain files get no badge.

## Name matching

Every command that takes an optional `[name]` (`edit`, `storage`, and `untrack` when the argument isn't an existing path) resolves it the same way:

- An exact match on a managed path wins.
- A single case-insensitive substring hit opens directly (`casa edit zsh` → `~/.zshrc`).
- Several hits open a pre-filtered picker.
- No argument opens the full picker, badges included.
