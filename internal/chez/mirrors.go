// casa-name mirrors: the gitignored symlinks that map casa-named special
// files to the names chezmoi hardcodes.
package chez

import (
	"os"
	"path/filepath"
	"strings"
)

// mirrors pairs casa-named special files with the chezmoi names chezmoi
// hardcodes. casa repos commit only the casa names; the chezmoi names are
// gitignored symlinks recreated here on demand. Repos that use chezmoi names
// directly are left untouched (the casa file simply doesn't exist).
// .casadata is a directory — chezmoi follows the symlink for template data.
var mirrors = [][2]string{
	{".casa.toml.tmpl", ".chezmoi.toml.tmpl"},
	{".casa.yaml.tmpl", ".chezmoi.yaml.tmpl"},
	{".casa.json.tmpl", ".chezmoi.json.tmpl"},
	{".casaignore", ".chezmoiignore"},
	{".casaremove", ".chezmoiremove"},
	{".casaversion", ".chezmoiversion"},
	{".casaexternal.toml", ".chezmoiexternal.toml"},
	{".casadata", ".chezmoidata"},
	{".casadata.toml", ".chezmoidata.toml"},
	{".casadata.yaml", ".chezmoidata.yaml"},
	{".casadata.json", ".chezmoidata.json"},
}

// prepare hooks run before every chezmoi invocation (after EnsureMirrors) —
// e.g. the app layer regenerating casa-owned run scripts.
var prepare []func(dir string)

// OnPrepare registers a hook to run before each chezmoi call.
func OnPrepare(f func(dir string)) { prepare = append(prepare, f) }

// EnsureMirrors creates any missing chezmoi-named symlinks for casa-named
// special files in dir, gitignoring the links it creates. Safe to call
// repeatedly; a user's own real chezmoi-named file is never touched.
func EnsureMirrors(dir string) {
	var created []string
	for _, m := range mirrors {
		casa, chz := m[0], m[1]
		if _, err := os.Lstat(filepath.Join(dir, casa)); err != nil {
			continue
		}
		link := filepath.Join(dir, chz)
		if _, err := os.Lstat(link); err == nil {
			continue // already linked, or the user's own real file
		}
		if os.Symlink(casa, link) == nil {
			created = append(created, chz)
		}
	}
	EnsureGitignored(dir, created)
}

// EnsureGitignored appends any missing names to dir's .gitignore.
func EnsureGitignored(dir string, names []string) {
	if len(names) == 0 {
		return
	}
	path := filepath.Join(dir, ".gitignore")
	data, _ := os.ReadFile(path)
	have := map[string]bool{}
	for l := range strings.SplitSeq(string(data), "\n") {
		have[strings.TrimSpace(l)] = true
	}
	var missing []string
	for _, n := range names {
		if !have[n] {
			missing = append(missing, n)
		}
	}
	if len(missing) == 0 {
		return
	}
	s := string(data)
	if s != "" && !strings.HasSuffix(s, "\n") {
		s += "\n"
	}
	if !strings.Contains(s, "chezmoi-named mirrors") {
		s += "# chezmoi-named mirrors of the .casa* files (casa recreates them)\n"
	}
	s += strings.Join(missing, "\n") + "\n"
	_ = os.WriteFile(path, []byte(s), 0o644)
}
