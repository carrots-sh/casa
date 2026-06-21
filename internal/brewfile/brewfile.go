// Package brewfile keeps the chezmoi-managed Brewfile in sync with casa actions.
package brewfile

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

// sourceTmpl returns the path to the Brewfile template in the chezmoi source dir.
func sourceTmpl() (string, error) {
	out, err := exec.Command("chezmoi", "source-path").Output()
	if err != nil {
		return "", fmt.Errorf("chezmoi source-path: %w", err)
	}
	return filepath.Join(strings.TrimSpace(string(out)), "dot_Brewfile.tmpl"), nil
}

func rendered() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".Brewfile"), nil
}

func anchor(mgr string) string {
	switch mgr {
	case "tap", "cask", "go", "uv", "npm", "cargo":
		return "# casa:" + mgr
	default:
		return "# casa:brew"
	}
}

// Declared returns the package names recorded for mgr in the rendered ~/.Brewfile.
func Declared(mgr string) ([]string, error) {
	path, err := rendered()
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
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

// Add inserts `mgr "name"` just before the manager's anchor in the source
// template. Idempotent: a no-op if the exact line already exists.
func Add(mgr, name string) error {
	t, err := sourceTmpl()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(t)
	if err != nil {
		return err
	}
	line := fmt.Sprintf("%s %q", mgr, name)
	anc := anchor(mgr)
	var out []string
	inserted := false
	for _, l := range strings.Split(string(data), "\n") {
		if l == line {
			return nil // already present
		}
		if l == anc {
			out = append(out, line)
			inserted = true
		}
		out = append(out, l)
	}
	if !inserted {
		return fmt.Errorf("anchor %q not found in %s", anc, t)
	}
	return os.WriteFile(t, []byte(strings.Join(out, "\n")), 0o644)
}

// Remove deletes any line beginning with `mgr "name"` from the source template.
func Remove(mgr, name string) error {
	t, err := sourceTmpl()
	if err != nil {
		return err
	}
	data, err := os.ReadFile(t)
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
	return os.WriteFile(t, []byte(strings.Join(out, "\n")), 0o644)
}

// Refresh re-renders ~/.Brewfile from the (now-edited) source, without running scripts.
func Refresh() error {
	path, err := rendered()
	if err != nil {
		return err
	}
	return exec.Command("chezmoi", "apply", "--exclude=scripts", path).Run()
}
