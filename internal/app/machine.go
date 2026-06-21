package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/config"
	"github.com/carrots-sh/casa/internal/ui"
)

// Setup provisions a new machine from a dotfiles repo. It accepts a github
// username (→ <user>/dotfiles), a user/repo, or a full URL; prefers SSH and
// falls back to HTTPS; and clones into casa's branded source dir (~/.local/share/casa).
func Setup(arg string) error {
	if !chez.Available() {
		return fmt.Errorf("install chezmoi first: brew install chezmoi")
	}
	if arg == "" {
		arg = config.Load().Setup.Repo
	}
	if arg == "" {
		var err error
		if arg, err = ui.Input("github username, user/repo, or repo url"); err != nil || arg == "" {
			return err
		}
	}

	// pin the source dir (default: casa's branded location, overridable via $CASA_SOURCE)
	target := os.Getenv("CASA_SOURCE")
	if target == "" {
		home, _ := os.UserHomeDir()
		target = filepath.Join(home, ".local", "share", "casa")
	}
	chez.SetSource(target)

	url, err := pickRepoURL(arg)
	if err != nil {
		return err
	}
	fmt.Printf("setting up this machine from %s\n  into %s ...\n", url, target)
	return chez.InitApply(url)
}

// pickRepoURL resolves arg to a reachable clone URL, preferring SSH then HTTPS.
func pickRepoURL(arg string) (string, error) {
	ssh, https := repoURLs(arg)
	if ssh == https { // explicit full URL — no fallback to try
		if reachable(ssh) {
			return ssh, nil
		}
		return "", fmt.Errorf("could not reach %s", ssh)
	}
	if reachable(ssh) {
		return ssh, nil
	}
	fmt.Println("ssh not available, trying https...")
	if reachable(https) {
		return https, nil
	}
	return "", fmt.Errorf("could not reach the repo over ssh or https (checked %s and %s)", ssh, https)
}

// repoURLs derives the SSH and HTTPS forms from a username, user/repo, or URL.
func repoURLs(arg string) (ssh, https string) {
	switch {
	case strings.Contains(arg, "://") || strings.HasPrefix(arg, "git@"):
		return arg, arg
	case strings.Contains(arg, "/"):
		return "git@github.com:" + arg + ".git", "https://github.com/" + arg + ".git"
	default:
		return "git@github.com:" + arg + "/dotfiles.git", "https://github.com/" + arg + "/dotfiles.git"
	}
}

// reachable tests a clone URL without prompting (no password/host prompts hang).
func reachable(url string) bool {
	c := exec.Command("git", "ls-remote", url)
	c.Env = append(os.Environ(),
		"GIT_TERMINAL_PROMPT=0",
		"GIT_SSH_COMMAND=ssh -oBatchMode=yes -oConnectTimeout=8 -oStrictHostKeyChecking=accept-new",
	)
	return c.Run() == nil
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
