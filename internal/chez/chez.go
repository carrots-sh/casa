// Package chez is a thin wrapper around the chezmoi CLI. casa never reimplements
// chezmoi behavior; it shells out so the user's repo stays the source of truth.
package chez

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

func run(args ...string) error {
	c := exec.Command("chezmoi", args...)
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	return c.Run()
}

func out(args ...string) (string, error) {
	o, err := exec.Command("chezmoi", args...).Output()
	return string(o), err
}

// Available reports whether the chezmoi CLI is on PATH.
func Available() bool {
	_, err := exec.LookPath("chezmoi")
	return err == nil
}

// SourceDir returns the chezmoi source directory (the dotfiles repo).
func SourceDir() (string, error) {
	s, err := out("source-path")
	if err != nil {
		return "", fmt.Errorf("chezmoi source-path: %w", err)
	}
	return strings.TrimSpace(s), nil
}

// Managed returns the managed target files (paths relative to home).
func Managed() ([]string, error) {
	s, err := out("managed", "--include=files")
	if err != nil {
		return nil, err
	}
	return nonEmpty(s), nil
}

// Status returns the raw `chezmoi status` lines (drift between source and target).
func Status() ([]string, error) {
	s, err := out("status")
	if err != nil {
		return nil, err
	}
	return nonEmpty(s), nil
}

// Apply applies the given target paths (all if none), running scripts.
func Apply(paths ...string) error { return run(append([]string{"apply"}, paths...)...) }

// ApplyNoScripts applies without running run_ scripts (fast, for refreshes).
func ApplyNoScripts(paths ...string) error {
	return run(append([]string{"apply", "--exclude=scripts"}, paths...)...)
}

// Update pulls the repo and applies (catch this machine up).
func Update() error { return run("update") }

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
	src, err := SourceDir()
	if err != nil {
		return "", err
	}
	return out("decrypt", filepath.Join(src, sourceRelPath))
}

// Doctor runs chezmoi's health check.
func Doctor() error { return run("doctor") }

// Git runs a git command inside the source repo.
func Git(args ...string) error {
	src, err := SourceDir()
	if err != nil {
		return err
	}
	c := exec.Command("git", args...)
	c.Dir = src
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	return c.Run()
}

// GitOut runs a git command in the source repo and returns stdout.
func GitOut(args ...string) (string, error) {
	src, err := SourceDir()
	if err != nil {
		return "", err
	}
	c := exec.Command("git", args...)
	c.Dir = src
	o, err := c.Output()
	return string(o), err
}

// EncryptedSources lists source-relative paths of encrypted files (managed
// targets and template fragments alike), found by the "encrypted_" name attribute.
func EncryptedSources() ([]string, error) {
	src, err := SourceDir()
	if err != nil {
		return nil, err
	}
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

// EncryptInto encrypts plaintext and writes the ciphertext to a source file.
func EncryptInto(plaintext, sourceRelPath string) error {
	src, err := SourceDir()
	if err != nil {
		return err
	}
	c := exec.Command("chezmoi", "encrypt")
	c.Stdin = strings.NewReader(plaintext)
	cipher, err := c.Output()
	if err != nil {
		return fmt.Errorf("chezmoi encrypt: %w", err)
	}
	return os.WriteFile(filepath.Join(src, sourceRelPath), cipher, 0o644)
}

func nonEmpty(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimRight(l, "\r"); strings.TrimSpace(l) != "" {
			out = append(out, l)
		}
	}
	return out
}
