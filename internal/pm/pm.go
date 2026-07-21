// Package pm wraps the supported package managers behind a uniform interface.
package pm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
)

// Managers is the set of supported managers, in display order.
var Managers = []string{"brew", "cask", "tap", "go", "uv", "npm", "cargo"}

func run(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	return c.Run()
}

func capture(name string, args ...string) string {
	out, _ := exec.Command(name, args...).Output()
	return string(out)
}

// Install installs name via mgr.
func Install(mgr, name string) error {
	switch mgr {
	case "brew":
		return run("brew", "install", name)
	case "cask":
		return run("brew", "install", "--cask", name)
	case "tap":
		return run("brew", "tap", name)
	case "go":
		return run("go", "install", name+"@latest")
	case "uv":
		return run("uv", "tool", "install", name)
	case "npm":
		return run("npm", "install", "-g", name)
	case "cargo":
		return run("cargo", "install", name)
	}
	return fmt.Errorf("unknown manager %q", mgr)
}

// Uninstall removes name via mgr.
func Uninstall(mgr, name string) error {
	switch mgr {
	case "brew":
		return run("brew", "uninstall", name)
	case "cask":
		return run("brew", "uninstall", "--cask", name)
	case "tap":
		return run("brew", "untap", name)
	case "go":
		gopath := strings.TrimSpace(capture("go", "env", "GOPATH"))
		if gopath == "" {
			return fmt.Errorf("could not resolve GOPATH")
		}
		return os.Remove(filepath.Join(gopath, "bin", filepath.Base(name)))
	case "uv":
		return run("uv", "tool", "uninstall", name)
	case "npm":
		return run("npm", "uninstall", "-g", name)
	case "cargo":
		return run("cargo", "uninstall", name)
	}
	return fmt.Errorf("unknown manager %q", mgr)
}

// Upgrade upgrades a single named package (brew/cask/npm).
func Upgrade(mgr, name string) error {
	switch mgr {
	case "brew":
		return run("brew", "upgrade", "--formula", name)
	case "cask":
		return run("brew", "upgrade", "--cask", name)
	case "npm":
		return run("npm", "install", "-g", name+"@latest")
	}
	return nil
}

// Outdated returns "mgr name" items that have updates, for the managers that
// expose per-package outdated info cleanly (brew, cask, npm).
func Outdated() []string {
	var items []string
	for _, n := range lines(capture("brew", "outdated", "--formula", "--quiet")) {
		items = append(items, "brew "+n)
	}
	for _, n := range lines(capture("brew", "outdated", "--cask", "--quiet")) {
		items = append(items, "cask "+n)
	}
	for _, l := range lines(capture("npm", "-g", "outdated", "--parseable")) {
		// format: <path>:<name>@<wanted>:<name>@<current>:<name>@<latest>
		parts := strings.Split(l, ":")
		if len(parts) >= 2 {
			nv := parts[1]
			if i := strings.LastIndex(nv, "@"); i > 0 {
				items = append(items, "npm "+nv[:i])
			}
		}
	}
	return items
}

// UpgradeAllBrew upgrades every outdated formula and cask in a single brew call.
func UpgradeAllBrew() error { return run("brew", "upgrade") }

// UpgradeAll runs a manager-wide upgrade for tools without per-package outdated info.
func UpgradeAll(mgr string) error {
	switch mgr {
	case "uv":
		return run("uv", "tool", "upgrade", "--all")
	case "cargo":
		if _, err := exec.LookPath("cargo-install-update"); err != nil {
			return fmt.Errorf("install cargo-update first: cargo install cargo-update")
		}
		return run("cargo", "install-update", "-a")
	}
	return nil
}

// Installed returns the top-level packages currently installed via mgr —
// used to seed the manifest from an existing machine. Best-effort: a missing
// manager just returns nothing.
func Installed(mgr string) []string {
	switch mgr {
	case "tap":
		var out []string
		for _, t := range lines(capture("brew", "tap")) {
			if t != "homebrew/core" && t != "homebrew/cask" && t != "homebrew/bundle" {
				out = append(out, t)
			}
		}
		return out
	case "brew":
		if out := lines(capture("brew", "leaves", "--installed-on-request")); len(out) > 0 {
			return out
		}
		return lines(capture("brew", "leaves"))
	case "cask":
		return lines(capture("brew", "list", "--cask"))
	case "go":
		return goInstalled()
	case "uv":
		// `uv tool list` prints "name v1.2.3" headers with "- <exe>" bullets under each.
		var out []string
		for _, l := range lines(capture("uv", "tool", "list")) {
			if strings.HasPrefix(l, "-") || strings.HasPrefix(l, "warning") {
				continue
			}
			if f := strings.Fields(l); len(f) >= 2 && strings.HasPrefix(f[1], "v") {
				out = append(out, f[0])
			}
		}
		return out
	case "npm":
		// parseable: one node_modules path per line; scoped names keep their @scope/.
		var out []string
		for _, l := range lines(capture("npm", "ls", "-g", "--depth=0", "--parseable")) {
			if _, n, ok := strings.Cut(l, "node_modules/"); ok && n != "" && n != "npm" && n != "corepack" {
				out = append(out, n)
			}
		}
		return out
	case "cargo":
		// `cargo install --list` headers look like "eza v0.18.0:"; the indented
		// binary lines under them don't end with ':'.
		var out []string
		for _, l := range lines(capture("cargo", "install", "--list")) {
			if strings.HasSuffix(l, ":") {
				out = append(out, strings.Fields(l)[0])
			}
		}
		return out
	}
	return nil
}

// goInstalled recovers each go-installed binary's main package path via
// `go version -m`, which is what `go install <path>@latest` needs.
func goInstalled() []string {
	gobin := strings.TrimSpace(capture("go", "env", "GOBIN"))
	if gobin == "" {
		if gp := strings.TrimSpace(capture("go", "env", "GOPATH")); gp != "" {
			gobin = filepath.Join(gp, "bin")
		}
	}
	if gobin == "" {
		return nil
	}
	ents, err := os.ReadDir(gobin)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		for _, l := range lines(capture("go", "version", "-m", filepath.Join(gobin, e.Name()))) {
			if f := strings.Fields(l); len(f) >= 2 && f[0] == "path" {
				out = append(out, f[1])
				break
			}
		}
	}
	return out
}

// Result is a single search hit: which manager offers the package, and its name.
type Result struct{ Mgr, Name string }

// Searchable lists the managers whose CLIs expose a usable package search.
// tap/go/uv have no meaningful search, so they're omitted.
var Searchable = []string{"brew", "cask", "npm", "cargo"}

// Search returns package names matching query for a single manager.
func Search(mgr, query string) []string {
	switch mgr {
	case "brew":
		return lines(capture("brew", "search", "--formula", query))
	case "cask":
		return lines(capture("brew", "search", "--cask", query))
	case "npm":
		var out []string
		for _, l := range lines(capture("npm", "search", query, "--parseable")) {
			if name := strings.SplitN(l, "\t", 2)[0]; name != "" {
				out = append(out, name)
			}
		}
		return out
	case "cargo":
		var out []string
		for _, l := range lines(capture("cargo", "search", query)) {
			// format: name = "version"   # description
			if i := strings.Index(l, " = "); i > 0 {
				out = append(out, strings.TrimSpace(l[:i]))
			}
		}
		return out
	}
	return nil
}

// SearchAll runs Search across every Searchable manager in parallel and returns
// the combined hits, grouped in manager order.
func SearchAll(query string) []Result {
	hits := make([][]string, len(Searchable))
	var wg sync.WaitGroup
	for i, mgr := range Searchable {
		wg.Add(1)
		go func(i int, mgr string) {
			defer wg.Done()
			hits[i] = Search(mgr, query)
		}(i, mgr)
	}
	wg.Wait()
	var out []Result
	for i, mgr := range Searchable {
		for _, n := range hits[i] {
			out = append(out, Result{mgr, n})
		}
	}
	return out
}

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}
