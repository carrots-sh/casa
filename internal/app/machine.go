package app

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
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
	fmt.Println("changes to save:")
	_ = chez.Git("status", "--short")
	_ = chez.Git("--no-pager", "diff", "--stat")
	if msg == "" {
		msg = autoMessage(porcelain)
	}
	ok, err := ui.Confirm("commit + push?  “" + msg + "”")
	if err != nil || !ok {
		return err
	}
	return saveAll(msg)
}

// autoMessage builds a commit message from the names of changed files.
func autoMessage(porcelain string) string {
	seen := map[string]bool{}
	var names []string
	for _, l := range strings.Split(porcelain, "\n") {
		if strings.TrimSpace(l) == "" {
			continue
		}
		f := strings.TrimSpace(l)
		if i := strings.LastIndex(f, " "); i >= 0 {
			f = f[i+1:]
		}
		b := filepath.Base(f)
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
	fmt.Printf("local drift:       %d file(s) need apply\n", s.drift)
	fmt.Printf("outdated tools:    %d\n", s.updates)
	return nil
}

// Context lets you toggle this machine's contexts (the on/off setup answers)
// from a checklist, then re-applies. Falls back to re-asking the prompts if it
// can't read or write the values directly.
func Context() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	data, err := chez.Data()
	if err != nil {
		return rerunPrompts()
	}
	var keys, current []string
	for k, v := range data {
		if b, ok := v.(bool); ok {
			keys = append(keys, k)
			if b {
				current = append(current, k)
			}
		}
	}
	if len(keys) == 0 {
		return rerunPrompts()
	}
	sort.Strings(keys)
	sel, err := ui.MultiSelectPreselected("which contexts are on for this machine?", keys, current)
	if err != nil {
		return err
	}
	want := map[string]bool{}
	for _, k := range sel {
		want[k] = true
	}
	if err := setContextData(keys, want); err != nil {
		fmt.Println("couldn't update the config directly; re-asking the setup questions instead...")
		return rerunPrompts()
	}
	fmt.Println("applying...")
	return chez.Apply()
}

func rerunPrompts() error {
	c := exec.Command("chezmoi", "init", "--prompt")
	c.Stdin, c.Stdout, c.Stderr = os.Stdin, os.Stdout, os.Stderr
	if err := c.Run(); err != nil {
		return err
	}
	return chez.Apply()
}

// setContextData rewrites the bool context keys in ~/.config/chezmoi/chezmoi.toml.
func setContextData(keys []string, want map[string]bool) error {
	home, _ := os.UserHomeDir()
	cfg := filepath.Join(home, ".config", "chezmoi", "chezmoi.toml")
	data, err := os.ReadFile(cfg)
	if err != nil {
		return err
	}
	_ = os.WriteFile(cfg+".casa.bak", data, 0o644) // safety backup
	lines := strings.Split(string(data), "\n")
	set := map[string]bool{}
	for i, l := range lines {
		for _, k := range keys {
			re := regexp.MustCompile(`^(\s*` + regexp.QuoteMeta(k) + `\s*=\s*)(true|false)\s*$`)
			if m := re.FindStringSubmatch(l); m != nil {
				lines[i] = fmt.Sprintf("%s%t", m[1], want[k])
				set[k] = true
			}
		}
	}
	for _, k := range keys {
		if !set[k] {
			return fmt.Errorf("context %q is not a simple value in the config", k)
		}
	}
	return os.WriteFile(cfg, []byte(strings.Join(lines, "\n")), 0o644)
}

// Undo reverts the last saved change and re-applies.
func Undo() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	last := strings.TrimSpace(mustOut(chez.GitOut("log", "-1", "--oneline")))
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
	return chez.Apply()
}

func mustOut(s string, _ error) string { return s }

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
