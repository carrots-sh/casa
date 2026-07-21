// Tracking files: the adopt picker, storage choice, and its heuristics.
package app

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

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

	// suggested unmanaged dotfiles, plus an always-present "type a path" row —
	// a single pick, so enter acts on the highlighted row.
	const other = "another file · type a path"
	labels := []string{}
	for _, rel := range unmanagedCommonDotfiles() {
		labels = append(labels, tilde(rel))
	}
	labels = append(labels, other)
	sel, err := ui.Select("track which file?", labels)
	if err != nil || sel == "" {
		return err
	}
	p := expand(sel)
	if sel == other {
		if p, err = ui.PathInput("path of the file to start managing"); err != nil || p == "" {
			return err
		}
		p = expand(p)
	}
	if err := trackOne(p); err != nil {
		return err
	}
	offerSave("casa: track " + filepath.Base(p))
	return nil
}

// storage options offered when tracking a file. Order matters: plain first.
const (
	storePlain    = "plain              · same on every machine"
	storeTemplate = "template           · differs per machine (auto-fills your data)"
	storeSecret   = "encrypted          · secret, sealed in the repo"
	storeBoth     = "encrypted template · secret and per-machine"
)

// trackOne manages a single file, asking how it should be stored. The default
// follows two heuristics: sensitive-looking names suggest encryption, content
// containing this machine's data values (email, hostname, …) suggests a template.
func trackOne(abs string) error {
	def := storePlain
	if looksSensitive(abs) {
		def = storeSecret
	} else if hasDataValues(abs) {
		def = storeTemplate
	}
	choice, err := ui.SelectDefault("how should casa store "+filepath.Base(abs)+"?",
		[]string{storePlain, storeTemplate, storeSecret, storeBoth}, def)
	if err != nil || choice == "" {
		return err
	}
	switch choice {
	case storeTemplate:
		err = chez.AddTemplate(abs)
	case storeSecret:
		err = chez.AddEncrypt(abs)
	case storeBoth:
		err = chez.AddEncryptedTemplate(abs)
	default:
		err = chez.Add(abs)
	}
	if err != nil {
		return err
	}
	fmt.Printf("✓ now managing %s (%s)\n", tilde(abs), strings.TrimSpace(strings.SplitN(choice, "·", 2)[0]))
	return nil
}

var dataStringsCached []string

// dataStrings collects this machine's string data values (once per run) —
// the things a template would substitute.
func dataStrings() []string {
	if dataStringsCached != nil {
		return dataStringsCached
	}
	dataStringsCached = []string{} // non-nil: compute once even on error
	data, err := chez.Data()
	if err != nil {
		return dataStringsCached
	}
	for k, v := range data {
		if k == "chezmoi" {
			continue
		}
		if s, ok := v.(string); ok && len(s) >= 4 {
			dataStringsCached = append(dataStringsCached, s)
		}
	}
	if cm, ok := data["chezmoi"].(map[string]any); ok {
		for _, k := range []string{"hostname", "username"} {
			if s, ok := cm[k].(string); ok && len(s) >= 4 {
				dataStringsCached = append(dataStringsCached, s)
			}
		}
	}
	return dataStringsCached
}

// hasDataValues reports whether the file's content mentions any of this
// machine's data values — a good template candidate.
func hasDataValues(abs string) bool {
	b, err := os.ReadFile(abs)
	if err != nil {
		return false
	}
	s := string(b)
	for _, v := range dataStrings() {
		if strings.Contains(s, v) {
			return true
		}
	}
	return false
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
