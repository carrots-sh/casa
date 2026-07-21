package manifest

import (
	"os"
	"path/filepath"
)

// Script filenames casa maintains in the source dir. Both are chezmoi
// run_onchange scripts: they re-run on apply whenever their rendered content
// changes — i.e. whenever the relevant part of the manifest changes.
const (
	ScriptPackages = "run_onchange_after_20-packages.sh.tmpl"
	ScriptShTools  = "run_onchange_after_30-sh-tools.sh.tmpl"
)

// skeleton is the initial manifest for a repo that has never used one.
const skeleton = `# casa's package manifest — the single source of truth for tools on your machines.
# Edit freely (casa edits it too; your comments are preserved).
# On every machine, ` + "`chezmoi apply`" + ` installs what's listed here and uninstalls
# what you remove (brew bundle does the diffing).

[packages]

# Homebrew taps
taps = [
]

# Taps whose formulae brew bundle may manage without prompting
# (casa tools trust moves taps here)
taps_trusted = [
]

# CLI tools — cross-platform (macOS + Linux via Homebrew)
brew = [
]

# macOS-only formulae
brew_darwin = [
]

# macOS apps + fonts (casks)
cask = [
]

# go install
go = [
]

# uv tool install
uv = [
]

# npm install -g
npm = [
]

# bun add -g
bun = [
]

# cargo install
cargo = [
]

# Raw Brewfile lines passed through verbatim — for anything with extra
# arguments (custom tap URLs, link: false, trusted: true, mas/vscode
# directives, ...). Hand-managed; casa's add/remove don't touch these.
extra = [
]

# Same, but only rendered on macOS.
extra_darwin = [
]

# Tools that ship their own installer are recorded as blocks, e.g.:
# [[packages.sh]]
# bin = "herdr"
# install = "curl -fsSL https://herdr.dev/install.sh | sh"
# update = "herdr self-update"  # optional; omit if the tool updates itself
# os = "darwin"                 # optional: darwin | linux
`

// packagesScript pipes the manifest, rendered as a Brewfile, straight into
// brew bundle — no Brewfile ever exists on disk.
const packagesScript = `#!/bin/bash
# Managed by casa — installs everything declared in casa's package manifest.
# Re-runs on ` + "`chezmoi apply`" + ` whenever the rendered package list below changes.
# Removing a package from the manifest + apply → brew bundle cleanup uninstalls it.
set -e
{{ if hasKey . "packages" -}}
command -v brew >/dev/null 2>&1 || { echo "casa: brew not found; skipping packages"; exit 0; }

brewfile=$(cat <<'BREWFILE'
{{ range index .packages "taps" }}tap "{{ . }}"
{{ end }}{{ range index .packages "taps_trusted" }}tap "{{ . }}", trusted: true
{{ end }}{{ range index .packages "extra" }}{{ . }}
{{ end }}{{ range index .packages "brew" }}brew "{{ . }}"
{{ end }}{{ if eq .chezmoi.os "darwin" }}{{ range index .packages "brew_darwin" }}brew "{{ . }}"
{{ end }}{{ range index .packages "cask" }}cask "{{ . }}"
{{ end }}{{ range index .packages "extra_darwin" }}{{ . }}
{{ end }}{{ end }}{{ range index .packages "go" }}go "{{ . }}"
{{ end }}{{ range index .packages "uv" }}uv "{{ . }}"
{{ end }}{{ range index .packages "npm" }}npm "{{ . }}"
{{ end }}{{ range index .packages "cargo" }}cargo "{{ . }}"
{{ end }}BREWFILE
)

# An empty manifest renders an empty Brewfile, and cleanup --force against an
# empty Brewfile would uninstall everything. Never run against an empty list.
[ -n "${brewfile//[[:space:]]/}" ] || { echo "casa: package manifest is empty; skipping"; exit 0; }

printf '%s\n' "$brewfile" | brew bundle --file=-

# Uninstall anything installed but no longer declared. brew bundle's uv cleanup
# mis-parses 'uv tool list' (treats the "- <exe>" bullet lines as a tool named
# "-") and prints a harmless "invalid value '-'" error while still exiting 0.
# Filter just that line; preserve the real exit status.
out=$(printf '%s\n' "$brewfile" | brew bundle cleanup --force --file=- 2>&1) && status=0 || status=$?
printf '%s\n' "$out" | grep -vF "invalid value '-'" || true

# bun globals — brew bundle can't manage these. bun add -g is idempotent and
# fast, so just re-assert the list. ponytail: no cleanup diffing; removing a
# bun entry by hand needs a manual bun remove -g (casa's rm does it for you).
{{ if index .packages "bun" -}}
if command -v bun >/dev/null 2>&1; then
{{ range index .packages "bun" }}  bun add -g "{{ . }}"
{{ end }}fi
{{ end -}}
exit "$status"
{{ end -}}
`

// shToolsScript installs the [[packages.sh]] tools; command -v guards make
// re-runs free and the whole script idempotent.
const shToolsScript = `#!/bin/sh
# Managed by casa — tools that ship their own installer, declared under
# [[packages.sh]] in casa's package manifest. Re-runs on ` + "`chezmoi apply`" + `
# when the list changes; the command -v guards keep re-runs free.
set -e
{{ if hasKey . "packages" -}}
{{ range index .packages "sh" -}}
{{ $os := default "" (get . "os") -}}
{{ if or (not $os) (eq $os $.chezmoi.os) -}}
if ! command -v {{ .bin }} >/dev/null 2>&1; then
  echo "casa: installing {{ .bin }}..."
  {{ .install }}
fi
{{ end -}}
{{ end -}}
{{ end -}}
`

// Bootstrap creates the manifest and both run scripts in srcDir, skipping any
// that already exist. manifestPath is the manifest's absolute path. It returns
// the source-relative names of the files it created.
func Bootstrap(srcDir, manifestPath string) ([]string, error) {
	var created []string
	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		if err := os.MkdirAll(filepath.Dir(manifestPath), 0o755); err != nil {
			return created, err
		}
		if err := os.WriteFile(manifestPath, []byte(skeleton), 0o644); err != nil {
			return created, err
		}
		if rel, err := filepath.Rel(srcDir, manifestPath); err == nil {
			created = append(created, rel)
		} else {
			created = append(created, manifestPath)
		}
	}
	for name, content := range map[string]string{
		ScriptPackages: packagesScript,
		ScriptShTools:  shToolsScript,
	} {
		p := filepath.Join(srcDir, name)
		if _, err := os.Stat(p); os.IsNotExist(err) {
			if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
				return created, err
			}
			created = append(created, name)
		}
	}
	return created, nil
}
