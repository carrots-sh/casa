package app

import (
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"slices"
	"sort"
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

// TrustTaps picks which taps brew bundle may manage without prompting —
// trusted taps render as `tap "…", trusted: true`, so their formulae stop
// producing "tap formula is not trusted" warnings on install/cleanup.
func TrustTaps() error {
	m := mf()
	if !m.Configured() {
		fmt.Println("no manifest yet — try: casa tools add")
		return nil
	}
	plain, _ := m.List("taps")
	trusted, _ := m.List("taps_trusted")
	all := append(append([]string{}, plain...), trusted...)
	if len(all) == 0 {
		fmt.Println("no taps recorded yet")
		return nil
	}
	sort.Strings(all)
	want, err := ui.MultiSelect("which taps are trusted? (their formulae update without prompting)", all, trusted...)
	if err != nil {
		return err
	}
	wantSet := map[string]bool{}
	for _, t := range want {
		wantSet[t] = true
	}
	changed := 0
	for _, t := range all {
		was := slices.Contains(trusted, t)
		switch {
		case wantSet[t] && !was:
			_ = m.Remove("taps", t)
			if err := m.Add("taps_trusted", t); err != nil {
				return err
			}
			changed++
		case !wantSet[t] && was:
			_ = m.Remove("taps_trusted", t)
			if err := m.Add("taps", t); err != nil {
				return err
			}
			changed++
		}
	}
	if changed == 0 {
		fmt.Println("nothing to change.")
		return nil
	}
	fmt.Printf("✓ trusted taps: %s\n", strings.Join(want, ", "))
	offerSave("casa: update trusted taps")
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
