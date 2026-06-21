# casa

Interactive package-manager front-end that keeps your **Brewfile** in sync.

`casa` wraps `brew`, `cask`, `tap`, `go`, `uv`, `npm`, and `cargo` behind one
command. Install or remove something through `casa` and it records the change in
your chezmoi-managed `~/.Brewfile` automatically — no `brew bundle dump`, no drift.

## Install

```bash
brew install carrots-sh/tap/casa
```

## Usage

```bash
casa            # menu: add / remove / update
casa add        # pick a manager → name → install + record
casa remove     # pick from ALL recorded packages (any manager) → uninstall + de-record
casa update     # pick from outdated packages → upgrade one / many / all
```

Non-interactive forms also work:

```bash
casa add uv ruff
casa remove brew ripgrep
```

### How sync works

`casa` edits the Brewfile **template** in your chezmoi source (`dot_Brewfile.tmpl`),
inserting each entry just before the matching `# casa:<manager>` anchor so it lands
in the OS-correct section, then runs `chezmoi apply` to refresh `~/.Brewfile`. It
offers to commit + push the change via chezmoi's git.

`remove` lists every recorded package across all managers in a single picker, so you
don't have to remember how you installed something — just select it.

`update` shows per-package outdated info for brew, casks, and global npm packages;
uv, go, and cargo are offered as manager-wide upgrades (they don't expose per-package
outdated cleanly).

## License

MIT
