package cmd

import (
	"fmt"

	"github.com/carrots-sh/casa/internal/brewfile"
	"github.com/carrots-sh/casa/internal/pm"
	"github.com/carrots-sh/casa/internal/ui"
	"github.com/spf13/cobra"
)

var removeCmd = &cobra.Command{
	Use:     "remove [manager] [name...]",
	Aliases: []string{"rm"},
	Short:   "Uninstall package(s) and remove them from the Brewfile",
	RunE: func(cmd *cobra.Command, args []string) error {
		return runRemove(args)
	},
}

type pkg struct{ mgr, name string }

func runRemove(args []string) error {
	var picks []pkg

	if len(args) >= 2 {
		// explicit: casa remove <manager> <name...>
		for _, n := range args[1:] {
			picks = append(picks, pkg{args[0], n})
		}
	} else {
		// interactive: one flat list of every recorded package, tagged with its manager
		labels := []string{}
		byLabel := map[string]pkg{}
		for _, mgr := range pm.Managers {
			names, _ := brewfile.Declared(mgr)
			for _, n := range names {
				label := fmt.Sprintf("%-6s %s", mgr, n)
				labels = append(labels, label)
				byLabel[label] = pkg{mgr, n}
			}
		}
		if len(labels) == 0 {
			return fmt.Errorf("nothing recorded in the Brewfile")
		}
		sel, err := ui.MultiSelect("Remove which package(s)? — pick any, any manager", labels)
		if err != nil {
			return err
		}
		for _, s := range sel {
			picks = append(picks, byLabel[s])
		}
	}

	if len(picks) == 0 {
		return nil
	}
	for _, p := range picks {
		fmt.Printf("Removing %s (%s)...\n", p.name, p.mgr)
		if err := pm.Uninstall(p.mgr, p.name); err != nil {
			fmt.Printf("  (uninstall errored: %v; removing from Brewfile anyway)\n", err)
		}
		if err := brewfile.Remove(p.mgr, p.name); err != nil {
			return err
		}
	}
	if err := brewfile.Refresh(); err != nil {
		return err
	}
	fmt.Println("✓ removed from Brewfile")
	maybeCommit("casa: remove")
	return nil
}
