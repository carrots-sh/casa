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

// pickManaged resolves query against the managed files: exact match wins,
// a single substring hit opens directly, several hits show a pre-filtered
// picker, none is an error. Empty query = full picker.
func pickManaged(title, query string) (string, error) {
	managed, err := chez.Managed()
	if err != nil {
		return "", err
	}
	if len(managed) == 0 {
		fmt.Println("no configs yet — start with: casa configs track <path>")
		return "", nil
	}
	if query == "" {
		return ui.Select(title, managed)
	}
	var filtered []string
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
	default:
		return ui.Select(title, filtered)
	}
}

// EditConfig fuzzy-picks a managed file and edits it (encrypted ones transparently).
func EditConfig(name string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	sel, err := pickManaged("edit which config?", name)
	if err != nil || sel == "" {
		return err
	}
	if err := chez.Edit(homePath(sel)); err != nil {
		return err
	}
	fmt.Printf("✓ edited %s\n", sel)
	offerSave("casa: edit " + sel)
	return nil
}

// TrackFile starts managing a file. With no path it offers an "adopt" picker of
// common dotfiles found in $HOME that aren't managed yet.
func TrackFile(path string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if path != "" {
		if err := trackOne(expand(path)); err != nil {
			return err
		}
		offerSave("casa: track " + filepath.Base(path))
		return nil
	}

	if cands := unmanagedCommonDotfiles(); len(cands) > 0 {
		sel, err := ui.MultiSelect("which files should casa manage?", cands)
		if err != nil || len(sel) == 0 {
			return err
		}
		for _, rel := range sel {
			if err := trackOne(homePath(rel)); err != nil {
				fmt.Printf("  (skipped %s: %v)\n", rel, err)
			}
		}
		offerSave("casa: track files")
		return nil
	}

	p, err := ui.Input("path of the file to start managing")
	if err != nil || p == "" {
		return err
	}
	if err := trackOne(expand(p)); err != nil {
		return err
	}
	offerSave("casa: track " + filepath.Base(p))
	return nil
}

// trackOne manages a single file, offering to encrypt it if it looks sensitive.
func trackOne(abs string) error {
	if looksSensitive(abs) {
		if ok, _ := ui.Confirm(filepath.Base(abs) + " looks sensitive — encrypt it?"); ok {
			if err := chez.AddEncrypt(abs); err != nil {
				return err
			}
			fmt.Printf("✓ now managing %s (encrypted)\n", abs)
			return nil
		}
	}
	if err := chez.Add(abs); err != nil {
		return err
	}
	fmt.Printf("✓ now managing %s\n", abs)
	return nil
}

func looksSensitive(path string) bool {
	b := strings.ToLower(filepath.Base(path))
	for _, p := range []string{".env", ".pem", ".key", "credential", "secret", "id_rsa", "id_ed25519", "token"} {
		if strings.Contains(b, p) {
			return true
		}
	}
	return false
}

// unmanagedCommonDotfiles lists well-known dotfiles present in $HOME but unmanaged.
func unmanagedCommonDotfiles() []string {
	common := []string{
		".zshrc", ".zprofile", ".bashrc", ".bash_profile", ".profile",
		".gitconfig", ".gitignore", ".vimrc", ".tmux.conf", ".inputrc",
		".config/starship.toml", ".config/nvim/init.lua", ".config/ghostty/config",
		".aliases", ".functions", ".curlrc", ".editorconfig",
	}
	managed := map[string]bool{}
	if m, err := chez.Managed(); err == nil {
		for _, f := range m {
			managed[f] = true
		}
	}
	home, _ := os.UserHomeDir()
	var out []string
	for _, rel := range common {
		if managed[rel] {
			continue
		}
		if _, err := os.Stat(filepath.Join(home, rel)); err == nil {
			out = append(out, rel)
		}
	}
	return out
}

// UntrackFile stops managing a file but keeps it on disk.
func UntrackFile(path string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if path != "" {
		if _, err := os.Stat(expand(path)); err == nil {
			path = expand(path)
		} else {
			sel, err := pickManaged("stop managing which file? (it stays on disk)", path)
			if err != nil || sel == "" {
				return err
			}
			path = homePath(sel)
		}
	} else {
		sel, err := pickManaged("stop managing which file? (it stays on disk)", "")
		if err != nil || sel == "" {
			return err
		}
		path = homePath(sel)
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
