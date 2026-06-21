package cmd

import (
	"github.com/carrots-sh/casa/internal/app"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:   "tools",
	Short: "Manage packages across brew, cask, tap, go, uv, npm, cargo",
}

func init() {
	toolsCmd.AddCommand(
		&cobra.Command{
			Use:   "add [manager] [name]",
			Short: "Install a package and record it in your Brewfile",
			Args:  cobra.MaximumNArgs(2),
			RunE: func(cmd *cobra.Command, args []string) error {
				var mgr, name string
				if len(args) > 0 {
					mgr = args[0]
				}
				if len(args) > 1 {
					name = args[1]
				}
				return app.AddTool(mgr, name)
			},
		},
		&cobra.Command{
			Use:   "rm",
			Short: "Uninstall package(s) — pick from everything across all managers",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.RemoveTools() },
		},
		&cobra.Command{
			Use:   "update",
			Short: "Upgrade outdated packages — one, many, or all",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.UpdateTools() },
		},
		&cobra.Command{
			Use:   "list",
			Short: "List recorded packages",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.ListTools() },
		},
	)
}
