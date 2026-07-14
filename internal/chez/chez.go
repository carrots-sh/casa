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
func SetSource(dir string) { srcCached, mirrored = dir, false }

// casaNames maps casa-named special files to the chezmoi names chezmoi reads.
// casa repos use the casa names; mirror symlinks them so chezmoi still works.
// ponytail: files only — chezmoi doesn't reliably walk symlinked dirs
// (.casatemplates etc.); add per-entry copying if anyone needs those.
var casaNames = map[string]string{
	".casa.toml.tmpl":    ".chezmoi.toml.tmpl",
	".casa.yaml.tmpl":    ".chezmoi.yaml.tmpl",
	".casa.json.tmpl":    ".chezmoi.json.tmpl",
	".casaignore":        ".chezmoiignore",
	".casaremove":        ".chezmoiremove",
	".casaversion":       ".chezmoiversion",
	".casaexternal.toml": ".chezmoiexternal.toml",
	".casadata.toml":     ".chezmoidata.toml",
	".casadata.yaml":     ".chezmoidata.yaml",
	".casadata.json":     ".chezmoidata.json",
}

var mirrored bool

// mirror links each casa-named special file to its chezmoi name (gitignored)
// so the repo reads casa-first while chezmoi finds what it expects.
func mirror() {
	if mirrored {
		return
	}
	mirrored = true
	src := resolve()
	var created []string
	for casa, chezName := range casaNames {
		if _, err := os.Lstat(filepath.Join(src, casa)); err != nil {
			continue
		}
		link := filepath.Join(src, chezName)
		if _, err := os.Lstat(link); err == nil {
			continue // already linked, or the user's own real file
		}
		if os.Symlink(casa, link) == nil {
			created = append(created, chezName)
		}
	}
	if len(created) > 0 {
		ensureGitignore(src, created)
	}
}

// ensureGitignore appends names to the repo's .gitignore if missing, so the
// mirrored links never show up in the save flow.
func ensureGitignore(src string, names []string) {
	p := filepath.Join(src, ".gitignore")
	data, _ := os.ReadFile(p)
	have := map[string]bool{}
	for _, l := range NonEmpty(string(data)) {
		have[strings.TrimSpace(l)] = true
	}
	out := string(data)
	changed := false
	for _, n := range names {
		if have[n] {
			continue
		}
		if out != "" && !strings.HasSuffix(out, "\n") {
			out += "\n"
		}
		out += n + "\n"
		changed = true
	}
	if changed {
		_ = os.WriteFile(p, []byte(out), 0o644)
	}
}

// cmd builds a chezmoi command pinned to casa's source dir. The --source flag
// is used (not CHEZMOI_SOURCE_DIR) because some subcommands (e.g. managed)
// ignore the env var.
func cmd(args ...string) *exec.Cmd {
	mirror()
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

// ExecuteTemplate renders a template string (as apply would), returning any
// template error. Used to validate edited templates before saving them.
func ExecuteTemplate(tmpl string) error {
	c := cmd("execute-template")
	c.Stdin = strings.NewReader(tmpl)
	if o, err := c.CombinedOutput(); err != nil {
		return fmt.Errorf("%s", strings.TrimSpace(string(o)))
	}
	return nil
}

// Edit opens a managed file in the configured editor and applies on close.
func Edit(homePath string) error { return run("edit", "--apply", homePath) }

// Add starts managing an existing file.
func Add(homePath string) error { return run("add", homePath) }

// AddEncrypt starts managing a file, encrypted.
func AddEncrypt(homePath string) error { return run("add", "--encrypt", homePath) }

// AddTemplate starts managing a file as a template, auto-substituting known
// data values (email, hostname, …) with {{ .var }} references.
func AddTemplate(homePath string) error { return run("add", "--autotemplate", homePath) }

// AddEncryptedTemplate starts managing a file as an encrypted template.
func AddEncryptedTemplate(homePath string) error {
	return run("add", "--encrypt", "--template", homePath)
}

// Chattr changes a target's storage attributes (e.g. "+template,-encrypted")
// by renaming/re-encoding its source.
func Chattr(mods, homePath string) error { return run("chattr", mods, homePath) }

// Init re-renders the machine config from the source questionnaire (and clones
// first when a repo is given). Prompts read the terminal unless answered via
// --promptString/Bool/Int/Choice/Multichoice flags.
func Init(args ...string) error { return run(append([]string{"init"}, args...)...) }

// ConfigTemplate returns the setup questionnaire's path, casa-named first.
// Symlinks are skipped so a mirrored link never shadows the real file.
func ConfigTemplate() (string, bool) {
	for _, n := range []string{
		".casa.toml.tmpl", ".casa.yaml.tmpl", ".casa.json.tmpl",
		".chezmoi.toml.tmpl", ".chezmoi.yaml.tmpl", ".chezmoi.json.tmpl",
	} {
		p := filepath.Join(SourceDir(), n)
		if fi, err := os.Lstat(p); err == nil && fi.Mode()&os.ModeSymlink == 0 {
			return p, true
		}
	}
	return "", false
}

// SourcePaths converts target paths (absolute) to their source paths,
// one per input, in order.
func SourcePaths(homePaths []string) ([]string, error) {
	if len(homePaths) == 0 {
		return nil, nil
	}
	o, err := out(append([]string{"source-path"}, homePaths...)...)
	if err != nil {
		return nil, err
	}
	return NonEmpty(o), nil
}

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
