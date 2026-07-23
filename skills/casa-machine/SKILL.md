---
name: casa-machine
description: Operate casa's machine lifecycle — provision, converge, and maintain a Mac or Linux box from its git repo. Covers the fresh-machine curl bootstrap, casa machine setup (clone-first, setup questionnaire), full push/pull semantics, status fields, answers/question/undo/doctor/info (plus top-level casa cd), and step-by-step VPS provisioning with passphrase key restore. Use when setting up a new machine or server from a casa repo, syncing (push/pull/status), changing setup answers, authoring questions, or diagnosing casa health. Skip for day-to-day file/tool/secret edits (see the casa skill) and non-casa repos.
license: MIT
---

# casa machine — provision, converge, maintain

casa manages your machines — files, tools, and secrets — from one git repo,
with two verbs: push and pull. This skill covers the machine lifecycle: making
a fresh Mac, Linux box, or VPS yours, and keeping every machine converged with
the repo afterwards. macOS + Linux only.

```bash
casa machine setup [repo]     # provision this machine from your repo
casa machine pull             # repo → machine (pushes yours first)
casa machine push [message]   # your changes → repo (commit + push)
casa machine status           # to push / to pull / drift / outdated
casa machine answers [name]   # re-ask setup questions, re-apply
casa machine question         # author a new setup question
casa machine undo             # revert the last push and re-apply
casa machine doctor           # dependency health check
casa machine info             # machine name, repo path, managed count
```

`casa pull`, `casa push`, and `casa status` are top-level shortcuts for the
same commands. Legacy aliases still work: `save`=`push`, `sync`=`pull`,
`casa machine context`=`answers`. `casa cd` opens a subshell inside the repo
(exit to return). `casa upgrade` updates casa itself.

Always start by checking state:

```bash
casa version          # is casa installed?
casa machine info     # repo path (default ~/.local/share/casa), managed count
casa status           # what would push/pull actually do
```

## Fresh machine: one curl

On a machine with nothing installed:

```bash
curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <github-user>
```

What the installer does, in order:

1. **Linux prereqs** (only if `brew` is missing, only on Linux): installs
   Homebrew's prerequisites via the distro's manager — `apt-get install
   build-essential procps curl file git`, or the dnf/pacman equivalents —
   using `sudo` when not root.
2. **Homebrew**: official installer under `NONINTERACTIVE=1`, then
   `eval "$(brew shellenv)"` for this session.
3. **casa**: `brew install carrots-sh/tap/casa`.
4. **Setup**: `casa machine setup <github-user>` — interactive from here
   (questionnaire, key passphrases). Omit the argument to install casa only;
   it then prints `run 'casa' to get started`.

The setup step needs a real terminal (ssh session is fine). Do not pipe the
installer with an argument from a headless script that has no tty.

## Setup, in detail

```bash
casa machine setup carlos            # → github.com/carlos/dotfiles
casa machine setup carlos/machines   # → that GitHub repo
casa machine setup git@host:me/dots  # any git URL, used as-is
casa machine setup                   # uses [setup].repo from .casa.toml, else asks
```

GitHub forms prefer SSH and fall back to HTTPS; each candidate URL is probed
with a non-interactive `git ls-remote` (no password/host-key prompt can hang
it) before cloning. To make bare `casa machine setup` work, commit this to the
repo root:

```toml
# .casa.toml
[setup]
repo = "carlos/dotfiles"
```

### Clone first, then init

Setup pins the source directory to `~/.local/share/casa` (override with
`$CASA_SOURCE`) and clones with plain `git clone` — not `chezmoi init <repo>`.
Deliberate ordering: casa needs the repo on disk *before* the first chezmoi
call, so it can mirror the casa-named special files (`.casa.toml.tmpl`,
`.casaignore`, `.casadata/`) to the names chezmoi expects via gitignored
symlinks, and generate the run scripts (package install, sh-tools, key
restore) from the installed casa binary. Those scripts are gitignored and
never committed — repos carry only data. If the target directory already has
a `.git`, the existing checkout is reused.

Under the hood, chezmoi renders and applies your files; if it is missing,
setup offers to install it (`brew install chezmoi`, else from get.chezmoi.io
into `~/.local/bin`). The repo stays a valid chezmoi repo throughout — you
can leave casa at any time and keep everything.

### The questionnaire

casa parses the repo's `.casa.toml.tmpl` for every `promptString` /
`promptBool` / `promptInt` / `promptChoice` / `promptMultichoice` call (and
`*Once` variants), including choices and defaults — dotted defaults like
`.chezmoi.hostname` are resolved from template data. Each question is asked in
casa's own UI, then all answers go to chezmoi in one non-interactive shot:

```bash
chezmoi init --promptString "your email=carlos@example.com" --promptBool "work machine=false"
```

Answers are stored per machine in `~/.config/chezmoi/chezmoi.toml`. Prompts
casa cannot parse fall through to chezmoi's own terminal prompting; a repo
with no config template gets a plain `chezmoi init`.

### Homebrew and apply

If `brew` is missing at this point, setup offers the official installer
(`NONINTERACTIVE=1`). Declining is fine — package installs simply skip until
brew exists (`casa machine doctor` shows how to get it). Finally setup runs
`chezmoi apply`: dotfiles are written and the generated run scripts execute —
key restore first (`run_once_before`), then packages, then sh-tools.

## Pull

```bash
casa pull
```

Pull brings the machine fully up to date in both directions; every difference
resolves by an explicit choice:

1. **Push first** — unsaved local changes are listed (by readable target path)
   and offered as a commit + push. Decline to leave them uncommitted.
2. **Drift review** — files changed outside casa get a keep-or-restore review
   (same as `casa files drift`) before anything applies over them.
3. **Pull + apply** — when brew is installed, `brew update` / `upgrade` /
   `cleanup` run; then the repo is pulled and applied (`chezmoi update`).
   Applying converges tools too: the manifest is piped to
   `brew bundle --file=-` and anything removed from
   `.casadata/packages.toml` is **uninstalled** (`brew bundle cleanup`), not
   left behind. No Brewfile ever exists on disk — never create one.

Tell the user to restart their shell afterwards.

## Push

```bash
casa push
casa push "tune tmux keybindings"
```

Lists pending changes by readable target path (`~/.zshrc`, not `dot_zshrc`),
stages everything, commits, pushes. Without a message casa builds one from the
changed basenames: `casa: update .zshrc, .gitconfig` (`… and 2 more` past
three). If the network push fails, the commit stays local — retry with
another `casa push`. Prefer `casa push` over raw `git` in the repo; git
*reads* are fine.

## Status

```bash
casa status
```

```
machine:           mbp
to push:           2 change(s)
to pull:           1 commit(s)
local drift:       3 file(s) need apply
outdated tools:    4
```

| Line | Source |
| --- | --- |
| to push | uncommitted files in the source repo (`git status`) |
| to pull | commits behind the remote (`git rev-list HEAD..@{u}`) |
| local drift | managed files differing from the repo (`chezmoi status`) |
| outdated tools | outdated packages across all managers |

The explicit `status` command computes the outdated count synchronously (can
take seconds); the interactive menu shows git counts instantly and fills in
outdated in the background.

## Answers

```bash
casa machine answers          # picker over all questions + current values
casa machine answers email    # fuzzy match on question text and data keys
```

Re-asks setup questions for this machine. An `everything · ask all questions
again` row re-asks the lot. Unchanged answers pass through untouched, so
chezmoi never re-asks them; casa runs `chezmoi init --prompt` plus the
collected `--promptX` flags (`--prompt` forces `*Once` questions to
re-evaluate), then applies. Interactive — needs a tty.

## Authoring questions

```bash
casa machine question
```

Appends a new question to `.casa.toml.tmpl` and answers it here immediately.
You give a data key (letters/digits/underscores, letter first), the question
text, and a kind: text (`promptStringOnce`), yes/no (`promptBoolOnce`), one of
a list (`promptChoiceOnce`), several of a list (`promptMultichoiceOnce`), or
number (`promptIntOnce`). List kinds take ≥2 comma-separated choices. casa
inserts two lines — the prompt assignment above `[data]`, the entry below it:

```
{{- $editor := promptChoiceOnce . "editor" "which editor?" (list "vim" "helix") }}

[data]
    editor = {{ $editor | quote }}
```

Then it renders the machine config and offers to save as
`casa: add setup question <key>`. Use `{{ .editor }}` in any template after.

## Undo

```bash
casa machine undo
```

Shows the last commit's one-line summary, confirms, then
`git revert --no-edit HEAD`, pushes, and re-applies. History is never
rewritten — a second push or undo can bring the change back.

## Doctor and info

`casa machine doctor` prints a dependency table — every binary casa shells
out to (`git`, `chezmoi`, `brew`, `age`, `go`, `uv`, `npm`, `bun`, `cargo`),
why, and how to install missing ones — then runs `chezmoi doctor`. Missing
managers are fine; casa skips ones you don't use. Run it first when anything
behaves oddly.

`casa machine info` prints machine name (hostname, lowercased, `.local`
stripped), repo path, and managed file count.

## Provisioning a VPS, step by step

Example: a Linux server that should get your shell config and CLI tools but
none of your personal identity.

1. **Repo prep (once, from any machine).** Gate identity files behind a
   questionnaire answer. With a `promptBoolOnce`-backed `personal` key
   (create it with `casa machine question`), commit gates in `.casaignore`:

   ```
   {{ if not .personal }}
   .ssh/**
   .gnupg/**
   .netrc
   {{ end }}
   ```

   Files matched by `.casaignore` are never rendered on that machine — the
   same repo produces a full setup on your Mac (`personal=true`) and a
   stripped one on the server (`personal=false`). Manifest sections handle
   the rest automatically: `cask`, `brew_darwin`, and `extra_darwin` never
   render off macOS, and `system` entries (distro packages like `zsh`)
   never render on it.

2. **Key backups (optional but recommended).** If any secrets should reach
   the server, back their key up first: `casa secrets keys` → pick the key →
   "backup to repo (passphrase)". This writes an armored, passphrase-sealed
   `.casa/keys/<name>.key.age` — safe to commit — then `casa push`.
   Skipping this is fine when the server needs no secrets.

3. **Bootstrap over ssh** (needs a tty; user needs sudo for Linux prereqs):

   ```bash
   ssh you@server
   curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <github-user>
   ```

4. **Answer the questionnaire** in casa's UI — e.g. `personal` → no.

5. **Key restore.** During apply, the generated `run_once_before` restore
   script globs `.casa/keys/*.key.age` and prompts for each key's
   passphrase, restoring identities into `~/.config/casa/keys`. It is fully
   generic (no key names inside), and skips cleanly if `age` is not
   installed yet or the key already exists. Alternatives that avoid the
   repo entirely: `scp ~/.config/casa/keys/main.txt server:~/.config/casa/keys/`
   before setup, or `casa secrets keys` → pull from doppler after.

6. **Verify.** `casa status` (all zeros), `casa machine doctor`, restart the
   shell. From then on the server converges with `casa pull`.

Keys are plain files in `~/.config/casa/keys/` — never in a repo, never
committed. Never generate, echo, or store passphrases on the user's behalf.

## Non-interactive operation (for agents)

- `CASA_YES=1` auto-answers **confirmation** prompts only (prints
  `<question> → yes (CASA_YES)`); selection and text prompts still render
  and will hang without a tty. So `CASA_YES=1 casa pull` is safe only when
  `casa status` shows no drift; `CASA_YES=1 casa push "msg"` is safe.
- Interactive-only (never run headless): `casa` (menu), `machine setup`,
  `answers`, `question`, `undo` (confirm), the drift review, and anything
  prompting for passphrases.
- `CASA_SOURCE` overrides the repo location for both setup (clone target)
  and every later command. `CASA_PLAIN_PATH=1` disables casa's PATH
  self-healing in sandboxes that mask package managers deliberately.
- Exit status: unknown commands print usage and exit 1.

## Guardrails

- Never present drift or manifest edits as free: applying **uninstalls**
  packages removed from the manifest, and restore overwrites local file
  changes. Surface `casa status` before pulling.
- Never create a Brewfile, commit `.chezmoi*`-named files, or hand-write
  `run_*` scripts — casa generates all of that, gitignored, from the
  installed binary.
- Don't rewrite history in the repo; `casa machine undo` is the supported
  revert.
- Keys and passphrases are the user's alone: don't read
  `~/.config/casa/keys/*.txt`, don't invent passphrases, don't move keys
  without being asked.
