# Installation

casa manages your machines — files, tools, and secrets — from one git repo. It
ships as a single static binary, and there are four ways to install it; pick
the one that matches your situation.

| Method | When to use it |
| --- | --- |
| [Bootstrap script](#bootstrap-script) | A fresh machine, especially one without Homebrew yet |
| [Homebrew](#homebrew) | You already have Homebrew |
| [Release binaries](#release-binaries) | No Homebrew, servers, containers, or `.deb`/`.rpm` systems |
| [go install](#go-install) | You have a Go toolchain and want to build from source |

Under the hood, [chezmoi](https://chezmoi.io) renders and applies your files,
so chezmoi must be installed too. The Homebrew formula declares chezmoi as a
dependency, and `casa machine setup` installs chezmoi itself when it is
missing (via Homebrew, or `get.chezmoi.io` as a fallback) — so in practice you
rarely install it by hand.

## Bootstrap script

A fresh machine is one curl away from being yours. The one-liner installs
Homebrew if missing, then casa from the Homebrew tap:

```console
$ curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh
```

Pass your GitHub username (or a full repo) to also provision the machine from
your dotfiles in the same step:

```console
$ curl -fsSL https://raw.githubusercontent.com/carrots-sh/casa/main/install.sh | sh -s -- <github-username>
```

The script does three things, in order:

1. If `brew` is not on `PATH`, first installs Homebrew's Linux prerequisites
   (build tools, `curl`, `file`, `git`) via apt, dnf, or pacman, then runs the
   official Homebrew installer non-interactively and loads `brew shellenv`
   from wherever it landed (`/opt/homebrew`, `/home/linuxbrew/.linuxbrew`, or
   `/usr/local`). `casa machine setup` does the same when it offers to
   install Homebrew.
2. Runs `brew install carrots-sh/tap/casa`.
3. If you passed an argument, runs `casa machine setup <arg>` to clone your
   repo, ask your setup questions, and apply everything — files, tools, and
   secrets. See [Machine setup](machine.md).

Use this on macOS or Linux when you are starting from nothing. If Homebrew is
already installed, the script skips straight to installing casa.

## Homebrew

If you already have Homebrew:

```console
$ brew install carrots-sh/tap/casa
```

The formula depends on `chezmoi` (required) and `git` (optional), so a plain
`brew install` gives you everything casa needs for its core workflow. casa's
package manifest also covers `go`, `uv`, `npm`, `bun`, and `cargo` tools —
install whichever of those managers you use; casa shells out to them on
demand.

## Release binaries

Every release publishes prebuilt archives for:

| OS | Architectures | Format |
| --- | --- | --- |
| macOS (`darwin`) | `amd64`, `arm64` | `.tar.gz` |
| Linux | `amd64`, `arm64` | `.tar.gz`, plus `.deb` and `.rpm` packages |
| Windows | `amd64`, `arm64` | `.zip` |

Archives are named `casa_<version>_<os>_<arch>.tar.gz`. For example, to
install v0.5.1 on an Apple Silicon Mac:

```console
$ curl -LO https://github.com/carrots-sh/casa/releases/download/v0.5.1/casa_0.5.1_darwin_arm64.tar.gz
$ tar xzf casa_0.5.1_darwin_arm64.tar.gz casa
$ install -m 755 casa ~/.local/bin/casa
```

On Debian/Ubuntu or Fedora/RHEL systems you can install the `.deb` or `.rpm`
instead; those place the binary at `/usr/bin/casa`. Each release also ships a
`checksums.txt` for verification.

Binaries are static (`CGO_ENABLED=0`), so they run on any machine of the right
OS and architecture with no runtime dependencies. Remember that chezmoi — the
engine casa uses to render and apply your files — is still required; releases
only contain casa itself.

## go install

With a Go toolchain:

```console
$ go install github.com/carrots-sh/casa/cmd/casa@latest
```

This builds the latest tagged release from source and puts `casa` in your
`GOBIN` (default `~/go/bin`). The binary picks up its version from Go module
build info, so `casa version` reports the real release tag.

Installing `@main` instead builds an untagged development snapshot; see the
note on dev builds below.

## Verifying

```console
$ casa version
casa v0.5.1 (f56c204)
$ casa machine doctor
```

`casa machine doctor` prints a dependency table (git, chezmoi, brew, age, go,
uv, npm, bun, cargo) and runs `chezmoi doctor`, so it tells you immediately if
anything the workflow needs is missing.

## Upgrading

casa updates itself:

```console
$ casa upgrade
```

This checks the latest GitHub release, and if it is newer than the running
build, downloads the archive for your OS and architecture and atomically
replaces the running binary in place — symlinks (such as Homebrew's
`bin/casa`) are resolved first, so the real file gets replaced. If you are
already on the newest release it says so and exits.

casa also checks for new releases in the background from the interactive menu,
throttled to at most one network request per day (cached in your user cache
directory), and shows an upgrade hint next to the `upgrade` entry when one is
available.

Notes:

- Self-update is not supported on Windows; download the release archive from
  GitHub instead.
- If you installed with Homebrew, `brew upgrade casa` also works and keeps
  Homebrew's own bookkeeping in sync. Either path ends with the same binary.

### Dev builds never self-update silently

Version comparison only considers proper `vMAJOR.MINOR.PATCH` release tags. A
source build (`casa dev`) or a Go pseudo-version from `go install ...@main`
never compares as older than a release, so:

- The background check never flags an upgrade on a dev build.
- Running `casa upgrade` on a dev build asks for explicit confirmation before
  replacing it, since swapping a dev build for the latest release could be a
  downgrade.

## Releases and versioning

casa follows semantic versioning (`vMAJOR.MINOR.PATCH`). Releases are cut from
CI and published to GitHub Releases and the Homebrew tap simultaneously, so
every install method above serves the same versions.
