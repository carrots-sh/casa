// Package chez is a thin wrapper around the chezmoi CLI. casa never reimplements
// chezmoi behavior; it shells out so the user's repo stays the source of truth.
package chez

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

var srcCached string

// resolve picks casa's source dir, in order:
//  1. $CASA_SOURCE
//  2. ~/.local/share/casa (casa's branded default) if it's a repo
//  3. chezmoi's own configured/default source if it exists (backward-compatible)
//  4. ~/.local/share/casa (where a fresh `casa machine setup` will clone)
func resolve() string {
	if srcCached != "" {
		return srcCached
	}
	if v := strings.TrimSpace(os.Getenv("CASA_SOURCE")); v != "" {
		srcCached = v
		return srcCached
	}
	home, _ := os.UserHomeDir()
	casa := filepath.Join(home, ".local", "share", "casa")
	if _, err := os.Stat(filepath.Join(casa, ".git")); err == nil {
		srcCached = casa
		return srcCached
	}
	if o, err := exec.Command("chezmoi", "source-path").Output(); err == nil {
		if p := strings.TrimSpace(string(o)); p != "" {
			if _, err := os.Stat(p); err == nil {
				srcCached = p
				return srcCached
			}
		}
	}
	srcCached = casa
	return srcCached
}

// SetSource overrides the resolved source dir (used after `setup` clones).
func SetSource(dir string) { srcCached = dir }

// cmd builds a chezmoi command pinned to casa's source dir. The --source flag
// is used (not CHEZMOI_SOURCE_DIR) because some subcommands (e.g. managed)
// ignore the env var.
func cmd(args ...string) *exec.Cmd {
	return exec.Command("chezmoi", append([]string{"--source", resolve()}, args...)...)
}

func run(args ...string) error {
	c := cmd(args...)
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	return c.Run()
}

func out(args ...string) (string, error) {
	o, err := cmd(args...).Output()
	return string(o), err
}

// Available reports whether the chezmoi CLI is on PATH.
func Available() bool {
	_, err := exec.LookPath("chezmoi")
	return err == nil
}

// HasRepo reports whether casa's source dir is an initialized git repo.
func HasRepo() bool {
	_, err := os.Stat(filepath.Join(resolve(), ".git"))
	return err == nil
}

// Data returns chezmoi's template data as a generic map (chezmoi data --format json).
func Data() (map[string]any, error) {
	o, err := out("data", "--format", "json")
	if err != nil {
		return nil, err
	}
	var m map[string]any
	if err := json.Unmarshal([]byte(o), &m); err != nil {
		return nil, err
	}
	return m, nil
}

// SourceDir returns casa's source directory (the dotfiles repo).
func SourceDir() string { return resolve() }

// Managed returns the managed target files (paths relative to home).
func Managed() ([]string, error) {
	s, err := out("managed", "--include=files")
	if err != nil {
		return nil, err
	}
	return NonEmpty(s), nil
}

// Status returns the raw `chezmoi status` lines (drift between source and target).
func Status() ([]string, error) {
	s, err := out("status")
	if err != nil {
		return nil, err
	}
	return NonEmpty(s), nil
}

// Apply applies the given target paths (all if none), running scripts.
func Apply(paths ...string) error { return run(append([]string{"apply"}, paths...)...) }

// ApplyNoScripts applies without running run_ scripts (fast, for refreshes).
func ApplyNoScripts(paths ...string) error {
	return run(append([]string{"apply", "--exclude=scripts"}, paths...)...)
}

// Update pulls the repo and applies (catch this machine up).
func Update() error { return run("update") }

// InitApply clones repo into casa's source dir and applies it.
func InitApply(repo string) error { return run("init", "--apply", repo) }

// Edit opens a managed file in the configured editor and applies on close.
func Edit(homePath string) error { return run("edit", "--apply", homePath) }

// Add starts managing an existing file.
func Add(homePath string) error { return run("add", homePath) }

// AddEncrypt starts managing a file, encrypted.
func AddEncrypt(homePath string) error { return run("add", "--encrypt", homePath) }

// Forget stops managing a target (leaves the file in place).
func Forget(homePath string) error { return run("forget", "--force", homePath) }

// Cat prints the target state of a managed file (decrypts encrypted ones).
func Cat(homePath string) (string, error) { return out("cat", homePath) }

// Decrypt returns the plaintext of an encrypted source file (relative to source dir).
func Decrypt(sourceRelPath string) (string, error) {
	return out("decrypt", filepath.Join(SourceDir(), sourceRelPath))
}

// Doctor runs chezmoi's health check.
func Doctor() error { return run("doctor") }

// Git runs a git command inside the source repo.
func Git(args ...string) error {
	c := exec.Command("git", args...)
	c.Dir = SourceDir()
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	return c.Run()
}

// GitOut runs a git command in the source repo and returns stdout.
func GitOut(args ...string) (string, error) {
	c := exec.Command("git", args...)
	c.Dir = SourceDir()
	o, err := c.Output()
	return string(o), err
}

// EncryptedSources lists source-relative paths of encrypted files (managed
// targets and template fragments alike), found by the "encrypted_" name attribute.
func EncryptedSources() ([]string, error) {
	src := SourceDir()
	var found []string
	_ = filepath.WalkDir(src, func(p string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		if strings.Contains(d.Name(), "encrypted_") {
			if rel, err := filepath.Rel(src, p); err == nil {
				found = append(found, rel)
			}
		}
		return nil
	})
	return found, nil
}

// TargetPaths converts source-relative paths to their readable target paths
// (home-relative), e.g. "dot_ssh/encrypted_private_github.age" -> ".ssh/github".
// Output order matches input order.
func TargetPaths(sourceRels []string) ([]string, error) {
	if len(sourceRels) == 0 {
		return nil, nil
	}
	args := []string{"target-path"}
	for _, r := range sourceRels {
		args = append(args, filepath.Join(SourceDir(), r))
	}
	o, err := out(args...)
	if err != nil {
		return nil, err
	}
	home, _ := os.UserHomeDir()
	var rels []string
	for _, l := range strings.Split(strings.TrimRight(o, "\n"), "\n") {
		l = strings.TrimSpace(l)
		if l == "" {
			continue
		}
		if r, err := filepath.Rel(home, l); err == nil {
			rels = append(rels, r)
		} else {
			rels = append(rels, l)
		}
	}
	return rels, nil
}

// EncryptInto encrypts plaintext and writes the ciphertext to a source file.
func EncryptInto(plaintext, sourceRelPath string) error {
	c := exec.Command("chezmoi", "encrypt")
	c.Stdin = strings.NewReader(plaintext)
	cipher, err := c.Output()
	if err != nil {
		return fmt.Errorf("chezmoi encrypt: %w", err)
	}
	return os.WriteFile(filepath.Join(SourceDir(), sourceRelPath), cipher, 0o644)
}

// NonEmpty splits s into its non-blank lines.
func NonEmpty(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimRight(l, "\r"); strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
