package cmd

import (
	"fmt"
	"strings"

	"github.com/carrots-sh/casa/internal/pm"
	"github.com/carrots-sh/casa/internal/ui"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Aliases: []string{"up"},
	Short:   "Update outdated packages — pick one, many, or all",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runUpdate()
	},
}

func runUpdate() error {
	items := pm.Outdated() // "brew foo", "cask bar", "npm baz"
	// uv/go/cargo don't expose per-package outdated cleanly → offer manager-wide upgrades
	items = append(items,
		"uv    (upgrade all uv tools)",
		"cargo (upgrade all)",
	)

	sel, err := ui.MultiSelect("Update which? — pick any", items)
	if err != nil {
		return err
	}
	if len(sel) == 0 {
		return nil
	}

	for _, line := range sel {
		fields := strings.Fields(line)
		if len(fields) == 0 {
			continue
		}
		mgr := fields[0]
		switch mgr {
		case "uv", "cargo":
			fmt.Printf("Upgrading all %s packages...\n", mgr)
			if err := pm.UpgradeAll(mgr); err != nil {
				fmt.Printf("  %v\n", err)
			}
		default:
			if len(fields) < 2 {
				continue
			}
			name := fields[1]
			fmt.Printf("Upgrading %s (%s)...\n", name, mgr)
			if err := pm.Upgrade(mgr, name); err != nil {
				fmt.Printf("  %v\n", err)
			}
		}
	}
	fmt.Println("✓ updates applied")
	return nil
}
