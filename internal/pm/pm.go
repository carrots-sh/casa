// Package pm wraps the supported package managers behind a uniform interface.
package pm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
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

func lines(s string) []string {
	var out []string
	for _, l := range strings.Split(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}
