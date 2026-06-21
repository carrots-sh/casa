// Package cmd holds casa's cobra command tree.
package cmd

import (
	"os/exec"
	"strings"

	"github.com/carrots-sh/casa/internal/ui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "casa",
	Short: "Add, remove, and update packages across brew/cask/tap/go/uv/npm/cargo — kept in sync with your Brewfile.",
	RunE: func(cmd *cobra.Command, args []string) error {
		action, err := ui.Select("casa:", []string{"add", "remove", "update"})
		if err != nil {
			return err
		}
		switch action {
		case "add":
			return runAdd(nil)
		case "remove":
			return runRemove(nil)
		case "update":
			return runUpdate()
		}
		return nil
	},
}

// Execute runs the root command.
func Execute(version, commit string) {
	rootCmd.Version = version + " (" + commit + ")"
	cobra.CheckErr(rootCmd.Execute())
}

func init() {
	rootCmd.AddCommand(addCmd, removeCmd, updateCmd)
}

// maybeCommit offers to commit + push the Brewfile change via chezmoi's git.
func maybeCommit(msg string) {
	ok, err := ui.Confirm("Commit this change to your dotfiles?")
	if err != nil || !ok {
		println("Not committed. Later: chezmoi git -- add -A && chezmoi git -- commit -m '...' && chezmoi git -- push")
		return
	}
	out, err := exec.Command("chezmoi", "source-path").Output()
	if err != nil {
		return
	}
	dir := strings.TrimSpace(string(out))
	for _, args := range [][]string{{"add", "-A"}, {"commit", "-m", msg}, {"push"}} {
		c := exec.Command("git", args...)
		c.Dir = dir
		_ = c.Run()
	}
	println("✓ committed + pushed")
}
