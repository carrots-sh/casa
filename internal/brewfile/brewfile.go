// Package brewfile keeps a chezmoi-managed Brewfile template in sync with casa
// package actions, inserting/removing entries at "# <anchor>:<manager>" markers.
package brewfile

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Brewfile points at a source template and its anchor convention.
type Brewfile struct {
	Tmpl   string // absolute path to the source template (e.g. .../dot_Brewfile.tmpl)
	Anchor string // anchor prefix word, e.g. "casa" → "# casa:brew"
}

func (b Brewfile) anchor(mgr string) string {
	switch mgr {
	case "tap", "cask", "go", "uv", "npm", "cargo":
		return "# " + b.Anchor + ":" + mgr
	default:
		return "# " + b.Anchor + ":brew"
	}
}

// Configured reports whether a Brewfile template is set and present.
func (b Brewfile) Configured() bool {
	if b.Tmpl == "" {
		return false
	}
	_, err := os.Stat(b.Tmpl)
	return err == nil
}

// Add inserts `mgr "name"` before the manager's anchor (idempotent).
func (b Brewfile) Add(mgr, name string) error {
	data, err := os.ReadFile(b.Tmpl)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("%s %q", mgr, name)
	anc := b.anchor(mgr)
	var out []string
	inserted := false
	for _, l := range strings.Split(string(data), "\n") {
		if l == line {
			return nil
		}
		if l == anc {
			out = append(out, line)
			inserted = true
		}
		out = append(out, l)
	}
	if !inserted {
		return fmt.Errorf("anchor %q not found in %s", anc, b.Tmpl)
	}
	return os.WriteFile(b.Tmpl, []byte(strings.Join(out, "\n")), 0o644)
}

// Remove deletes any line beginning with `mgr "name"`.
func (b Brewfile) Remove(mgr, name string) error {
	data, err := os.ReadFile(b.Tmpl)
	if err != nil {
		return err
	}
	prefix := fmt.Sprintf("%s %q", mgr, name)
	var out []string
	for _, l := range strings.Split(string(data), "\n") {
		if strings.HasPrefix(l, prefix) {
			continue
		}
		out = append(out, l)
	}
	return os.WriteFile(b.Tmpl, []byte(strings.Join(out, "\n")), 0o644)
}

// Declared returns package names recorded for mgr in the rendered ~/.Brewfile.
func Declared(renderedPath, mgr string) ([]string, error) {
	f, err := os.Open(renderedPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	re := regexp.MustCompile(`^` + regexp.QuoteMeta(mgr) + ` "([^"]+)"`)
	var names []string
	s := bufio.NewScanner(f)
	for s.Scan() {
		if m := re.FindStringSubmatch(s.Text()); m != nil {
			names = append(names, m[1])
		}
	}
	return names, s.Err()
}

// RenderedPath returns ~/.Brewfile (the applied target brew bundle reads).
func RenderedPath() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".Brewfile")
}
