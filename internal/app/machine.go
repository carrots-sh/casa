package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/config"
	"github.com/carrots-sh/casa/internal/pm"
	"github.com/carrots-sh/casa/internal/ui"
)

// Setup provisions a new machine from a dotfiles repo. It accepts a github
// username (→ <user>/dotfiles), a user/repo, or a full URL; prefers SSH and
// falls back to HTTPS; and clones into casa's branded source dir (~/.local/share/casa).
func Setup(arg string) error {
	if err := requireChezmoi(); err != nil {
		return err
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

	if _, err := os.Stat(filepath.Join(target, ".git")); err != nil {
		url, err := pickRepoURL(arg)
		if err != nil {
			return err
		}
		fmt.Printf("setting up this machine from %s\n  into %s ...\n", url, target)
		_ = os.MkdirAll(filepath.Dir(target), 0o755)
		if err := runShell("git", "clone", url, target); err != nil {
			return err
		}
	} else {
		fmt.Printf("using the repo already at %s\n", target)
	}
	// ask the repo's setup questions in casa's UI, then render + apply
	if err := askSetupQuestions(); err != nil {
		return err
	}
	if err := offerBrew(); err != nil {
		return err
	}
	fmt.Println("applying your dotfiles...")
	invalidateStatus()
	return chez.Apply()
}

// offerBrew installs Homebrew on a fresh machine (with consent) so the
// packages script has something to run — declining is fine, packages just
// skip until brew shows up.
func offerBrew() error {
	if _, err := exec.LookPath("brew"); err == nil {
		return nil
	}
	ok, err := ui.ConfirmDefault("Homebrew isn't installed — install it now? (your packages need it)", true)
	if err != nil || !ok {
		if err == nil {
			fmt.Println("skipping — packages won't install until brew exists (casa machine doctor shows how)")
		}
		return err
	}
	fmt.Println("installing Homebrew...")
	if err := runShell("/bin/bash", "-c",
		`NONINTERACTIVE=1 /bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"`); err != nil {
		return fmt.Errorf("homebrew install failed: %w", err)
	}
	pm.EnsurePath() // pick up /opt/homebrew/bin or linuxbrew immediately
	if _, err := exec.LookPath("brew"); err != nil {
		fmt.Println("note: brew installed but not on PATH yet — restart your shell after setup")
	}
	return nil
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

// Sync brings this machine fully up to date: upgrade packages, then pull the
// repo and apply it here. (Replaces the old `sysupdate` shell function.)
func Sync() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if _, err := exec.LookPath("brew"); err == nil {
		fmt.Println("upgrading packages...")
		_ = runShell("brew", "update")
		_ = runShell("brew", "upgrade")
		_ = runShell("brew", "cleanup")
	}
	fmt.Println("syncing dotfiles...")
	if err := chez.Update(); err != nil {
		return err
	}
	invalidateStatus()
	fmt.Println("✓ up to date  (restart your shell to pick up changes)")
	return nil
}

func runShell(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	return c.Run()
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
	// Show changes by their readable target paths, not raw source names.
	sources := changedPaths(porcelain)
	targets := targetLabels(sources)
	fmt.Println("changes to save:")
	for _, t := range targets {
		fmt.Println("  " + t)
	}
	if msg == "" {
		msg = autoMessageFrom(targets)
	}
	fmt.Println("committing: " + msg)
	return saveAll(msg)
}

// changedPaths extracts the source-relative paths from `git status --porcelain`.
func changedPaths(porcelain string) []string {
	var out []string
	for l := range strings.SplitSeq(porcelain, "\n") {
		if len(l) < 4 {
			continue
		}
		p := l[3:] // after the two-char status + space
		if i := strings.Index(p, " -> "); i >= 0 {
			p = p[i+4:] // a rename: take the new path
		}
		p = strings.Trim(p, "\"")
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// autoMessageFrom builds a commit message from the basenames of changed files.
func autoMessageFrom(paths []string) string {
	seen := map[string]bool{}
	var names []string
	for _, p := range paths {
		b := filepath.Base(p)
		if b != "" && !seen[b] {
			seen[b] = true
			names = append(names, b)
		}
	}
	switch {
	case len(names) == 0:
		return "casa: update dotfiles"
	case len(names) > 3:
		return fmt.Sprintf("casa: update %s and %d more", strings.Join(names[:3], ", "), len(names)-3)
	default:
		return "casa: update " + strings.Join(names, ", ")
	}
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
	fmt.Printf("local drift:       %d file(s) to review (casa files drift)\n", driftCount())
	if n := pendingScripts(); n > 0 {
		fmt.Printf("pending scripts:   %d (run on the next casa sync)\n", n)
	}
	fmt.Printf("outdated tools:    %d\n", outdatedCount())
	return nil
}

// rerunPrompts is the fallback when the questionnaire can't be parsed:
// chezmoi asks its own prompts on the terminal, then casa applies.
func rerunPrompts() error {
	if err := chez.Init("--prompt"); err != nil {
		return err
	}
	invalidateStatus()
	return chez.Apply()
}

// Undo reverts the last saved change and re-applies.
func Undo() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	out, _ := chez.GitOut("log", "-1", "--oneline")
	last := strings.TrimSpace(out)
	if last == "" {
		return fmt.Errorf("nothing to undo")
	}
	ok, err := ui.Confirm("undo last change?  " + last)
	if err != nil || !ok {
		return err
	}
	if err := chez.Git("revert", "--no-edit", "HEAD"); err != nil {
		return err
	}
	_ = chez.Git("push")
	fmt.Println("applying the revert...")
	invalidateStatus()
	return chez.Apply()
}

// Info prints machine + repo basics.
func Info() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	managed, _ := chez.Managed()
	fmt.Printf("machine:  %s\n", machineName())
	fmt.Printf("repo:     %s\n", chez.SourceDir())
	fmt.Printf("managed:  %d files\n", len(managed))
	return nil
}
