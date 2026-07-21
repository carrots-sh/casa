// Package app holds casa's actions (the work behind each command/menu item) and
// the interactive menu. Both cobra commands and the menu call into here.
package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/config"
	"github.com/carrots-sh/casa/internal/manifest"
	"github.com/carrots-sh/casa/internal/ui"
)

// mf builds the package-manifest handle from config.
func mf() manifest.Manifest {
	return manifest.Manifest{Path: config.Load().ManifestPath()}
}

// requireChezmoi makes sure chezmoi is available, offering to install it on a
// fresh machine (brew when present, otherwise chezmoi's own installer into
// ~/.local/bin) so plain `casa` works from nothing.
func requireChezmoi() error {
	if chez.Available() {
		return nil
	}
	fmt.Println("casa drives chezmoi under the hood, and it isn't installed yet.")
	ok, err := ui.Confirm("install chezmoi now?")
	if err != nil || !ok {
		return fmt.Errorf("chezmoi is not installed — run: brew install chezmoi")
	}
	if _, berr := exec.LookPath("brew"); berr == nil {
		if err := runShell("brew", "install", "chezmoi"); err != nil {
			return fmt.Errorf("brew install chezmoi failed: %w", err)
		}
	} else {
		home, _ := os.UserHomeDir()
		bin := filepath.Join(home, ".local", "bin")
		_ = os.MkdirAll(bin, 0o755)
		if err := runShell("sh", "-c", `sh -c "$(curl -fsSL get.chezmoi.io)" -- -b `+bin); err != nil {
			return fmt.Errorf("chezmoi install failed: %w", err)
		}
		// make it reachable for this process; later shells need it on PATH too
		os.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
		fmt.Printf("  installed to %s — add it to your PATH if it isn't already\n", bin)
	}
	if !chez.Available() {
		return fmt.Errorf("chezmoi still not found on PATH after install")
	}
	fmt.Println("✓ chezmoi installed")
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
