package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

func homePath(rel string) string {
	h, _ := os.UserHomeDir()
	return filepath.Join(h, rel)
}

func expand(p string) string {
	if strings.HasPrefix(p, "~/") {
		h, _ := os.UserHomeDir()
		return filepath.Join(h, p[2:])
	}
	return p
}

// EditConfig fuzzy-picks a managed file and edits it (encrypted ones transparently).
func EditConfig(name string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	sel := name
	if sel == "" {
		managed, err := chez.Managed()
		if err != nil {
			return err
		}
		if len(managed) == 0 {
			return fmt.Errorf("no managed files yet")
		}
		if sel, err = ui.Select("edit which config?", managed); err != nil || sel == "" {
			return err
		}
	}
	if err := chez.Edit(homePath(sel)); err != nil {
		return err
	}
	fmt.Printf("✓ edited %s\n", sel)
	offerSave("casa: edit " + sel)
	return nil
}

// TrackFile starts managing an existing file.
func TrackFile(path string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if path == "" {
		var err error
		if path, err = ui.Input("path of the file to start managing"); err != nil || path == "" {
			return err
		}
	}
	if err := chez.Add(expand(path)); err != nil {
		return err
	}
	fmt.Printf("✓ now managing %s\n", path)
	offerSave("casa: track " + filepath.Base(path))
	return nil
}

// UntrackFile stops managing a file but keeps it on disk.
func UntrackFile(path string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if path == "" {
		managed, err := chez.Managed()
		if err != nil {
			return err
		}
		sel, err := ui.Select("stop managing which file? (it stays on disk)", managed)
		if err != nil || sel == "" {
			return err
		}
		path = homePath(sel)
	} else {
		path = expand(path)
	}
	if err := chez.Forget(path); err != nil {
		return err
	}
	fmt.Printf("✓ no longer managing %s (file kept)\n", path)
	offerSave("casa: untrack")
	return nil
}

// ListConfigs prints all managed files.
func ListConfigs() error {
	managed, err := chez.Managed()
	if err != nil {
		return err
	}
	for _, m := range managed {
		fmt.Println(m)
	}
	return nil
}
