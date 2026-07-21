package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/home"
	"github.com/carrots-sh/casa/internal/ui"
)

// pickManaged resolves query against the managed files: exact match wins,
// a single substring hit opens directly, several hits show a pre-filtered
// picker (with storage badges), none is an error. Empty query = full picker.
func pickManaged(title, query string) (string, error) {
	managed, err := chez.Managed()
	if err != nil {
		return "", err
	}
	if len(managed) == 0 {
		fmt.Println("no configs yet — start with: casa configs track <path>")
		return "", nil
	}
	var filtered []string
	if query == "" {
		filtered = managed
	} else {
		for _, m := range managed {
			if m == query {
				return m, nil
			}
			if strings.Contains(strings.ToLower(m), strings.ToLower(query)) {
				filtered = append(filtered, m)
			}
		}
		switch len(filtered) {
		case 1:
			return filtered[0], nil
		case 0:
			return "", fmt.Errorf("nothing managed matches %q", query)
		}
	}
	badges := storageBadges(filtered)
	labels := make([]string, len(filtered))
	byLabel := make(map[string]string, len(filtered))
	for i, m := range filtered {
		labels[i] = home.Tilde(m) + badges[m]
		byLabel[labels[i]] = m
	}
	sel, err := ui.Select(title, labels)
	return byLabel[sel], err
}

// EditConfig fuzzy-picks any managed file and edits it. Encrypted files route
// through the secret flow (template validation + same-key re-seal) — the
// action is "edit"; the storage type is casa's problem.
func EditConfig(name string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	sel, err := pickManaged("edit which file?", name)
	if err != nil || sel == "" {
		return err
	}
	if attrs, err := sourceAttrs(sel); err == nil && attrs["encrypted"] {
		srcs, err := chez.SourcePaths([]string{home.Path(sel)})
		if err != nil || len(srcs) != 1 {
			return fmt.Errorf("couldn't find the encrypted source of %s", home.Tilde(sel))
		}
		rel, err := filepath.Rel(chez.SourceDir(), srcs[0])
		if err != nil {
			return err
		}
		return editSecretSource(rel, home.Tilde(sel))
	}
	if err := chez.Edit(home.Path(sel)); err != nil {
		return err
	}
	fmt.Printf("✓ edited %s\n", home.Tilde(sel))
	offerSave("casa: edit " + sel)
	return nil
}

// UntrackFile stops managing a file but keeps it on disk.
func UntrackFile(path string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if path != "" {
		if _, err := os.Stat(home.Expand(path)); err == nil {
			path = home.Expand(path)
		} else {
			sel, err := pickManaged("stop managing which file? (it stays on disk)", path)
			if err != nil || sel == "" {
				return err
			}
			path = home.Path(sel)
		}
	} else {
		sel, err := pickManaged("stop managing which file? (it stays on disk)", "")
		if err != nil || sel == "" {
			return err
		}
		path = home.Path(sel)
	}
	if err := chez.Forget(path); err != nil {
		return err
	}
	fmt.Printf("✓ no longer managing %s (file kept)\n", home.Tilde(path))
	offerSave("casa: untrack")
	return nil
}

// configLines renders all managed files with their storage badges.
func configLines() ([]string, error) {
	managed, err := chez.Managed()
	if err != nil {
		return nil, err
	}
	badges := storageBadges(managed)
	lines := make([]string, len(managed))
	for i, m := range managed {
		lines[i] = home.Tilde(m) + badges[m]
	}
	return lines, nil
}

// ListConfigs prints all managed files (plain output — pipeable).
func ListConfigs() error {
	lines, err := configLines()
	if err != nil {
		return err
	}
	for _, l := range lines {
		fmt.Println(l)
	}
	return nil
}
