// Package cmd is casa's cobra command tree. Running casa with no subcommand
// opens the interactive menu; the subcommands are the same actions for scripting.
package cmd

import (
	"github.com/carrots-sh/casa/internal/app"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "casa",
	Short: "easy chezmoi — manage your configs and tools from one friendly menu",
	Long: "casa is a friendly front-end for chezmoi. run it with no arguments for an\n" +
		"interactive menu; everything is guided, so there's nothing to memorize.",
	SilenceUsage: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return app.Menu()
	},
}

// Execute runs the root command.
func Execute(version, commit string) {
	rootCmd.Version = version + " (" + commit + ")"
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.CompletionOptions.DisableDefaultCmd = true
	rootCmd.AddCommand(toolsCmd, configsCmd, secretsCmd, machineCmd)
}
