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
// searches across all package managers and lets you pick where to install from.
func AddTool(mgr, name string) error {
	var err error
	if mgr == "sh" {
		return addShTool(name)
	}
	if name == "" {
		if mgr, name, err = searchPick(mgr); err != nil || name == "" {
			return err
		}
		if mgr == "sh" {
			return addShTool("")
		}
	} else if mgr == "" {
		opts := append(append([]string{}, pm.Managers...), "sh")
		if mgr, err = ui.Select("which package manager?", opts); err != nil || mgr == "" {
			return err
		}
		if mgr == "sh" {
			return addShTool(name)
		}
	}
	fmt.Printf("installing %s via %s...\n", name, mgr)
	if err := pm.Install(mgr, name); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
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
	fmt.Printf("✓ installed and recorded: %s %q\n", section, name)
	offerSave(fmt.Sprintf("casa: add %s %s", mgr, name))
	return nil
}

// searchPick prompts for a query, searches every manager in parallel, and
// returns the (manager, name) the user picks. If mgr is given, it scopes the
// search to that one manager. The sh option rides along as a synthetic row so
// self-installing tools are reachable from the menu's plain "add".
func searchPick(mgr string) (string, string, error) {
	query, err := ui.Input("search packages (" + strings.Join(pm.Searchable, ", ") + ")")
	if err != nil || query == "" {
		return "", "", err
	}
	var results []pm.Result
	if mgr != "" {
		for _, n := range pm.Search(mgr, query) {
			results = append(results, pm.Result{Mgr: mgr, Name: n})
		}
	} else {
		results = pm.SearchAll(query)
	}
	labels := make([]string, len(results))
	byLabel := map[string]pm.Result{}
	for i, r := range results {
		l := fmt.Sprintf("%-6s %s", r.Mgr, r.Name)
		labels[i] = l
		byLabel[l] = r
	}
	if mgr == "" {
		labels = append(labels, shRow)
	}
	if len(labels) == 0 {
		return "", "", fmt.Errorf("no packages found for %q", query)
	}
	sel, err := ui.Select("install which?", labels)
	if err != nil || sel == "" {
		return "", "", err
	}
	if sel == shRow {
		return "sh", "sh", nil // name is a placeholder; addShTool prompts for everything
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
		if err := pm.Uninstall(manifest.ManagerFor(t.section), t.name); err != nil {
			fmt.Printf("  (uninstall errored: %v; removing from manifest anyway)\n", err)
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
	items := pm.Outdated()
	shTools, _ := mf().ShTools()
	updatable := map[string]string{}
	for _, t := range shTools {
		if t.Update != "" {
			updatable[t.Bin] = t.Update
			items = append(items, "sh     "+t.Bin)
		}
	}
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
		switch f[0] {
		case "uv", "cargo":
			fmt.Printf("upgrading all %s packages...\n", f[0])
			if err := pm.UpgradeAll(f[0]); err != nil {
				fmt.Printf("  %v\n", err)
			}
		case "sh":
			if len(f) < 2 {
				continue
			}
			fmt.Printf("updating %s...\n", f[1])
			if err := runShell("sh", "-c", updatable[f[1]]); err != nil {
				fmt.Printf("  %v\n", err)
			}
		case "brew", "cask":
			if allSel {
				if !brewDone {
					brewDone = true
					fmt.Println("upgrading all brew packages...")
					if err := pm.UpgradeAllBrew(); err != nil {
						fmt.Printf("  %v\n", err)
					}
				}
				continue
			}
			fallthrough
		default:
			if len(f) < 2 {
				continue
			}
			fmt.Printf("upgrading %s (%s)...\n", f[1], f[0])
			if err := pm.Upgrade(f[0], f[1]); err != nil {
				fmt.Printf("  %v\n", err)
			}
		}
	}
	invalidateStatus()
	fmt.Println("✓ updates applied")
	return nil
}

// ListTools prints everything recorded, grouped by section.
func ListTools() error {
	m := mf()
	if !m.Configured() {
		fmt.Println("no manifest yet — try: casa tools add   (or: casa tools import)")
		return nil
	}
	for _, section := range manifest.Sections {
		names, _ := m.List(section)
		for _, n := range names {
			fmt.Printf("%-11s %s\n", section, n)
		}
	}
	tools, _ := m.ShTools()
	for _, t := range tools {
		marker := ""
		if _, err := exec.LookPath(t.Bin); err != nil {
			marker = "   (not installed here)"
		}
		fmt.Printf("%-11s %s%s\n", "sh", t.Bin, marker)
	}
	return nil
}
