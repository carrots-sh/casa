// PATH self-healing: the supported managers install binaries into well-known
// directories that a minimal environment (fresh machine, cron, GUI-spawned
// shell) may not have on PATH yet. casa prepends the ones that exist so every
// manager — and the tools they installed — resolves. macOS and Linux only.
package pm

import (
	"os"
	"path/filepath"
	"strings"
)

// binDirs are the well-known install locations, most-specific first. Dirs
// that don't exist on this machine are skipped, which is what keeps this
// OS-sensitive without GOOS switches (linuxbrew paths never exist on macOS
// and vice versa).
func binDirs() []string {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, "go", "bin"),     // go install
		filepath.Join(home, ".local", "bin"), // uv tools, sh installers, chezmoi bootstrap
		filepath.Join(home, ".cargo", "bin"), // cargo install
		filepath.Join(home, ".bun", "bin"),   // bun add -g
		"/opt/homebrew/bin",                  // brew, macOS arm64
		"/opt/homebrew/sbin",
		"/opt/homebrew/opt/rustup/bin", // rustup via brew is keg-only
		"/usr/local/bin",               // brew, macOS intel (+ misc)
		"/usr/local/opt/rustup/bin",
		"/home/linuxbrew/.linuxbrew/bin", // brew, linux
		"/home/linuxbrew/.linuxbrew/sbin",
		"/home/linuxbrew/.linuxbrew/opt/rustup/bin",
	}
	if gobin := strings.TrimSpace(os.Getenv("GOBIN")); gobin != "" {
		dirs = append([]string{gobin}, dirs...)
	}
	if bi := strings.TrimSpace(os.Getenv("BUN_INSTALL")); bi != "" {
		dirs = append([]string{filepath.Join(bi, "bin")}, dirs...)
	}
	return dirs
}

// EnsurePath prepends every existing well-known bin dir that's missing from
// $PATH (for this process and everything casa spawns). Call once at startup.
// CASA_PLAIN_PATH=1 disables it (sandboxed tests mask managers via PATH).
func EnsurePath() {
	if os.Getenv("CASA_PLAIN_PATH") == "1" {
		return
	}
	sep := string(os.PathListSeparator)
	path := os.Getenv("PATH")
	have := map[string]bool{}
	for p := range strings.SplitSeq(path, sep) {
		have[p] = true
	}
	var add []string
	for _, d := range binDirs() {
		if have[d] {
			continue
		}
		have[d] = true
		if fi, err := os.Stat(d); err == nil && fi.IsDir() {
			add = append(add, d)
		}
	}
	if len(add) > 0 {
		os.Setenv("PATH", strings.Join(add, sep)+sep+path)
	}
}
