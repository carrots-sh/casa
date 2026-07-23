# Repository layout

One git repo holds everything casa manages: your files, one package manifest,
and your secrets. This page maps out what gets committed, what casa generates
locally, and what never leaves the machine.

Under the hood, chezmoi renders and applies your files, so the repo is also a
valid chezmoi source directory — everything chezmoi understands works
unchanged. casa adds a naming convention for the special files, the package
manifest, and optional sealed key backups.

By default the repo lives at `~/.local/share/casa` (override with
`$CASA_SOURCE`; existing `~/.local/share/chezmoi` setups are picked up
automatically). casa pins every chezmoi call to this directory with
`chezmoi --source`.

## Anatomy

```
~/.local/share/casa
├── dot_zshrc                                  # committed: your dotfiles
├── dot_gitconfig.tmpl                         # committed: templated dotfiles
├── encrypted_private_dot_netrc.age            # committed: encrypted dotfiles
├── .casa.toml.tmpl                            # committed: setup questionnaire
├── .casaignore                                # committed: ignore rules
├── .casadata/
│   └── packages.toml                          # committed: package manifest
├── .casa/
│   └── keys/main.key.age                      # committed: sealed key backups (optional)
├── .casa.toml                                 # committed: casa config (optional)
├── .gitignore                                 # committed: casa appends entries below
│
├── .chezmoi.toml.tmpl -> .casa.toml.tmpl      # generated: gitignored symlink
├── .chezmoiignore -> .casaignore              # generated: gitignored symlink
├── .chezmoidata -> .casadata                  # generated: gitignored symlink
├── run_onchange_after_10-packages.sh.tmpl     # generated: gitignored run script
├── run_onchange_after_20-sh-tools.sh.tmpl     # generated: gitignored run script
└── run_once_before_00-casa-keys.sh.tmpl       # generated: gitignored run script
```

Outside the repo, per machine:

```
~/.config/casa/keys/                           # age identities (never committed)
│   ├── main.txt
│   └── .default                               # marker naming the default key
~/.config/chezmoi/chezmoi.toml                 # rendered by chezmoi init
```

## Committed files

These are the repo's content — everything a teammate or a future machine needs.

| Path | Purpose |
| --- | --- |
| `dot_*`, `private_*`, `encrypted_*`, … | Your dotfiles, in chezmoi's normal source-state naming. |
| `.casa.toml.tmpl` | The setup questionnaire. `casa machine setup` parses its `prompt*` calls and answers them via `chezmoi init`. YAML and JSON variants (`.casa.yaml.tmpl`, `.casa.json.tmpl`) work too. |
| `.casaignore` | Ignore rules, same syntax as `.chezmoiignore`. Commonly used to skip files per machine based on questionnaire answers. |
| `.casadata/packages.toml` | The single package manifest. chezmoi loads it as template data; the generated run scripts render it into `brew bundle` and the sh-tool installer at apply. See [Tools](tools.md). |
| `.casa/keys/<name>.key.age` | Optional passphrase-sealed backups of age keys, created by `casa secrets keys` → backup to repo. Safe to commit: each backup is encrypted with a passphrase, not with the key itself. See [Secrets](secrets.md). |
| `.casa.toml` | Optional casa config: `[pkg] manifest` (source-relative manifest path, default `.casadata/packages.toml`) and `[setup] repo` (default repo for `casa machine setup`). casa works without it. |

The manifest path has one automatic exception: a repo with its own real
`.chezmoidata` directory (not casa's symlink) and no `.casadata/packages.toml`
keeps the manifest at `.chezmoidata/packages.toml`, since chezmoi only loads
template data from that name.

## Generated files (gitignored)

casa regenerates these before every chezmoi call. They are appended to the
repo's `.gitignore` automatically, so the repo never carries them.

### Symlink mirrors

casa repos commit casa-named special files; chezmoi hardcodes its own names.
Before each chezmoi invocation casa creates any missing symlink from the
chezmoi name to the casa file:

| Committed (casa name) | Symlinked (chezmoi name) |
| --- | --- |
| `.casa.toml.tmpl` | `.chezmoi.toml.tmpl` |
| `.casa.yaml.tmpl` | `.chezmoi.yaml.tmpl` |
| `.casa.json.tmpl` | `.chezmoi.json.tmpl` |
| `.casaignore` | `.chezmoiignore` |
| `.casaremove` | `.chezmoiremove` |
| `.casaversion` | `.chezmoiversion` |
| `.casaexternal.toml` | `.chezmoiexternal.toml` |
| `.casadata` (directory) | `.chezmoidata` |
| `.casadata.toml` | `.chezmoidata.toml` |
| `.casadata.yaml` | `.chezmoidata.yaml` |
| `.casadata.json` | `.chezmoidata.json` |

A symlink is only created when the casa file exists and nothing already sits at
the chezmoi name — a repo that uses chezmoi names directly is left untouched.
Every symlink casa creates is added to `.gitignore` under a
`# chezmoi-named mirrors` comment.

### Run scripts

casa's default behavior never lives in the repo. Three run scripts are written
into the source directory from templates embedded in the casa binary, and
refreshed whenever their on-disk content differs from the installed version —
so script behavior always matches the casa you are running, not the casa that
last touched the repo.

| Script | Generated when | Does |
| --- | --- | --- |
| `run_onchange_after_10-packages.sh.tmpl` | a package manifest exists | Renders the manifest into `brew bundle --file=-` (install and cleanup) on apply. |
| `run_onchange_after_20-sh-tools.sh.tmpl` | a package manifest exists | Installs `[[packages.sh]]` tools behind `command -v` guards. |
| `run_once_before_00-casa-keys.sh.tmpl` | `.casa/keys/` backups exist | Restores backed-up age keys into `~/.config/casa/keys` on a new machine, before anything needs decrypting. |

These are also gitignored automatically. Deleting one is harmless; casa
recreates it on the next invocation.

## Per-machine state

Nothing key-related is ever committed in identifiable form. Each machine keeps:

- `~/.config/casa/keys/<name>.txt` — age identities. A key *is* its file; the
  filename is the key's name and recipients are derived with `age-keygen -y`.
- `~/.config/casa/keys/.default` — a one-line marker naming the default key.
  When the marker is missing or stale, the first key by name is used.
- `~/.config/chezmoi/chezmoi.toml` — chezmoi's rendered config, produced by
  `chezmoi init` from `.casa.toml.tmpl` with this machine's answers. Re-render
  answers any time with `casa machine answers`.

## Compatibility with bare chezmoi

The repo stays a valid chezmoi repo — repos carry only data, never casa-specific
machinery. You can leave casa at any time and keep everything.

Works with plain `chezmoi` immediately, on a clone where casa has run at least
once:

- All dotfiles, templates, and encrypted files — standard chezmoi source state.
- The symlink mirrors and run scripts — they exist on disk (gitignored), so
  `chezmoi apply`, `chezmoi diff`, and `chezmoi update` behave normally,
  including package installs from the manifest.

Needs casa to have run once (or three symlinks made by hand) on a **fresh
clone**, because the gitignored mirrors are not in the repo:

```bash
cd ~/.local/share/casa
ln -s .casa.toml.tmpl .chezmoi.toml.tmpl
ln -s .casaignore .chezmoiignore
ln -s .casadata .chezmoidata
```

Without the mirrors, bare chezmoi on a fresh clone sees no questionnaire, no
ignore rules, and no template data. Without casa having generated the run
scripts, package installs and key restore will not run either — any `casa`
invocation against the clone creates all of it.
