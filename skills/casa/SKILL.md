---
name: casa
description: Drive casa — the interactive chezmoi front-end that manages dotfiles, packages, and secrets from one repo. Use when the user mentions casa, dotfiles, tracked configs, package manifests (packages.toml), encrypted configs/age keys, or machine setup/sync. Prefer casa commands over raw chezmoi/git in a casa-managed repo; never create a Brewfile or commit chezmoi-named files. Skip for repos not managed by casa/chezmoi.
license: MIT
---

# casa

casa (github.com/carrots-sh/casa) wraps [chezmoi](https://chezmoi.io): the
user's dotfiles repo is the single source of truth for files, packages, and
secrets across machines. Every menu action is also a typed command — agents
should use the typed commands and let casa handle git, chezmoi, and rendering.

Check it exists and where the repo lives:

```bash
casa version          # installed?
casa machine info     # repo path (default ~/.local/share/casa), managed count
casa status           # to push / to pull / drift / outdated — start here
```

## The model

- **Repo = data.** Managed files (chezmoi `dot_*` names), the package
  manifest `.casadata/packages.toml`, encrypted `*.age` files, and the
  questionnaire `.casa.toml.tmpl`. All casa-named; gitignored symlinks map
  them to chezmoi's names — never commit `.chezmoi*` paths or `run_*`
  scripts (casa generates those from the installed binary).
- **Apply = converge.** `chezmoi apply` (via casa) renders files AND
  installs/uninstalls packages to match the manifest (brew bundle + cleanup,
  distro packages, self-installing tools). Removing a manifest entry
  uninstalls it on the next apply — treat manifest edits as real actions.
- **push / pull.** `casa push [msg]` commits + pushes repo changes (auto
  message if omitted). `casa pull` is bidirectional sync: offers to push
  local changes, surfaces drift for keep/restore, upgrades packages, then
  pulls + applies.

## Editing managed files (agent-safe path)

1. Find the source file in the repo: `casa files list` shows targets;
   sources live at `<repo>/dot_<path>` (templates end `.tmpl`).
2. Edit the SOURCE file, then apply just that target:
   `chezmoi --source <repo> apply ~/<target>`
3. `casa push "casa: <what changed>"`

`casa edit [name]` opens $EDITOR interactively (handles encrypted files
transparently) — fine for users, avoid in automation.

## Packages (the manifest)

`.casadata/packages.toml` sections: `taps`, `taps_trusted`, `brew`,
`brew_darwin`, `cask`, `go`, `uv`, `npm`, `bun`, `cargo`, `system` (Linux
distro pkgs), `extra`/`extra_darwin` (raw Brewfile lines, hand-managed),
`[[packages.sh]]` (self-installing tools: bin, install, optional update/os/
creates). Comments survive casa's edits.

```bash
casa tools add brew ripgrep        # install + record + push, no prompts
casa tools add go golang.org/x/tools/gopls
casa tools list                    # everything recorded
casa tools add                     # interactive search — users only
```

Tap-scoped formulae use full paths (`showwin/speedtest/speedtest`). For
removals or bulk edits, edit the manifest directly, then apply (next
`casa pull`) — cleanup uninstalls what you removed. NEVER create a
Brewfile; the manifest is the only package record.

## Secrets and keys

age-encrypted; keys are files in `~/.config/casa/keys/<name>.txt` — never
in any repo, never in the manifest. Repo may carry passphrase-sealed
backups (`.casa/keys/*.key.age`); restore prompts on fresh machines.
`casa secrets add/edit/remove/keys/list` are interactive (key pickers,
editors) — leave them to the user; never generate or handle passphrases.

## Machine lifecycle

```bash
casa machine setup <github-user|user/repo|url>   # provision a fresh machine
casa pull                                        # converge this machine
casa files drift                                 # review out-of-band changes
casa machine doctor                              # dependency health table
casa upgrade                                     # update casa itself
```

Fresh-machine one-liner (installs brew + casa, then sets up):
`curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <github-user>`

## Guardrails

- Don't run raw `git` in the repo — `casa push` / `casa undo` keep status
  hints and messages consistent (`git` reads are fine).
- Don't hand-write `run_*` scripts; behavior belongs in the manifest or in
  casa itself.
- Interactive commands (menus, pickers, editors) hang without a tty — in
  automation use only the fully-argumented forms shown above.
