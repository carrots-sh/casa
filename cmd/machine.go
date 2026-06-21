package cmd

import (
	"github.com/carrots-sh/casa/internal/app"
	"github.com/spf13/cobra"
)

var machineCmd = &cobra.Command{
	Use:   "machine",
	Short: "set up, sync, and inspect this machine",
}

func init() {
	machineCmd.AddCommand(
		&cobra.Command{
			Use:   "setup [repo]",
			Short: "provision this machine from your dotfiles repo",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				repo := ""
				if len(args) > 0 {
					repo = args[0]
				}
				return app.Setup(repo)
			},
		},
		&cobra.Command{
			Use:   "sync",
			Short: "pull your repo and apply it here",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.Sync() },
		},
		&cobra.Command{
			Use:   "save [message]",
			Short: "commit + push your changes",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				msg := ""
				if len(args) > 0 {
					msg = args[0]
				}
				return app.Save(msg)
			},
		},
		&cobra.Command{
			Use:   "status",
			Short: "show what's changed, behind, or outdated",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.Status() },
		},
		&cobra.Command{
			Use:   "context",
			Short: "change this machine's setup answers (contexts) and re-apply",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.Context() },
		},
		&cobra.Command{
			Use:   "doctor",
			Short: "health check",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.Doctor() },
		},
		&cobra.Command{
			Use:   "info",
			Short: "machine + repo basics",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.Info() },
		},
	)
}
