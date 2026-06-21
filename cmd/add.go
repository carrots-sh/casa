package cmd

import (
	"fmt"

	"github.com/carrots-sh/casa/internal/brewfile"
	"github.com/carrots-sh/casa/internal/pm"
	"github.com/carrots-sh/casa/internal/ui"
	"github.com/spf13/cobra"
)

var addCmd = &cobra.Command{
	Use:   "add [manager] [name]",
	Short: "Install a package and record it in the Brewfile",
	Args:  cobra.MaximumNArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		return runAdd(args)
	},
}

func runAdd(args []string) error {
	var mgr, name string
	if len(args) >= 1 {
		mgr = args[0]
	}
	if len(args) >= 2 {
		name = args[1]
	}

	var err error
	if mgr == "" {
		if mgr, err = ui.Select("Package manager:", pm.Managers); err != nil || mgr == "" {
			return err
		}
	}
	if name == "" {
		if name, err = ui.Input("package name (" + mgr + ")"); err != nil || name == "" {
			return err
		}
	}

	fmt.Printf("Installing %s via %s...\n", name, mgr)
	if err := pm.Install(mgr, name); err != nil {
		return fmt.Errorf("install failed: %w", err)
	}
	if err := brewfile.Add(mgr, name); err != nil {
		return err
	}
	if err := brewfile.Refresh(); err != nil {
		return err
	}
	fmt.Printf("✓ added: %s %q (installed + recorded in Brewfile)\n", mgr, name)
	maybeCommit(fmt.Sprintf("casa: add %s %s", mgr, name))
	return nil
}
