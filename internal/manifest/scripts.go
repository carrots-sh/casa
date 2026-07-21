package manifest

import (
	_ "embed"
	"os"
	"path/filepath"
)

// The script/skeleton bodies live in embedded/ as real files so they stay
// editable, formattable, and lintable as shell/TOML.

// skeleton is the initial manifest for a repo that has never used one.
//
//go:embed embedded/packages.toml
var skeleton string

// PackagesScript renders the manifest as a Brewfile piped straight into
// brew bundle — no Brewfile ever exists on disk.
//
//go:embed embedded/packages.sh.tmpl
var PackagesScript string

// ShToolsScript installs the [[packages.sh]] tools; command -v guards make
// re-runs free and the whole script idempotent.
//
//go:embed embedded/sh-tools.sh.tmpl
var ShToolsScript string

// Script filenames casa maintains in the source dir. Both are chezmoi
// run_onchange scripts: they re-run on apply whenever their rendered content
// changes — i.e. whenever the relevant part of the manifest changes.
const (
	ScriptPackages = "run_onchange_after_20-packages.sh.tmpl"
	ScriptShTools  = "run_onchange_after_30-sh-tools.sh.tmpl"
)

// Bootstrap creates the manifest skeleton if missing and returns the created
// source-relative names. The run scripts are NOT committed repo content —
// casa generates them (gitignored) before every chezmoi call.
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
	return created, nil
}
