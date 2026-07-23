# Reference

The complete casa surface: every command, every environment variable, the
keyboard controls, and the `.casa.toml` config file.

casa's commands take at most two positional arguments and no flags. Every
argument shown in brackets is optional — omit it and casa prompts
interactively instead.

## Commands

Running `casa` with no arguments opens the interactive menu. Every subcommand
below is the same action, addressable for scripting.

```console
$ casa help
casa — easy chezmoi: manage your configs and tools from one friendly menu

usage: casa [command]           (no command opens the interactive menu)
```

### Top-level shortcuts

The most frequent actions are available without a group prefix:

| Command | Description |
| --- | --- |
| `casa` | Open the interactive menu |
| `casa edit [name]` | Pick and edit a config (same as `casa configs edit`) |
| `casa push [msg]` | Commit and push your changes (same as `casa machine save`) |
| `casa pull` | Upgrade packages, then pull + apply dotfiles (same as `casa machine sync`) |
| `casa status` | Show what's changed, behind, or outdated (same as `casa machine status`) |
| `casa upgrade` | Update casa itself to the latest release |
| `casa help` | Print usage (also `-h`, `--help`) |
| `casa version` | Print version and commit (also `-v`, `--version`) |

### `casa configs`

Manage tracked configuration files. See [Configs](configs.md).

| Command | Description |
| --- | --- |
| `casa configs edit [name]` | Pick and edit a config; encrypted ones are handled transparently |
| `casa configs track [path]` | Start managing an existing file (plain, template, or encrypted) |
| `casa configs storage [name]` | Change how a file is stored (template/encrypted/…) |
| `casa configs untrack [path]` | Stop managing a file (keeps it on disk) |
| `casa configs list` | List managed files |

### `casa tools`

Manage installed tools through the package manifest. See [Tools](tools.md).

| Command | Description |
| --- | --- |
| `casa tools add [manager] [name]` | Install a tool and record it in your manifest |
| `casa tools add sh` | Add a tool that ships its own installer (`curl \| sh`) |
| `casa tools add cmd ["command"]` | Paste any install command — casa detects the manager |
| `casa tools rm` | Uninstall tool(s) — pick across all managers |
| `casa tools update` | Upgrade outdated tools — one, many, or all |
| `casa tools list` | List recorded tools |
| `casa tools import` | Seed the manifest from what's installed here |
| `casa tools trust` | Pick which taps update without prompting |

### `casa secrets`

Manage encrypted files and age keys. See [Secrets](secrets.md).

| Command | Description |
| --- | --- |
| `casa secrets add [path]` | Encrypt and start managing a file |
| `casa secrets edit [name]` | Pick a secret, decrypt, edit, re-encrypt |
| `casa secrets keys` | Encryption keys — create, default, delete, doppler |
| `casa secrets list` | List encrypted files |

### `casa machine`

Machine-level operations. See [Machine](machine.md).

| Command | Description |
| --- | --- |
| `casa machine setup [repo]` | Provision this machine from your dotfiles repo |
| `casa machine sync` | Upgrade packages, then pull + apply dotfiles |
| `casa machine save [message]` | Commit + push your changes |
| `casa machine status` | Show what's changed, behind, or outdated |
| `casa machine answers [name]` | Change this machine's setup answers and re-apply |
| `casa machine question` | Add a setup question to your repo |
| `casa machine undo` | Revert the last saved change and re-apply |
| `casa machine doctor` | Health check |
| `casa machine info` | Machine + repo basics |

`casa machine setup` accepts a GitHub username (expands to `<user>/dotfiles`),
a `user/repo`, or a full URL. It prefers SSH and falls back to HTTPS. With no
argument, it uses `[setup].repo` from `.casa.toml` (see below), then prompts.

!!! note

    `casa machine context` is accepted as a legacy alias for
    `casa machine answers`.

An unknown command prints the usage text and exits with status 1.

## Environment variables

### `CASA_SOURCE`

Overrides the source directory (where your dotfiles repo lives). Without it,
casa resolves the source directory in order:

1. `$CASA_SOURCE`, if set.
2. `~/.local/share/casa`, if it is a git repo.
3. chezmoi's own configured source (`chezmoi source-path`), if that directory
   exists — this keeps existing chezmoi setups working unchanged.
4. `~/.local/share/casa` — where a fresh `casa machine setup` will clone.

`casa machine setup` also honors `CASA_SOURCE` as the clone target.

### `CASA_YES`

`CASA_YES=1` answers yes to every confirmation prompt without showing it,
printing `<question>  → yes (CASA_YES)` instead. Use it to run casa
non-interactively in scripts. Only confirmations are affected; selection and
text prompts still render.

### `CASA_PLAIN_PATH`

`CASA_PLAIN_PATH=1` disables PATH self-healing (see below). Useful in
sandboxed environments that deliberately mask package managers via `PATH`.

### `EDITOR`

`casa secrets edit` opens the decrypted content in `$EDITOR`, falling back to
`vi` when unset. `casa configs edit` runs `chezmoi edit --apply`, which
follows chezmoi's own editor resolution (also `$EDITOR`-based).

### `GOBIN` and `BUN_INSTALL`

At startup casa prepends well-known package-manager bin directories that
exist on the machine but are missing from `PATH`, so every manager — and the
tools they installed — resolves even under a minimal environment (fresh
machine, cron, GUI-spawned shell). The built-in list, most specific first:

```text
~/go/bin                                      # go install
~/.local/bin                                  # uv tools, sh installers, chezmoi bootstrap
~/.cargo/bin                                  # cargo install
~/.bun/bin                                    # bun add -g
/opt/homebrew/bin                             # brew, macOS arm64
/opt/homebrew/sbin
/opt/homebrew/opt/rustup/bin                  # rustup via brew is keg-only
/usr/local/bin                                # brew, macOS intel
/usr/local/opt/rustup/bin
/home/linuxbrew/.linuxbrew/bin                # brew, linux
/home/linuxbrew/.linuxbrew/sbin
/home/linuxbrew/.linuxbrew/opt/rustup/bin
```

If `GOBIN` is set, it is prepended ahead of the list; if `BUN_INSTALL` is
set, `$BUN_INSTALL/bin` is prepended ahead of the list. Directories that do
not exist are skipped. The change applies to the casa process and everything
it spawns; your shell's `PATH` is untouched.

### `CASA_AGE_KEY_<NAME>`

The Doppler secret name casa uses when pushing or pulling an age key named
`<name>` (uppercased, dashes become underscores). This is a Doppler-side
naming convention, not a variable casa reads from your shell. See
[Secrets](secrets.md).

## Keyboard controls

Controls are consistent across every prompt:

| Key | Action |
| --- | --- |
| `tab` | Select: toggle the highlighted row (multi-select), accept a completion (inputs), toggle yes/no (confirms) |
| `enter` | Submit: pick the highlighted row, submit the selection or text |
| `esc` | Back: cancel the current prompt and return to the previous screen |
| `←` | Back, on list prompts only — text inputs keep `←` for the cursor |
| `ctrl+c` | Quit casa entirely, from anywhere |

Additional per-prompt keys:

- **Single-choice lists** open with the filter active — just type to narrow
  the options.
- **Multi-select lists** are filterable; `space` and `x` also toggle rows.
- **Path inputs** complete as you type from the filesystem; `tab` or `→`
  accepts the suggestion (directories complete with a trailing `/`, and `~`
  stays `~`).
- **Confirms** toggle with `tab`, `h`, `l`, `←`, or `→`.

## The `.casa.toml` file

An optional, committed file at the root of your source directory. casa works
against any chezmoi repo without it — every key has a default.

```toml
[pkg]
# Source-relative path of the package manifest.
# Default: ".casadata/packages.toml"
manifest = ".casadata/packages.toml"

[setup]
# Default repo for `casa machine setup` when no argument is given.
repo = "you/dotfiles"
```

### `[pkg].manifest`

Source-relative path of the package manifest read by `casa tools`. Defaults
to `.casadata/packages.toml`. One automatic fallback: a repo that has its own
real `.chezmoidata` directory (not casa's mirror symlink) and no manifest at
the default path keeps the manifest at `.chezmoidata/packages.toml`, since
chezmoi only loads data from that name. See [Tools](tools.md) for the
manifest format.

### `[setup].repo`

The repo `casa machine setup` provisions from when invoked without an
argument. Same forms as the positional argument: a GitHub username, a
`user/repo`, or a full URL.
