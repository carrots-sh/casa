// Generated run scripts: casa default behavior (package install, sh tools,
// key restore) never lives in a repo — the scripts are written into the
// source dir gitignored and refreshed from THIS casa's embedded templates
// before every chezmoi call, so behavior always matches the installed casa.
package app

import (
	"os"
	"path/filepath"

	"github.com/carrots-sh/casa/internal/agekey"
	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/manifest"
)

func init() {
	chez.OnPrepare(ensureGenerated)
}

// ensureGenerated writes each generated script when its feature is in use
// and the on-disk copy is missing or stale. Content is casa-versioned; the
// files are gitignored so repos only carry data.
func ensureGenerated(src string) {
	type gen struct {
		name, body string
		when       bool
	}
	m := manifest.Manifest{Path: filepath.Join(src, manifest.DefaultRel)}
	if !m.Configured() {
		m.Path = filepath.Join(src, manifest.ChezmoiRel)
	}
	keysBackedUp := false
	if fi, err := os.Stat(filepath.Join(src, agekey.BackupRel)); err == nil && fi.IsDir() {
		keysBackedUp = true
	}
	var names []string
	for _, g := range []gen{
		{manifest.ScriptPackages, manifest.PackagesScript, m.Configured()},
		{manifest.ScriptShTools, manifest.ShToolsScript, m.Configured()},
		{agekey.RestoreScript, agekey.RestoreScriptBody, keysBackedUp},
	} {
		if !g.when {
			continue
		}
		p := filepath.Join(src, g.name)
		if cur, err := os.ReadFile(p); err != nil || string(cur) != g.body {
			_ = os.WriteFile(p, []byte(g.body), 0o644)
		}
		names = append(names, g.name)
	}
	chez.EnsureGitignored(src, names)
}
