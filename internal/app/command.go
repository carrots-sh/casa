// Paste-an-install-command support: casa detects the manager (go, cargo,
// brew, npm, uv, curl|sh, …) from a pasted command, installs if needed, and
// records the result in the manifest.
package app

import (
	"fmt"
	"os/exec"
	"slices"
	"strings"

	"github.com/carrots-sh/casa/internal/manifest"
	"github.com/carrots-sh/casa/internal/pm"
	"github.com/carrots-sh/casa/internal/ui"
)

// cmdRow is the "paste a command" row offered in the add flows.
const cmdRow = "command · paste an install command (go install …, curl … | sh, …)"

// addCommand asks for a full install command, detects what it is, and takes
// it from there. Already-installed tools are recorded without reinstalling.
func addCommand(cmd string) error {
	var err error
	if cmd == "" {
		if cmd, err = ui.Input("paste the install command"); err != nil || cmd == "" {
			return err
		}
	}
	mgr, pkg := parseInstallCommand(cmd)
	switch {
	case mgr == "sh":
		return addShTool("", cmd)
	case mgr == "" || pkg == "":
		return fmt.Errorf("couldn't detect a package manager in %q — supported: go, cargo, npm -g, uv tool, brew, curl|sh", cmd)
	}
	pmgr, _ := pm.ByName(mgr)
	if slices.Contains(pmgr.Installed(), pkg) {
		fmt.Printf("%s is already installed — recording it.\n", pkg)
	} else {
		fmt.Printf("installing %s via %s...\n", pkg, mgr)
		if err := pmgr.Install(pkg); err != nil {
			return fmt.Errorf("install failed: %w", err)
		}
	}
	return recordTool(mgr, pkg)
}

// parseInstallCommand detects the manager and package in an install one-liner.
// mgr "sh" means a curl/wget installer (the whole command is the payload).
func parseInstallCommand(s string) (mgr, pkg string) {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "curl ") || strings.HasPrefix(s, "wget ") ||
		strings.Contains(s, "| sh") || strings.Contains(s, "| bash") {
		return "sh", ""
	}
	f := strings.Fields(s)
	if len(f) >= 2 && f[0] == "sudo" {
		f = f[1:]
	}
	if len(f) < 2 {
		return "", ""
	}
	arg := func(from int) string { // first non-flag argument
		for _, a := range f[from:] {
			if !strings.HasPrefix(a, "-") {
				return a
			}
		}
		return ""
	}
	switch f[0] {
	case "go":
		if f[1] == "install" || f[1] == "get" {
			p := arg(2)
			p, _, _ = strings.Cut(p, "@") // go install path@latest → path
			return "go", p
		}
	case "cargo":
		if f[1] == "install" {
			return "cargo", arg(2)
		}
	case "npm", "pnpm", "yarn":
		if (f[1] == "install" || f[1] == "i" || f[1] == "add" || f[1] == "global") &&
			(slices.Contains(f, "-g") || slices.Contains(f, "--global") || f[1] == "global") {
			return "npm", arg(2)
		}
	case "bun":
		if (f[1] == "add" || f[1] == "install" || f[1] == "i") &&
			(slices.Contains(f, "-g") || slices.Contains(f, "--global")) {
			return "bun", arg(2)
		}
	case "uv":
		if len(f) >= 3 && f[1] == "tool" && f[2] == "install" {
			return "uv", arg(3)
		}
	case "brew":
		switch f[1] {
		case "install":
			if slices.Contains(f, "--cask") {
				return "cask", arg(2)
			}
			return "brew", arg(2)
		case "tap":
			return "tap", arg(2)
		}
	}
	return "", ""
}

// recordTool writes an installed package into the manifest (bootstrapping it
// on first use) and offers to save.
func recordTool(mgr, name string) error {
	m, ok, err := ensurePkg()
	if err != nil {
		return err
	}
	if !ok {
		fmt.Printf("✓ installed %s (no manifest, so not recorded)\n", name)
		return nil
	}
	section := manifest.SectionFor(mgr)
	if err := m.Add(section, name); err != nil {
		return err
	}
	fmt.Printf("✓ recorded: %s %q\n", section, name)
	offerSave(fmt.Sprintf("casa: add %s %s", mgr, name))
	return nil
}

// unrecordedPairs lists installed packages the manifest doesn't know about,
// as {manager, name} — the drift casa offers to record. Best-effort: managers
// that aren't installed contribute nothing.
func unrecordedPairs(m manifest.Manifest) []pm.Result {
	if !m.Configured() {
		return nil
	}
	var out []pm.Result
	for _, mgr := range pm.Managers {
		if _, err := exec.LookPath(mgrBinary(mgr.Name())); err != nil {
			continue
		}
		section := manifest.SectionFor(mgr.Name())
		have := map[string]bool{}
		for _, s := range recordedSections(m, section) {
			have[s] = true
		}
		for _, name := range mgr.Installed() {
			if !have[name] {
				out = append(out, pm.Result{Mgr: mgr.Name(), Name: name})
			}
		}
	}
	return out
}

// recordedSections returns everything recorded that installs via the same
// manager as section (brew covers brew_darwin; tap covers taps_trusted).
func recordedSections(m manifest.Manifest, section string) []string {
	var out []string
	for _, s := range manifest.Sections {
		if manifest.ManagerFor(s) == manifest.ManagerFor(section) {
			names, _ := m.List(s)
			out = append(out, names...)
		}
	}
	return out
}

// mgrBinary maps a manager name to the CLI it needs on PATH.
func mgrBinary(name string) string {
	switch name {
	case "cask", "tap":
		return "brew"
	default:
		return name
	}
}
