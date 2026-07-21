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

// tilde renders a path for display as ~/…: home-relative paths get the prefix,
// absolute paths under $HOME get shortened, anything else is left alone.
func tilde(p string) string {
	h, _ := os.UserHomeDir()
	if h != "" && strings.HasPrefix(p, h+string(os.PathSeparator)) {
		return "~" + p[len(h):]
	}
	if p == "" || filepath.IsAbs(p) || strings.HasPrefix(p, "~") {
		return p
	}
	return "~/" + p
}

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
		labels[i] = tilde(m) + badges[m]
		byLabel[labels[i]] = m
	}
	sel, err := ui.Select(title, labels)
	return byLabel[sel], err
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
	fmt.Printf("✓ edited %s\n", tilde(sel))
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
		labels := make([]string, len(cands))
		for i, rel := range cands {
			labels[i] = tilde(rel)
		}
		sel, err := ui.MultiSelect("which files should casa manage?", labels)
		if err != nil || len(sel) == 0 {
			return err
		}
		for _, l := range sel {
			rel := strings.TrimPrefix(l, "~/")
			if err := trackOne(homePath(rel)); err != nil {
				fmt.Printf("  (skipped %s: %v)\n", l, err)
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
	fmt.Printf("✓ no longer managing %s (file kept)\n", tilde(path))
	offerSave("casa: untrack")
	return nil
}

// ListConfigs prints all managed files with their storage badges.
func ListConfigs() error {
	managed, err := chez.Managed()
	if err != nil {
		return err
	}
	badges := storageBadges(managed)
	for _, m := range managed {
		fmt.Println(tilde(m) + badges[m])
	}
	return nil
}

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
