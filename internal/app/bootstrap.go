package app

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/manifest"
	"github.com/carrots-sh/casa/internal/pm"
	"github.com/carrots-sh/casa/internal/ui"
)

// ensurePkg makes sure the package manifest and its install scripts exist,
// offering to create them on first use — seeded from a legacy Brewfile when
// one exists, or from what's already installed on this machine. Returns
// ok=false if the user declines (casa still installs, it just won't record).
func ensurePkg() (manifest.Manifest, bool, error) {
	m := mf()
	if m.Configured() {
		return m, true, nil
	}
	fmt.Println("casa records tools in " + manifest.DefaultRel + " and installs them")
	fmt.Println("on every machine via chezmoi (this repo has no manifest yet).")
	ok, err := ui.Confirm("set up package management now?")
	if err != nil || !ok {
		return m, false, err
	}
	created, err := manifest.Bootstrap(chez.SourceDir(), m.Path)
	if err != nil {
		return m, false, err
	}
	chez.EnsureMirrors(chez.SourceDir()) // .casadata needs its .chezmoidata symlink now
	for _, f := range created {
		fmt.Println("  + " + f)
	}
	if legacy := findLegacyBrewfiles(); len(legacy) > 0 {
		fmt.Printf("found an existing Brewfile setup (%s).\n", strings.Join(legacy, ", "))
		ok, err := ui.Confirm("import its packages into the manifest and retire it?")
		if err != nil {
			return m, false, err
		}
		if ok {
			if err := migrateBrewfile(m, legacy); err != nil {
				return m, false, err
			}
		}
	} else {
		ok, err := ui.Confirm("import the tools already installed on this machine?")
		if err != nil {
			return m, false, err
		}
		if ok {
			importMachine(m)
		}
	}
	offerSave("casa: set up package manifest")
	return m, true, nil
}

// ImportTools seeds the manifest from everything installed on this machine
// (idempotent — already-recorded packages are skipped).
func ImportTools() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	m, ok, err := ensurePkg()
	if err != nil || !ok {
		return err
	}
	importMachine(m)
	offerSave("casa: import installed tools")
	return nil
}

// importMachine scans each package manager and records what it finds.
func importMachine(m manifest.Manifest) {
	total := 0
	for _, mgr := range []string{"tap", "brew", "cask", "go", "uv", "npm", "cargo"} {
		section := manifest.SectionFor(mgr)
		existing, _ := m.List(section)
		have := map[string]bool{}
		for _, e := range existing {
			have[e] = true
		}
		n := 0
		for _, name := range pm.Installed(mgr) {
			if have[name] {
				continue
			}
			if err := m.Add(section, name); err != nil {
				fmt.Printf("  (couldn't record %s %q: %v)\n", mgr, name, err)
				continue
			}
			have[name] = true
			n++
		}
		if n > 0 {
			fmt.Printf("  + %d from %s\n", n, mgr)
			total += n
		}
	}
	if total == 0 {
		fmt.Println("  nothing new to import")
	}
}

// legacyBrewfileNames are source files an older Brewfile-based setup used.
var legacyBrewfileNames = []string{"dot_Brewfile.tmpl", "dot_Brewfile", "Brewfile.tmpl", "Brewfile"}

// findLegacyBrewfiles returns the legacy Brewfile sources present in the repo.
func findLegacyBrewfiles() []string {
	var found []string
	for _, n := range legacyBrewfileNames {
		if _, err := os.Stat(filepath.Join(chez.SourceDir(), n)); err == nil {
			found = append(found, n)
		}
	}
	return found
}

var brewfileLine = regexp.MustCompile(`^(tap|brew|cask|go|uv|npm|cargo) "([^"]+)"`)

// migrateBrewfile imports the rendered ~/.Brewfile into the manifest, then
// retires the legacy setup: deletes the Brewfile source(s) and any old
// run script that ran brew bundle against it, and unmanages ~/.Brewfile.
// The old rendered file must go, or its cleanup step would fight the manifest.
func migrateBrewfile(m manifest.Manifest, legacy []string) error {
	home, _ := os.UserHomeDir()
	rendered := filepath.Join(home, ".Brewfile")
	content, err := os.ReadFile(rendered)
	if err != nil {
		// not applied on this machine — render it from source instead
		if s, cerr := chez.Cat(rendered); cerr == nil {
			content = []byte(s)
		} else {
			return fmt.Errorf("could not read or render ~/.Brewfile: %w", err)
		}
	}
	n := 0
	for l := range strings.SplitSeq(string(content), "\n") {
		mm := brewfileLine.FindStringSubmatch(strings.TrimSpace(l))
		if mm == nil {
			continue
		}
		if err := m.Add(manifest.SectionFor(mm[1]), mm[2]); err != nil {
			return err
		}
		n++
	}
	fmt.Printf("  + imported %d packages from the Brewfile\n", n)
	fmt.Println("  note: everything imported as cross-platform; move macOS-only entries")
	fmt.Println("        to brew_darwin/cask in the manifest if you also run Linux.")

	src := chez.SourceDir()
	for _, f := range legacy {
		if err := os.Remove(filepath.Join(src, f)); err == nil {
			fmt.Println("  - " + f)
		}
	}
	for _, s := range oldBrewBundleScripts(src) {
		if err := os.Remove(filepath.Join(src, s)); err == nil {
			fmt.Println("  - " + s)
		}
	}
	_ = chez.Forget(rendered)
	if err := os.Remove(rendered); err == nil {
		fmt.Println("  - ~/.Brewfile (no longer needed — the manifest renders straight into brew bundle)")
	}
	return nil
}

// oldBrewBundleScripts finds run scripts that fed the legacy ~/.Brewfile to
// brew bundle (but not casa's own manifest-driven script).
func oldBrewBundleScripts(src string) []string {
	ents, err := os.ReadDir(src)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		name := e.Name()
		if e.IsDir() || !strings.HasPrefix(name, "run_") || name == manifest.ScriptPackages {
			continue
		}
		b, err := os.ReadFile(filepath.Join(src, name))
		if err != nil {
			continue
		}
		if strings.Contains(string(b), "brew bundle") && strings.Contains(string(b), "Brewfile") {
			out = append(out, name)
		}
	}
	return out
}
