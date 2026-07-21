// Package manifest reads and edits casa's package manifest —
// .chezmoidata/packages.toml in the chezmoi source dir. It is the single
// source of truth for what's installed on a machine: chezmoi loads it as
// template data, and the run_onchange scripts (see scripts.go) render it into
// a brew bundle run and an installers script on every apply.
//
// Reads decode the TOML properly; writes edit the file line by line so
// hand-written comments and ordering survive.
package manifest

import (
	"fmt"
	"os"
	"regexp"
	"slices"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

// DefaultRel is the manifest's default path relative to the source dir.
const DefaultRel = ".chezmoidata/packages.toml"

// Sections lists the string-list sections, in display order.
var Sections = []string{"taps", "brew", "brew_darwin", "cask", "go", "uv", "npm", "cargo"}

// SectionFor maps a pm manager name to its manifest section.
func SectionFor(mgr string) string {
	if mgr == "tap" {
		return "taps"
	}
	return mgr
}

// ManagerFor maps a section back to the pm manager that installs it.
func ManagerFor(section string) string {
	switch section {
	case "taps":
		return "tap"
	case "brew_darwin":
		return "brew"
	default:
		return section
	}
}

// ShTool is one self-installing tool ([[packages.sh]] entry).
type ShTool struct {
	Bin     string `toml:"bin"`     // binary name, for install detection
	Install string `toml:"install"` // the installer one-liner
	Update  string `toml:"update"`  // optional self-update command
	OS      string `toml:"os"`      // optional: "darwin" | "linux"; "" = all
}

// Manifest points at a packages.toml file.
type Manifest struct{ Path string }

// Configured reports whether the manifest file exists.
func (m Manifest) Configured() bool {
	_, err := os.Stat(m.Path)
	return err == nil
}

type doc struct {
	Packages struct {
		Taps       []string `toml:"taps"`
		Brew       []string `toml:"brew"`
		BrewDarwin []string `toml:"brew_darwin"`
		Cask       []string `toml:"cask"`
		Go         []string `toml:"go"`
		Uv         []string `toml:"uv"`
		Npm        []string `toml:"npm"`
		Cargo      []string `toml:"cargo"`
		Sh         []ShTool `toml:"sh"`
	} `toml:"packages"`
}

func (m Manifest) decode() (doc, error) {
	var d doc
	_, err := toml.DecodeFile(m.Path, &d)
	return d, err
}

// List returns the entries of a string-list section.
func (m Manifest) List(section string) ([]string, error) {
	d, err := m.decode()
	if err != nil {
		return nil, err
	}
	switch section {
	case "taps":
		return d.Packages.Taps, nil
	case "brew":
		return d.Packages.Brew, nil
	case "brew_darwin":
		return d.Packages.BrewDarwin, nil
	case "cask":
		return d.Packages.Cask, nil
	case "go":
		return d.Packages.Go, nil
	case "uv":
		return d.Packages.Uv, nil
	case "npm":
		return d.Packages.Npm, nil
	case "cargo":
		return d.Packages.Cargo, nil
	}
	return nil, fmt.Errorf("unknown manifest section %q", section)
}

// ShTools returns the declared self-installing tools.
func (m Manifest) ShTools() ([]ShTool, error) {
	d, err := m.decode()
	return d.Packages.Sh, err
}

func arrayOpen(section string) *regexp.Regexp {
	return regexp.MustCompile(`^\s*` + regexp.QuoteMeta(section) + `\s*=\s*\[`)
}

// Add inserts name into section's array (idempotent). Comments are preserved:
// only a single line is inserted (or one inline array rewritten).
func (m Manifest) Add(section, name string) error {
	existing, err := m.List(section)
	if err != nil {
		return err
	}
	if slices.Contains(existing, name) {
		return nil
	}
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	open := arrayOpen(section)
	entry := "  " + strconv.Quote(name) + ","
	for i, l := range lines {
		if !open.MatchString(l) {
			continue
		}
		if strings.Contains(l, "]") { // inline array → rewrite as multiline
			block := append([]string{section + " = ["}, quoteAll(append(existing, name))...)
			lines = splice(lines, i, 1, append(block, "]")...)
			return m.write(lines)
		}
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "]" {
				lines = splice(lines, j, 0, entry)
				return m.write(lines)
			}
		}
		return fmt.Errorf("unclosed array %q in %s", section, m.Path)
	}
	// section missing: create it right after the [packages] header.
	for i, l := range lines {
		if strings.TrimSpace(l) == "[packages]" {
			lines = splice(lines, i+1, 0, "", section+" = [", entry, "]")
			return m.write(lines)
		}
	}
	return fmt.Errorf("no [packages] table in %s", m.Path)
}

// Remove drops name from a section (no-op if absent).
func (m Manifest) Remove(section, name string) error {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	open := arrayOpen(section)
	entryRe := regexp.MustCompile(`^\s*"` + regexp.QuoteMeta(name) + `"\s*,?\s*(#.*)?$`)
	for i, l := range lines {
		if !open.MatchString(l) {
			continue
		}
		if strings.Contains(l, "]") { // inline array → rewrite without the entry
			existing, err := m.List(section)
			if err != nil {
				return err
			}
			var kept []string
			for _, e := range existing {
				if e != name {
					kept = append(kept, strconv.Quote(e))
				}
			}
			lines[i] = section + " = [" + strings.Join(kept, ", ") + "]"
			return m.write(lines)
		}
		for j := i + 1; j < len(lines); j++ {
			if strings.TrimSpace(lines[j]) == "]" {
				return nil // not found; idempotent
			}
			if entryRe.MatchString(lines[j]) {
				lines = splice(lines, j, 1)
				return m.write(lines)
			}
		}
	}
	return nil
}

// AddSh appends a [[packages.sh]] block (no-op if bin is already recorded).
func (m Manifest) AddSh(t ShTool) error {
	tools, err := m.ShTools()
	if err != nil {
		return err
	}
	for _, e := range tools {
		if e.Bin == t.Bin {
			return nil
		}
	}
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}
	block := []string{"", "[[packages.sh]]",
		"bin = " + strconv.Quote(t.Bin),
		"install = " + strconv.Quote(t.Install),
	}
	if t.Update != "" {
		block = append(block, "update = "+strconv.Quote(t.Update))
	}
	if t.OS != "" {
		block = append(block, "os = "+strconv.Quote(t.OS))
	}
	s := strings.TrimRight(string(data), "\n") + "\n" + strings.Join(block, "\n") + "\n"
	return os.WriteFile(m.Path, []byte(s), 0o644)
}

// RemoveSh drops the [[packages.sh]] block for bin (no-op if absent).
func (m Manifest) RemoveSh(bin string) error {
	data, err := os.ReadFile(m.Path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	binRe := regexp.MustCompile(`^\s*bin\s*=\s*"` + regexp.QuoteMeta(bin) + `"\s*(#.*)?$`)
	for i, l := range lines {
		if strings.TrimSpace(l) != "[[packages.sh]]" {
			continue
		}
		end := len(lines) // block runs to the next table header or EOF
		match := false
		for j := i + 1; j < len(lines); j++ {
			if strings.HasPrefix(strings.TrimSpace(lines[j]), "[") {
				end = j
				break
			}
			if binRe.MatchString(lines[j]) {
				match = true
			}
		}
		if !match {
			continue
		}
		start := i
		if start > 0 && strings.TrimSpace(lines[start-1]) == "" {
			start-- // eat the blank line the block was appended with
		}
		lines = splice(lines, start, end-start)
		return m.write(lines)
	}
	return nil
}

func (m Manifest) write(lines []string) error {
	return os.WriteFile(m.Path, []byte(strings.Join(lines, "\n")), 0o644)
}

func quoteAll(names []string) []string {
	out := make([]string, len(names))
	for i, n := range names {
		out[i] = "  " + strconv.Quote(n) + ","
	}
	return out
}

// splice removes del lines at i and inserts ins there.
func splice(lines []string, i, del int, ins ...string) []string {
	out := make([]string, 0, len(lines)-del+len(ins))
	out = append(out, lines[:i]...)
	out = append(out, ins...)
	return append(out, lines[i+del:]...)
}
