package cmd

import (
	"github.com/carrots-sh/casa/internal/app"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "manage encrypted files",
}

func init() {
	secretsCmd.AddCommand(
		&cobra.Command{
			Use:   "add [path]",
			Short: "encrypt and start managing a file",
			Args:  cobra.MaximumNArgs(1),
			RunE: func(cmd *cobra.Command, args []string) error {
				path := ""
				if len(args) > 0 {
					path = args[0]
				}
				return app.AddSecret(path)
			},
		},
		&cobra.Command{
			Use:   "edit",
			Short: "pick a secret, decrypt, edit, re-encrypt",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.EditSecret() },
		},
		&cobra.Command{
			Use:   "list",
			Short: "list encrypted files",
			RunE:  func(cmd *cobra.Command, args []string) error { return app.ListSecrets() },
		},
	)
}
