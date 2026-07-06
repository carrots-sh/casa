// Package app holds casa's actions (the work behind each command/menu item) and
// the interactive menu. Both cobra commands and the menu call into here.
package app

import (
	"fmt"
	"strings"

	"github.com/carrots-sh/casa/internal/brewfile"
	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/config"
)

// bf builds the Brewfile handle from config.
func bf() brewfile.Brewfile {
	c := config.Load()
	return brewfile.Brewfile{Tmpl: c.BrewfileTmpl(), Anchor: "casa"}
}

// requireChezmoi returns an error if chezmoi isn't installed.
func requireChezmoi() error {
	if !chez.Available() {
		return fmt.Errorf("chezmoi is not installed — run: brew install chezmoi")
	}
	return nil
}

// offerSave asks to commit+push after a change.
func offerSave(msg string) {
	invalidateStatus()
	if err := Save(msg); err != nil {
		fmt.Println(err)
	}
}

// saveAll stages, commits, and pushes the source repo (no-op if clean).
func saveAll(msg string) error {
	porcelain, _ := chez.GitOut("status", "--porcelain")
	if strings.TrimSpace(porcelain) == "" {
		fmt.Println("nothing to save — everything's already committed.")
		return nil
	}
	if err := chez.Git("add", "-A"); err != nil {
		return err
	}
	if err := chez.Git("commit", "-m", msg); err != nil {
		return err
	}
	if err := chez.Git("push"); err != nil {
		fmt.Println("committed locally, but push failed — push later from casa → save.")
		return nil
	}
	fmt.Println("✓ saved + pushed  (casa undo to revert)")
	return nil
}
