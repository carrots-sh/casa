# Machine lifecycle

The `casa machine` commands cover the life of a machine: provision it from your
dotfiles repo, keep it in sync, save changes back, and check its health.

```console
$ casa machine setup [repo]     # provision this machine from your dotfiles repo
$ casa machine pull             # repo → this machine (pushes yours first)
$ casa machine push [message]   # your changes → repo (commit + push)
$ casa machine status           # show what's changed, behind, or outdated
$ casa machine answers [name]   # change this machine's setup answers and re-apply
$ casa machine question         # add a setup question to your repo
$ casa machine undo             # revert the last saved change and re-apply
$ casa machine doctor           # health check
$ casa machine info             # machine + repo basics
```

`casa pull`, `casa push`, and `casa status` are top-level shortcuts for the same
commands. `casa sync` and `casa save` still work as legacy aliases.

## Setup

```console
$ casa machine setup carlos
```

The repo argument accepts three forms:

| Argument | Resolved to |
| --- | --- |
| `carlos` | `carlos/dotfiles` on GitHub |
| `carlos/machines` | that GitHub repo |
| `git@…` or any `…://…` URL | used as-is |

With no argument, casa uses the repo remembered in its config, or asks.
For GitHub forms, casa prefers SSH and falls back to HTTPS. Each URL is probed
with `git ls-remote` in a non-interactive mode (no password or host-key prompts
can hang the check) before cloning.

If `chezmoi` is not installed yet, setup offers to install it first — via
`brew install chezmoi` when Homebrew exists, otherwise from `get.chezmoi.io`
into `~/.local/bin`.

### Clone first, then init

Setup pins the source directory to `~/.local/share/casa` (override with
`CASA_SOURCE`) and clones the repo there with plain `git clone` — it does not
use `chezmoi init <repo>`. This ordering is deliberate: casa needs the repo on
disk *before* the first chezmoi call, so it can mirror the casa-named special
files (`.casa.toml.tmpl`, `.casaignore`, `.casadata/`) to the names chezmoi
expects and generate the run scripts. If the target directory already contains
a `.git`, the existing checkout is reused instead of cloning.

### The questionnaire

Next, casa reads the repo's config template (`.casa.toml.tmpl`) and parses
every `promptString`/`promptBool`/`promptInt`/`promptChoice`/`promptMultichoice`
call (and their `*Once` variants), including choices and defaults. Dotted
defaults such as `.chezmoi.hostname` are resolved from the template data.

Each question is asked in casa's own UI, then all answers are handed to chezmoi
non-interactively:

```console
$ chezmoi init --promptString "your email=carlos@example.com" --promptBool "work machine=false"
```

chezmoi still does all the rendering, so template semantics stay native. Any
prompt casa fails to parse falls through to chezmoi's own terminal prompting
(`chezmoi init --prompt`). A repo with no config template just gets a plain
`chezmoi init` so a machine config exists.

### Homebrew and apply

If `brew` is missing, setup offers to install it with the official installer
(`NONINTERACTIVE=1`). Declining is fine — package installation simply skips
until brew exists (`casa machine doctor` shows how to get it). Finally, casa
runs `chezmoi apply`, which writes your dotfiles and executes the generated run
scripts (package install, key restore). See [tools](tools.md) and
[secrets](secrets.md).

## Pull

```console
$ casa pull
```

Pull brings the machine fully up to date — in both directions, so every
difference resolves by an explicit choice:

1. **Push:** unsaved local changes are listed and offered as a commit + push
   first (decline to leave them uncommitted).
2. **Drift:** files changed outside casa get the keep-or-restore review
   (the same one as `casa files drift`) before the pull can apply over them.
3. **Pull:** when brew is installed, `brew update` / `upgrade` / `cleanup`
   run, then the repo is pulled and applied (`chezmoi update`).

Restart your shell afterwards to pick up changes.

## Push

```console
$ casa push
$ casa push "tune tmux keybindings"
```

Push lists pending changes by their readable target paths (`~/.zshrc`, not
`dot_zshrc`), then stages everything, commits, and pushes. If you omit the
message, casa builds one from the basenames of the changed files:

```
casa: update .zshrc, .gitconfig
casa: update .zshrc, .gitconfig, .tmux.conf and 2 more
```

If the push fails (offline, no remote access), the commit stays local and casa
tells you to retry with another `casa push`.

## Status

```console
$ casa status
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
| local drift | managed files that differ from the repo (`chezmoi status`) |
| outdated tools | outdated packages across managers |

The interactive menu shows the cheap git-derived counts instantly and computes
the outdated count in the background; the explicit `status` command computes it
synchronously if needed.

## Answers

```console
$ casa machine answers
$ casa machine answers email
```

Answers re-asks setup questions for this machine. With no argument you get a
picker over every question with its current value; an
`everything · ask all questions again` row re-asks the lot. With a name, casa
fuzzy-matches against question text and data keys, showing a picker only on
multiple hits.

Every answer you did not change is passed through unchanged, so chezmoi never
re-asks it. casa runs `chezmoi init --prompt` plus the collected `--promptX`
flags — `--prompt` forces `*Once` questions to re-evaluate instead of reusing
stored values — then applies.

If the repo has a questionnaire casa cannot parse, chezmoi asks its own prompts
on the terminal and casa applies afterwards.

## Authoring questions

```console
$ casa machine question
```

This appends a new setup question to the repo questionnaire and answers it for
this machine right away. You provide a data key (letters, digits, underscores,
starting with a letter), the question text, and the answer kind:

| Kind | Template function | `[data]` value |
| --- | --- | --- |
| text | `promptStringOnce` | quoted string |
| yes / no | `promptBoolOnce` | bare bool |
| one of a list | `promptChoiceOnce` | quoted string |
| several of a list | `promptMultichoiceOnce` | TOML array |
| number | `promptIntOnce` | bare int |

List kinds take comma-separated choices (at least two). casa inserts two lines
into `.casa.toml.tmpl`: the prompt assignment directly above the `[data]`
section, and the data entry directly below it —

```
{{- $editor := promptChoiceOnce . "editor" "which editor?" (list "vim" "helix") }}

[data]
    editor = {{ $editor | quote }}
```

The file is created if it does not exist, and both lines are appended if there
is no `[data]` section yet. casa then asks you the new question, renders the
machine config, and offers to save with the message
`casa: add setup question <key>`. Use `{{ .editor }}` in any template from then
on.

## Undo

```console
$ casa machine undo
```

Undo shows the last commit's one-line summary, asks for confirmation, then
`git revert --no-edit HEAD`, pushes, and re-applies. Because it is a revert, a
second save or undo can bring the change back — history is never rewritten.

## Doctor

```console
$ casa machine doctor
```

Doctor prints a dependency table — every binary casa shells out to, why it is
needed, and how to get it when missing — then runs `chezmoi doctor`:

| Binary | Used for |
| --- | --- |
| `git` | versioning your dotfiles |
| `chezmoi` | the engine behind casa |
| `brew` | brew/cask/tap packages |
| `age` | secret encryption |
| `go` | go installs |
| `uv` | uv tools |
| `npm` | npm globals |
| `bun` | bun globals |
| `cargo` | cargo installs |

Missing managers are fine — casa skips the ones you don't use.

## Info

```console
$ casa machine info
machine:  mbp
repo:     ~/.local/share/casa
managed:  42 files
```

The machine name is the hostname, lowercased with any `.local` suffix
stripped; `managed` counts files under chezmoi management.
