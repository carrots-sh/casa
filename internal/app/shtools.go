// Tools that ship their own installer — recorded as [[packages.sh]] entries.
package app

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"strings"

	"github.com/carrots-sh/casa/internal/manifest"
	"github.com/carrots-sh/casa/internal/ui"
)

// binGuess pulls a likely binary name out of an installer URL (herdr.dev → herdr).
var binGuess = regexp.MustCompile(`https?://(?:www\.)?([a-z0-9-]+)\.`)

// addShTool records a tool that ships its own installer: run the one-liner
// once, then declare it in [[packages.sh]] so every machine gets it on apply.
func addShTool(bin string) error {
	install, err := ui.Input("install command (e.g. curl -fsSL https://herdr.dev/install.sh | sh)")
	if err != nil || install == "" {
		return err
	}
	if bin == "" {
		guess := ""
		if mm := binGuess.FindStringSubmatch(install); mm != nil {
			guess = mm[1]
		}
		prompt := "binary name (how casa detects it's installed)"
		if guess != "" {
			prompt += " [" + guess + "]"
		}
		if bin, err = ui.Input(prompt); err != nil {
			return err
		}
		if bin == "" {
			bin = guess
		}
		if bin == "" {
			return fmt.Errorf("a binary name is required")
		}
	}
	update, err := ui.Input("self-update command (leave empty if it updates itself)")
	if err != nil {
		return err
	}
	osChoice, err := ui.Select("runs on", []string{"all platforms", "darwin (macOS only)", "linux only"})
	if err != nil || osChoice == "" {
		return err
	}
	osTag := ""
	if f := strings.Fields(osChoice)[0]; f == "darwin" || f == "linux" {
		osTag = f
	}
	if _, err := exec.LookPath(bin); err == nil {
		fmt.Printf("%s is already installed — recording it without re-running the installer.\n", bin)
	} else {
		ok, err := ui.Confirm("run now:  " + install)
		if err != nil || !ok {
			return err
		}
		if err := runShell("sh", "-c", install); err != nil {
			return fmt.Errorf("installer failed: %w", err)
		}
		if _, err := exec.LookPath(bin); err != nil {
			fmt.Printf("note: %q isn't on PATH after the install — check the binary name (still recording).\n", bin)
		}
	}
	m, ok, err := ensurePkg()
	if err != nil || !ok {
		return err
	}
	if err := m.AddSh(manifest.ShTool{Bin: bin, Install: install, Update: update, OS: osTag}); err != nil {
		return err
	}
	fmt.Printf("✓ installed and recorded: sh %q\n", bin)
	offerSave("casa: add sh " + bin)
	return nil
}

// removeShTool drops the manifest block and offers to delete the binary the
// installer left behind (casa never deletes it silently — it didn't put it there).
func removeShTool(m manifest.Manifest, bin string) {
	_ = m.RemoveSh(bin)
	path, err := exec.LookPath(bin)
	if err != nil {
		return
	}
	ok, _ := ui.Confirm("also delete the binary at " + path + "?")
	if !ok {
		fmt.Println("  left in place: " + path)
		return
	}
	if err := os.Remove(path); err != nil {
		fmt.Printf("  (couldn't delete %s: %v)\n", path, err)
		return
	}
	fmt.Println("  deleted " + path + "  (any ~/." + bin + "-style data dirs are yours to clean)")
}
