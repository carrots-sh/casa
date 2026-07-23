---
name: casa
description: Overview and router for casa, the machine manager that runs a user's Macs and Linux boxes — files, tools, and secrets — from one git repo with two verbs, push and pull. Covers the mental model (repo = source of truth, apply converges files AND packages), state checks (casa status / machine info), the agent-safe edit flow, and when to load casa-files, casa-tools, casa-secrets, or casa-machine. Use when the user mentions casa, dotfiles, packages.toml, encrypted configs, age keys, or machine setup/sync — load this first, then route. Skip for repos casa does not manage.
license: MIT
---

# casa — make any machine feel like home

casa manages your machines — files, tools, and secrets — from one git repo,
with two verbs: **push** and **pull**. One repo holds the dotfiles, one
manifest (`.casadata/packages.toml`) declares every package — brew, distro,
go, uv, npm, bun, cargo — and secrets travel age-encrypted, with keys that
never touch a repo. It converges instead of accumulates: `casa pull` pushes
your changes first, reviews drift, upgrades packages, and actually uninstalls
whatever you removed from the manifest. macOS + Linux.

This is the overview + router skill. It gives you the mental model, the state
checks, and the agent-safe edit flow, then points you at the sibling skills
(`casa-files`, `casa-tools`, `casa-secrets`, `casa-machine`) for deep work.

## The mental model

- **One repo, three things.** The repo (default `~/.local/share/casa`,
  override with `$CASA_SOURCE`) holds:
  - **files** — dotfiles as chezmoi-style source files (`dot_zshrc`,
    `dot_gitconfig.tmpl`, `encrypted_private_dot_netrc.age`);
  - **tools** — a single package manifest `.casadata/packages.toml` covering
    `brew`/`brew_darwin`/`cask`/`taps`, `go`, `uv`, `npm`, `bun`, `cargo`,
    `system` (Linux distro packages), and `[[packages.sh]]` self-installing
    tools;
  - **secrets** — age-encrypted files in the repo; age identities are plain
    files in `~/.config/casa/keys/` and are NEVER in any repo. Opt-in
    passphrase-sealed key backups may live at `.casa/keys/*.key.age`.
- **Repo = data only.** Run scripts (package install, sh-tools, key restore)
  are GENERATED from the installed casa binary, gitignored, never committed.
- **Apply = converge.** Applying renders files AND converges packages to the
  manifest: it installs what is listed and uninstalls what you removed
  (brew bundle cleanup). A deleted manifest line is a real uninstall on the
  next apply — treat manifest edits as actions, not notes. No Brewfile ever
  exists; the manifest is the only package record.
- **Two verbs.**
  - `casa push [msg]` — your changes → repo (commit + push; auto-message if
    `msg` omitted).
  - `casa pull` — repo → machine, but safe in both directions: it pushes your
    local changes first, reviews drift (keep/restore per file), upgrades
    outdated packages, then applies everything.
- **Fresh machine is one curl away:**

  ```bash
  curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <github-user>
  ```

  This installs Homebrew if needed, installs casa, then runs
  `casa machine setup <github-user>` (expands to `<user>/dotfiles`; a
  `user/repo` or full URL also works). Setup asks the questions defined in
  the repo's `.casa.toml.tmpl` questionnaire and applies everything.

## Check state first

Before doing anything in a casa-managed environment, establish the facts:

```bash
casa version          # is casa installed? prints version + commit
casa status           # what's changed, behind, or outdated — start here
casa machine info     # repo path, machine + repo basics
```

- `casa status` is the one-look summary: local changes to push, remote
  commits to pull, drifted files, outdated packages.
- To locate the repo without casa: `$CASA_SOURCE` if set, else
  `~/.local/share/casa` if it is a git repo, else chezmoi's own source
  (`chezmoi source-path`). `casa machine info` prints the resolved path.
- Health problems (missing managers, broken keys): `casa machine doctor`
  prints a dependency table.

If none of these resolve to a casa repo, this skill does not apply — do not
force casa onto an unmanaged machine.

## Agent-safe editing flow

`casa edit` opens `$EDITOR` interactively — fine for humans, wrong for
agents. The agent path edits the source file directly:

1. Find the source. `casa files list` shows managed targets; the source for
   `~/.zshrc` is `<repo>/dot_zshrc` (templates add `.tmpl`, e.g.
   `<repo>/dot_gitconfig.tmpl`).
2. Edit the SOURCE file with your file tools.
3. Apply just that target (chezmoi is casa's rendering engine under the
   hood):

   ```bash
   chezmoi --source <repo> apply ~/.zshrc
   ```

4. Push:

   ```bash
   casa push "casa: describe what changed"
   ```

The same pattern covers manifest edits: edit `.casadata/packages.toml`
directly (casa edits it line-by-line, so comments and ordering survive),
then let the next `casa pull` converge — or use `casa tools add <manager>
<name>` for install+record in one step.

## Non-interactive operation

casa commands take at most two positional arguments and no flags. Any
bracketed argument you omit becomes an interactive prompt — which **hangs
without a tty**. In automation:

- Use only fully-argumented forms: `casa push "msg"`, `casa tools add brew
  ripgrep`, `casa files add ~/.config/foo/config`.
- `CASA_YES=1` auto-answers yes to confirmation prompts (prints
  `<question> → yes (CASA_YES)`). It does NOT satisfy selection or text
  prompts — those still need a tty, so commands that open pickers
  (`casa tools remove`, `casa secrets keys`, bare `casa`) remain
  human-only.
- Expect list/status commands (`casa status`, `casa files list`,
  `casa tools list`, `casa secrets list`, `casa machine info`) to be safe
  read-only calls.

## CLI surface

```text
usage: casa [command]           (no command opens the interactive menu)
shortcuts: casa edit [name] · casa push [msg] · casa pull · casa status
           casa cd · casa upgrade
files:   edit [name] · add [path] · drift · storage [name] · remove [path] · list
tools:   add [manager] [name] · add sh · add cmd ["command"] · remove · update · list · import · trust
secrets: add [path] · edit [name] · remove · keys · list
machine: setup [repo] · pull · push [message] · status · answers [name] · question · undo · doctor · info
```

Legacy aliases accepted everywhere: `save`=`push`, `sync`=`pull`,
`configs`=`files`, `track`=`add`, `untrack`=`remove`, `rm`=`remove`.
Do not invent commands beyond this surface. The interactive menu (bare
`casa`) is one flat filterable list of the same actions — every menu item is
addressable as a typed command, so agents never need the menu.

## Route to the sibling skills

| Task | Skill |
| --- | --- |
| Track/untrack dotfiles, templates, storage modes, drift review, `.casaignore` | `casa-files` |
| The package manifest: managers, `[[packages.sh]]` tools, taps/trust, import, converge/uninstall semantics | `casa-tools` |
| Encrypted files, age keys in `~/.config/casa/keys`, sealed backups, Doppler | `casa-secrets` |
| Fresh-machine setup, questionnaire (`.casa.toml.tmpl`), answers, undo, doctor, repo layout | `casa-machine` |

Stay in this skill for: identity questions ("what is casa"), quick state
checks, single-file edits via the agent-safe flow, and a plain
`casa push` / `casa pull`.

## Under the hood: chezmoi, and the exit door

chezmoi renders and applies the files — casa pins every call with
`chezmoi --source <repo>`. The repo stays a valid chezmoi repo: casa-named
files (`.casa.toml.tmpl`, `.casaignore`, `.casadata/`) are mirrored to
chezmoi's names via gitignored symlinks (`.chezmoi.toml.tmpl`,
`.chezmoiignore`, `.chezmoidata`) that casa self-heals. You can leave casa at
any time and keep everything. This is where chezmoi is relevant — never
present casa as a chezmoi wrapper or front-end; it is a machine manager that
uses chezmoi as its rendering engine.

## Guardrails

- **Never create a Brewfile.** The manifest `.casadata/packages.toml` is the
  only package record; casa pipes it to `brew bundle --file=-` at apply.
- **Never commit chezmoi-named files or `run_*` scripts.** The
  `.chezmoi*` symlinks and run scripts are generated and gitignored; repos
  carry only casa-named data. Don't hand-write run scripts — behavior
  belongs in the manifest or in casa.
- **Prefer casa over raw git for writes.** `casa push` / `casa machine undo`
  keep messages and status hints consistent. Read-only `git log`/`git diff`
  in the repo is fine.
- **Interactive pickers hang without a tty.** In automation, only
  fully-argumented forms plus `CASA_YES=1` for confirmations.
- **Secrets and passphrases stay human.** Never generate, type, or store key
  passphrases; never move key files out of `~/.config/casa/keys`; leave
  `casa secrets` interactive flows (key pickers, editors) to the user.
- **Manifest deletions uninstall.** Confirm with the user before removing
  manifest entries; the next apply removes the package from the machine.
