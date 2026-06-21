package app

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/config"
	"github.com/carrots-sh/casa/internal/ui"
)

// Setup provisions a new machine from a dotfiles repo (chezmoi init --apply).
func Setup(repo string) error {
	if !chez.Available() {
		return fmt.Errorf("install chezmoi first: brew install chezmoi")
	}
	if repo == "" {
		repo = config.Load().Setup.Repo
	}
	if repo == "" {
		var err error
		if repo, err = ui.Input("your dotfiles repo (e.g. your-username or user/repo)"); err != nil || repo == "" {
			return err
		}
	}
	fmt.Printf("setting up this machine from %s...\n", repo)
	c := exec.Command("chezmoi", "init", "--apply", repo)
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	return c.Run()
}

// Sync pulls the repo and applies it here.
func Sync() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	fmt.Println("catching this machine up...")
	if err := chez.Update(); err != nil {
		return err
	}
	fmt.Println("✓ up to date")
	return nil
}

// Save shows pending changes then commits + pushes.
func Save(msg string) error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	porcelain, _ := chez.GitOut("status", "--porcelain")
	if strings.TrimSpace(porcelain) == "" {
		fmt.Println("nothing to save — everything's already committed.")
		return nil
	}
	fmt.Println("changes to save:")
	_ = chez.Git("status", "--short")
	if msg == "" {
		var err error
		if msg, err = ui.Input("describe this change"); err != nil {
			return err
		}
		if msg == "" {
			msg = "casa: update dotfiles"
		}
	}
	ok, err := ui.Confirm("commit + push these?")
	if err != nil || !ok {
		return err
	}
	return saveAll(msg)
}

// Status prints the full overview.
func Status() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	s := computeStatus()
	fmt.Printf("machine:           %s\n", s.machine)
	fmt.Printf("unsaved changes:   %d\n", s.toSave)
	fmt.Printf("behind your repo:  %d commit(s)\n", s.behind)
	fmt.Printf("local drift:       %d file(s) need apply\n", s.drift)
	fmt.Printf("outdated tools:    %d\n", s.updates)
	return nil
}

// Context re-asks this machine's setup questions (contexts) and re-applies.
func Context() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	fmt.Println("re-running this machine's setup questions...")
	c := exec.Command("chezmoi", "init", "--prompt")
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return err
	}
	return chez.Apply()
}

// Doctor runs chezmoi's health check.
func Doctor() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	return chez.Doctor()
}

// Info prints machine + repo basics.
func Info() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	src, _ := chez.SourceDir()
	managed, _ := chez.Managed()
	fmt.Printf("machine:  %s\n", machineName())
	fmt.Printf("repo:     %s\n", src)
	fmt.Printf("managed:  %d files\n", len(managed))
	return nil
}
