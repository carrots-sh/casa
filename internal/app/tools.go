package app

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/carrots-sh/casa/internal/manifest"
	"github.com/carrots-sh/casa/internal/pm"
	"github.com/carrots-sh/casa/internal/ui"
)

// shRow is the synthetic search-result row for tools with their own installer.
const shRow = "sh     (a tool with its own installer — curl | sh)"

// AddTool installs a package and records it in the manifest. With no name it
// searches across all package managers and lets you pick where to install
// from; "sh" and "cmd" route to the installer/pasted-command flows.
func AddTool(mgr, name string) error {
	var err error
	switch mgr {
	case "sh":
		return addShTool(name, "")
	case "cmd", "command":
		return addCommand(name)
	}
	if name == "" && mgr == "" {
		if mgr, name, err = searchPick(); err != nil || mgr == "" {
			return err
		}
		switch mgr {
		case "sh":
			return addShTool("", "")
		case "cmd":
			return addCommand("")
		}
	}
	if mgr == "" {
		opts := append(pm.Names(), "sh", "command")
		if mgr, err = ui.Select("which package manager?", opts); err != nil || mgr == "" {
			return err
		}
		switch mgr {
		case "sh":
			return addShTool(name, "")
		case "command":
			return addCommand("")
		}
	}
	pmgr, ok := pm.ByName(mgr)
	if !ok {
		return fmt.Errorf("unknown manager %q", mgr)
	}
	// managers without a search (go, uv, tap) take the package directly
	if name == "" {
		if name, err = ui.Input("package for " + mgr + " (e.g. golang.org/x/tools/gopls)"); err != nil || name == "" {
			return err
		}
	}
	fmt.Printf("installing %s via %s...\n", name, mgr)
	if err := pmgr.Install(name); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	return recordTool(mgr, name)
}

// searchPick prompts for a query, searches every manager in parallel, and
// returns the (manager, name) the user picks. A pasted install command is
// detected and routed straight to the command flow; sh and command rows ride
// along as synthetic results so every flow is reachable from plain "add".
func searchPick() (string, string, error) {
	query, err := ui.Input("search (" + strings.Join(pm.Searchable(), ", ") + ") — or paste an install command")
	if err != nil || query == "" {
		return "", "", err
	}
	if mgr, _ := parseInstallCommand(query); mgr != "" {
		if err := addCommand(query); err != nil {
			return "", "", err
		}
		return "", "", nil // handled
	}
	results := pm.SearchAll(query)
	labels := make([]string, len(results))
	byLabel := map[string]pm.Result{}
	for i, r := range results {
		l := fmt.Sprintf("%-6s %s", r.Mgr, r.Name)
		labels[i] = l
		byLabel[l] = r
	}
	labels = append(labels, shRow, cmdRow)
	sel, err := ui.Select("install which?", labels)
	if err != nil || sel == "" {
		return "", "", err
	}
	switch sel {
	case shRow:
		return "sh", "", nil
	case cmdRow:
		return "cmd", "", nil
	}
	r := byLabel[sel]
	return r.Mgr, r.Name, nil
}

type tool struct{ section, name string }

// toolRows builds the flat "everything recorded" list: pm sections first,
// then sh tools. Labels are keyed back to (section, name).
func toolRows(m manifest.Manifest) ([]string, map[string]tool) {
	var labels []string
	byLabel := map[string]tool{}
	add := func(section, name string) {
		l := fmt.Sprintf("%-11s %s", section, name)
		labels = append(labels, l)
		byLabel[l] = tool{section, name}
	}
	for _, section := range manifest.Sections {
		names, _ := m.List(section)
		for _, n := range names {
			add(section, n)
		}
	}
	tools, _ := m.ShTools()
	for _, t := range tools {
		add("sh", t.Bin)
	}
	return labels, byLabel
}

// RemoveTools shows a flat list of everything recorded across all managers.
func RemoveTools() error {
	m := mf()
	if !m.Configured() {
		fmt.Println("nothing recorded yet — try: casa tools add")
		return nil
	}
	labels, byLabel := toolRows(m)
	if len(labels) == 0 {
		fmt.Println("nothing recorded yet — try: casa tools add")
		return nil
	}
	sel, err := ui.MultiSelect("remove which? (space to pick, enter to confirm)", labels)
	if err != nil || len(sel) == 0 {
		return err
	}
	for _, l := range sel {
		t := byLabel[l]
		fmt.Printf("removing %s (%s)...\n", t.name, t.section)
		if t.section == "sh" {
			removeShTool(m, t.name)
			continue
		}
		if m, ok := pm.ByName(manifest.ManagerFor(t.section)); ok {
			if err := m.Uninstall(t.name); err != nil {
				fmt.Printf("  (uninstall errored: %v; removing from manifest anyway)\n", err)
			}
		}
		_ = m.Remove(t.section, t.name)
	}
	fmt.Println("✓ removed")
	offerSave("casa: remove tools")
	return nil
}

// UpdateTools lists outdated packages and upgrades the chosen ones. sh tools
// appear only when they declared a self-update command.
func UpdateTools() error {
	items, shUpdates := updatableItems()
	if len(items) == 0 {
		fmt.Println("✓ nothing outdated (brew, cask, npm)")
		return nil
	}
	// ponytail: uv/cargo blanket upgrades only reachable when something has updates.
	items = append(items, "uv    (upgrade all uv tools)", "cargo (upgrade all)")
	sel, err := ui.MultiSelect("update which? (space to pick, enter to confirm)", items)
	if err != nil || len(sel) == 0 {
		return err
	}
	allSel := len(sel) == len(items) // everything picked → one brew upgrade, not per-package
	brewDone := false
	for _, line := range sel {
		f := strings.Fields(line)
		if len(f) == 0 {
			continue
		}
		if allSel && (f[0] == "brew" || f[0] == "cask") {
			if !brewDone {
				brewDone = true
				fmt.Println("upgrading all brew packages...")
				report(pm.UpgradeAllBrew())
			}
			continue
		}
		applyUpdate(f, shUpdates)
	}
	invalidateStatus()
	fmt.Println("✓ updates applied")
	return nil
}

// updatableItems builds the update picker rows: per-package outdated entries
// plus sh tools that declared a self-update command (mapped bin → command).
func updatableItems() ([]string, map[string]string) {
	items := pm.Outdated()
	shTools, _ := mf().ShTools()
	shUpdates := map[string]string{}
	for _, t := range shTools {
		if t.Update != "" {
			shUpdates[t.Bin] = t.Update
			items = append(items, "sh     "+t.Bin)
		}
	}
	return items, shUpdates
}

// applyUpdate runs one picked update row ("mgr name" or a bulk-upgrade row).
func applyUpdate(f []string, shUpdates map[string]string) {
	m, _ := pm.ByName(f[0])
	switch {
	case f[0] == "sh" && len(f) >= 2:
		fmt.Printf("updating %s...\n", f[1])
		report(runShell("sh", "-c", shUpdates[f[1]]))
	case m == nil:
		return
	default:
		if b, ok := m.(pm.BulkUpgrader); ok {
			fmt.Printf("upgrading all %s packages...\n", f[0])
			report(b.UpgradeAll())
			return
		}
		if o, ok := m.(pm.Outdater); ok && len(f) >= 2 {
			fmt.Printf("upgrading %s (%s)...\n", f[1], f[0])
			report(o.Upgrade(f[1]))
		}
	}
}

// toolLines renders everything recorded, grouped by section.
func toolLines() ([]string, error) {
	m := mf()
	if !m.Configured() {
		return nil, fmt.Errorf("no manifest yet — try: casa tools add   (or: casa tools import)")
	}
	var lines []string
	for _, section := range manifest.Sections {
		names, _ := m.List(section)
		for _, n := range names {
			lines = append(lines, fmt.Sprintf("%-11s %s", section, n))
		}
	}
	tools, _ := m.ShTools()
	for _, t := range tools {
		marker := ""
		if _, err := exec.LookPath(t.Bin); err != nil {
			marker = "   (not installed here)"
		}
		lines = append(lines, fmt.Sprintf("%-11s %s%s", "sh", t.Bin, marker))
	}
	return lines, nil
}

// ListTools prints everything recorded (plain output — pipeable).
func ListTools() error {
	lines, err := toolLines()
	if err != nil {
		fmt.Println(err)
		return nil
	}
	for _, l := range lines {
		fmt.Println(l)
	}
	return nil
}
