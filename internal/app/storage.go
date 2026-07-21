// Storage attributes: how a managed file is stored (template/encrypted/…).
package app

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

// attrOpts are the storage attributes casa lets you toggle after tracking.
var attrOpts = []string{"template", "encrypted", "private", "executable"}

// ChangeStorage toggles how a managed file is stored (template, encrypted, …)
// by renaming/re-encoding its source via chezmoi chattr.
func ChangeStorage(name string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	sel, err := pickManaged("change storage of which file?", name)
	if err != nil || sel == "" {
		return err
	}
	cur, err := sourceAttrs(sel)
	if err != nil {
		return err
	}
	var preset []string
	for _, a := range attrOpts {
		if cur[a] {
			preset = append(preset, a)
		}
	}
	want, err := ui.MultiSelect("how should "+tilde(sel)+" be stored?", attrOpts, preset...)
	if err != nil {
		return err
	}
	wantSet := map[string]bool{}
	for _, a := range want {
		wantSet[a] = true
	}
	var mods []string
	for _, a := range attrOpts {
		switch {
		case wantSet[a] && !cur[a]:
			mods = append(mods, "+"+a)
		case !wantSet[a] && cur[a]:
			mods = append(mods, "-"+a)
		}
	}
	if len(mods) == 0 {
		fmt.Println("nothing to change.")
		return nil
	}
	if err := chez.Chattr(strings.Join(mods, ","), homePath(sel)); err != nil {
		return err
	}
	if len(want) == 0 {
		fmt.Printf("✓ %s is now stored plain\n", tilde(sel))
	} else {
		fmt.Printf("✓ %s is now stored: %s\n", tilde(sel), strings.Join(want, ", "))
	}
	if wantSet["template"] && !cur["template"] {
		fmt.Printf("  tip: casa edit %s to add {{ … }} per-machine bits\n", filepath.Base(sel))
	}
	offerSave("casa: change storage of " + filepath.Base(sel))
	return nil
}

// sourceAttrs reads a managed target's storage attributes from its source filename.
func sourceAttrs(target string) (map[string]bool, error) {
	srcs, err := chez.SourcePaths([]string{homePath(target)})
	if err != nil || len(srcs) != 1 {
		return nil, fmt.Errorf("couldn't find the source of %s", target)
	}
	return attrsFromSourceName(filepath.Base(srcs[0])), nil
}

// attrsFromSourceName decodes chezmoi's filename attributes we care about.
func attrsFromSourceName(base string) map[string]bool {
	name := strings.TrimSuffix(strings.TrimSuffix(base, ".age"), ".asc")
	return map[string]bool{
		"template":   strings.HasSuffix(name, ".tmpl"),
		"encrypted":  strings.Contains(base, "encrypted_"),
		"private":    strings.Contains(base, "private_"),
		"executable": strings.Contains(base, "executable_"),
	}
}

// storageBadges maps each target to a "  (template, encrypted)" suffix, empty
// for plain files. Best-effort: on any error every badge is just "".
func storageBadges(targets []string) map[string]string {
	out := map[string]string{}
	homes := make([]string, len(targets))
	for i, t := range targets {
		homes[i] = homePath(t)
	}
	srcs, err := chez.SourcePaths(homes)
	if err != nil || len(srcs) != len(targets) {
		return out
	}
	for i, t := range targets {
		a := attrsFromSourceName(filepath.Base(srcs[i]))
		var tags []string
		for _, k := range []string{"template", "encrypted"} {
			if a[k] {
				tags = append(tags, k)
			}
		}
		if len(tags) > 0 {
			out[t] = "  (" + strings.Join(tags, ", ") + ")"
		}
	}
	return out
}
