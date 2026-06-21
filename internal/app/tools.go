package app

import (
	"fmt"
	"strings"

	"github.com/carrots-sh/casa/internal/brewfile"
	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/pm"
	"github.com/carrots-sh/casa/internal/ui"
)

// AddTool installs a package and records it in the Brewfile.
func AddTool(mgr, name string) error {
	var err error
	if mgr == "" {
		if mgr, err = ui.Select("which package manager?", pm.Managers); err != nil || mgr == "" {
			return err
		}
	}
	if name == "" {
		if name, err = ui.Input("package name (" + mgr + ")"); err != nil || name == "" {
			return err
		}
	}
	fmt.Printf("installing %s via %s...\n", name, mgr)
	if err := pm.Install(mgr, name); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	b := bf()
	if !b.Configured() {
		fmt.Printf("✓ installed %s (no brewfile configured, so not recorded)\n", name)
		return nil
	}
	if err := b.Add(mgr, name); err != nil {
		return err
	}
	_ = chez.ApplyNoScripts(brewfile.RenderedPath())
	fmt.Printf("✓ installed and recorded: %s %q\n", mgr, name)
	offerSave(fmt.Sprintf("casa: add %s %s", mgr, name))
	return nil
}

type tool struct{ mgr, name string }

// RemoveTools shows a flat list of everything recorded across all managers.
func RemoveTools() error {
	rendered := brewfile.RenderedPath()
	var labels []string
	byLabel := map[string]tool{}
	for _, mgr := range pm.Managers {
		names, _ := brewfile.Declared(rendered, mgr)
		for _, n := range names {
			l := fmt.Sprintf("%-6s %s", mgr, n)
			labels = append(labels, l)
			byLabel[l] = tool{mgr, n}
		}
	}
	if len(labels) == 0 {
		return fmt.Errorf("nothing recorded in the brewfile")
	}
	sel, err := ui.MultiSelect("remove which? (space to pick, enter to confirm)", labels)
	if err != nil || len(sel) == 0 {
		return err
	}
	b := bf()
	for _, l := range sel {
		t := byLabel[l]
		fmt.Printf("removing %s (%s)...\n", t.name, t.mgr)
		if err := pm.Uninstall(t.mgr, t.name); err != nil {
			fmt.Printf("  (uninstall errored: %v; removing from brewfile anyway)\n", err)
		}
		if b.Configured() {
			_ = b.Remove(t.mgr, t.name)
		}
	}
	if b.Configured() {
		_ = chez.ApplyNoScripts(rendered)
	}
	fmt.Println("✓ removed")
	offerSave("casa: remove tools")
	return nil
}

// UpdateTools lists outdated packages and upgrades the chosen ones.
func UpdateTools() error {
	items := pm.Outdated()
	items = append(items, "uv    (upgrade all uv tools)", "cargo (upgrade all)")
	sel, err := ui.MultiSelect("update which? (space to pick, enter to confirm)", items)
	if err != nil || len(sel) == 0 {
		return err
	}
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
	fmt.Println("✓ updates applied")
	return nil
}

// ListTools prints everything recorded, grouped by manager.
func ListTools() error {
	rendered := brewfile.RenderedPath()
	for _, mgr := range pm.Managers {
		names, _ := brewfile.Declared(rendered, mgr)
		for _, n := range names {
			fmt.Printf("%-6s %s\n", mgr, n)
		}
	}
	return nil
}
