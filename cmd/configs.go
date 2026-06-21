package cmd

import (
	"github.com/carrots-sh/casa/internal/app"
	"github.com/spf13/cobra"
)

var configsCmd = &cobra.Command{
	Use:   "configs",
	Short: "manage your dotfiles",
}

func init() {
	configsCmd.AddCommand(
		&cobra.Command{
			Use:   "edit [name]",
			Short: "pick and edit a config (encrypted ones handled transparently)",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				name := ""
				if len(args) > 0 {
					name = args[0]
				}
				return app.EditConfig(name)
			},
		},
		&cobra.Command{
			Use:   "track [path]",
			Short: "start managing an existing file",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				path := ""
				if len(args) > 0 {
					path = args[0]
				}
				return app.TrackFile(path)
			},
		},
		&cobra.Command{
			Use:   "untrack [path]",
			Short: "stop managing a file (keeps it on disk)",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				path := ""
				if len(args) > 0 {
					path = args[0]
				}
				return app.UntrackFile(path)
			},
		},
		&cobra.Command{
			Use:   "list",
			Short: "list managed files",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.ListConfigs() },
		},
	)
}
