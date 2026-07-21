# Getting started

casa is an interactive front-end for [chezmoi](https://www.chezmoi.io/). It shells out to chezmoi for every operation — your dotfiles repo stays a normal chezmoi repo — and adds a menu, package management, and registry-free encryption keys on top.

There are three ways to start, depending on where you are today:

1. [Starting from nothing](#starting-from-nothing) — no dotfiles repo yet.
2. [Setting up a new machine](#setting-up-a-new-machine) — you already have a casa repo.
3. [Adopting casa in an existing chezmoi repo](#adopting-casa-in-an-existing-chezmoi-repo).

## Starting from nothing

### Install casa

```console
$ brew install carrots-sh/tap/casa
```

Or, without Homebrew: download a release binary from GitHub, or build from source:

```console
$ go install github.com/carrots-sh/casa/cmd/casa@latest
```

### Create the repo

Create a repository named `dotfiles` on GitHub (empty is fine), then point casa at it:

```console
$ casa machine setup <your-github-username>
```

If chezmoi is not installed yet, casa offers to install it first:

```text
casa drives chezmoi under the hood, and it isn't installed yet.
install chezmoi now?
```

casa uses Homebrew when available, and chezmoi's own installer (into `~/.local/bin`) otherwise. It then clones your repo into `~/.local/share/casa` — the source directory casa uses from here on (override with `CASA_SOURCE`):

```text
setting up this machine from git@github.com:you/dotfiles.git
  into ~/.local/share/casa ...
```

An empty repo has no setup questions and no packages, so setup finishes immediately with `applying your dotfiles...`. Everything else is created lazily, the first time you use each feature.

### Track your first config

```console
$ casa configs track ~/.zshrc
```

casa suggests common files, completes paths as you type, and asks how to store the file — plain, template, encrypted, or encrypted template — with a sensible default based on the file. See [configs](configs.md).

### Add your first tool

```console
$ casa tools add
```

The first `tools add` bootstraps package management:

```text
casa records tools in .casadata/packages.toml and installs them
on every machine via chezmoi (this repo has no manifest yet).
set up package management now?
```

Accepting creates the manifest and its (gitignored, casa-generated) install scripts, then offers to seed it:

```text
import the tools already installed on this machine?
```

Accept and casa scans brew, cask, go, uv, npm, bun, and cargo, and records everything it finds. From then on, every machine that applies this repo installs the same tools. casa then offers to commit (`casa: set up package manifest`). See [tools](tools.md).

### Add your first secret

Encryption requires `age` (`brew install age` — or just `casa tools add age`). Then:

```console
$ casa secrets add ~/.netrc
```

The first secret creates your encryption key:

```text
no encryption key yet — creating one.
✓ created ~/.config/casa/keys/main.txt (back it up! without it your secrets are unreadable)
✓ encrypted with main and now managing ~/.netrc
```

A key is just a file in `~/.config/casa/keys/` — nothing key-related is stored in the repo. Back the key up right away:

```console
$ casa secrets keys
```

Pick `main`, then `backup to repo (passphrase)`:

```text
choose a passphrase — a new machine needs it (plus the repo) to restore this key.
```

This commits a passphrase-sealed copy (`.casa/keys/main.key.age`) plus a restore script that runs automatically on new machines. See [secrets](secrets.md).

### Author your first setup question

Setup questions let one repo produce different results per machine (work vs. personal, email addresses, feature toggles):

```console
$ casa machine question
```

casa asks, in order:

```text
data key (used as {{ .key }} in templates)
question to ask when setting up a machine
what kind of answer?          text · yes / no · one of a list · several of a list · number
```

List kinds also ask for `choices (comma-separated)`. casa writes the question into the repo's questionnaire (`.casa.toml.tmpl`), asks it on this machine immediately, and confirms:

```text
✓ added — use {{ .work }} in any template
```

Every future `casa machine setup` asks this question during provisioning.

### Save

Most actions offer to commit as you go. To commit and push everything pending:

```console
$ casa save
```

With no message, casa builds one from the changed files and shows the changes by their real paths (`~/.zshrc`, not source names) before committing.

## Setting up a new machine

You have a casa repo; this machine has nothing on it.

### The one-liner

```console
$ curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <your-github-username>
```

This installs Homebrew if missing, installs casa from the tap, and runs `casa machine setup <your-github-username>` — dropping you into the prompt sequence described below (with the chezmoi and Homebrew steps typically already satisfied).

### Binary-only (headless VPS)

On a minimal server where you'd rather not start with Homebrew, install the two things casa cannot install for itself:

```console
$ sudo apt-get update && sudo apt-get install -y git curl
```

If your repo contains encrypted secrets, also install `age` now (`sudo apt-get install -y age`) — the key-restore step needs it during the first apply.

Then get a casa binary — download one from the GitHub releases page, or:

```console
$ go install github.com/carrots-sh/casa/cmd/casa@latest
```

### The setup sequence

```console
$ casa machine setup <your-github-username>
```

Prompts appear in this order (each only when it applies):

1. **chezmoi.** If chezmoi is missing:

   ```text
   casa drives chezmoi under the hood, and it isn't installed yet.
   install chezmoi now?
   ```

   Without Homebrew, casa installs it via `get.chezmoi.io` into `~/.local/bin`.

2. **The repo.** If you gave no argument (and none is configured), casa asks:

   ```text
   github username, user/repo, or repo url
   ```

   A bare username means `<user>/dotfiles`. casa prefers SSH and falls back automatically (`ssh not available, trying https...`), then clones into `~/.local/share/casa`.

3. **The questionnaire.** Every question in the repo's `.casa.toml.tmpl` is asked in casa's own UI — text inputs, yes/no confirms, and pickers, with defaults prefilled. Answers are handed to `chezmoi init` non-interactively; any prompt casa cannot parse falls through to chezmoi's own terminal prompting.

4. **Homebrew.** If brew is missing:

   ```text
   Homebrew isn't installed — install it now? (your packages need it)
   ```

   Defaults to yes. Declining is fine — casa continues, and packages simply skip until brew exists (`casa machine doctor` shows how).

5. **Key restore.** casa prints `applying your dotfiles...` and hands off to `chezmoi apply`. If the repo carries key backups, the generated restore script runs before anything needs decrypting:

   ```text
   Restoring age key 'main' (enter its passphrase):
   ```

   Enter the passphrase you chose when backing the key up. If `age` is not installed yet, the script warns and skips instead of failing:

   ```text
   WARNING: age not installed yet — restore keys later via: casa secrets keys
   ```

6. **Packages.** The apply then installs everything in your manifest (brew/cask/go/uv/npm/bun/cargo and `sh`-installed tools) via casa's generated run scripts.

When it finishes, restart your shell so `PATH` picks up the newly installed tools.

For unattended provisioning, `CASA_YES=1` answers yes to every confirmation prompt.

## Adopting casa in an existing chezmoi repo

casa is backward-compatible with a plain chezmoi setup. Nothing is renamed and nothing is required of your repo.

### Point casa at your repo

Install casa (see above) and run it:

```console
$ casa
```

casa resolves its source directory in order: `$CASA_SOURCE`, then `~/.local/share/casa` if it is a git repo, then **chezmoi's own configured source** (`chezmoi source-path`). An existing `~/.local/share/chezmoi` setup is found automatically — your `.chezmoi.toml.tmpl`, `.chezmoiignore`, and `run_` scripts all keep working as-is, and `chezmoi` on the command line keeps working alongside casa.

The only note: special files that *casa* creates (the questionnaire, `.casadata/`) use casa names in the repo, mirrored to the chezmoi names via gitignored symlinks that casa recreates before every chezmoi call. On a fresh clone of such a repo, run casa once before using bare chezmoi. See [Repository layout](repo-layout.md).

### Import your Brewfile

If your repo manages packages through a Brewfile, the first `casa tools add` offers a migration:

```text
casa records tools in .casadata/packages.toml and installs them
on every machine via chezmoi (this repo has no manifest yet).
set up package management now?
found an existing Brewfile setup (dot_Brewfile.tmpl).
import its packages into the manifest and retire it?
```

Accepting imports every `tap`/`brew`/`cask` (and `go`/`uv`/`npm`/`cargo`) line into the manifest, then retires the old setup: the Brewfile source is deleted, any old run script that fed it to `brew bundle` is deleted, and the rendered `~/.Brewfile` is unmanaged and removed — the manifest now renders straight into `brew bundle` at apply time. One caveat casa prints itself:

```text
note: everything imported as cross-platform; move macOS-only entries
      to brew_darwin/cask in the manifest if you also run Linux.
```

### Existing age key

If you kept an age identity at `~/key.txt` (a common plain-chezmoi convention), the first secrets operation adopts it:

```text
moving your age key into casa's keys dir: ~/key.txt → ~/.config/casa/keys/main.txt
```

casa also replaces the `[age]` block in your config template with a generic one that discovers keys from `~/.config/casa/keys` at init time — no key names, paths, or recipients are ever written into the repo. See [secrets](secrets.md).
