// The seven Manager implementations, thin wrappers over each CLI.
package pm

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// ---- brew (formulae) --------------------------------------------------------

type brew struct{}

func (brew) Name() string             { return "brew" }
func (brew) Install(pkg string) error { return run("brew", "install", pkg) }
func (brew) Uninstall(pkg string) error {
	return run("brew", "uninstall", pkg)
}

func (brew) Installed() []string {
	if out := lines(capture("brew", "leaves", "--installed-on-request")); len(out) > 0 {
		return out
	}
	return lines(capture("brew", "leaves"))
}

func (brew) Search(query string) []string {
	return lines(capture("brew", "search", "--formula", query))
}

func (brew) Outdated() []string {
	return lines(capture("brew", "outdated", "--formula", "--quiet"))
}
func (brew) Upgrade(pkg string) error { return run("brew", "upgrade", "--formula", pkg) }

// ---- cask -------------------------------------------------------------------

type cask struct{}

func (cask) Name() string               { return "cask" }
func (cask) Install(pkg string) error   { return run("brew", "install", "--cask", pkg) }
func (cask) Uninstall(pkg string) error { return run("brew", "uninstall", "--cask", pkg) }
func (cask) Installed() []string        { return lines(capture("brew", "list", "--cask")) }
func (cask) Search(query string) []string {
	return lines(capture("brew", "search", "--cask", query))
}

func (cask) Outdated() []string {
	return lines(capture("brew", "outdated", "--cask", "--quiet"))
}
func (cask) Upgrade(pkg string) error { return run("brew", "upgrade", "--cask", pkg) }

// ---- tap --------------------------------------------------------------------

type tap struct{}

func (tap) Name() string               { return "tap" }
func (tap) Install(pkg string) error   { return run("brew", "tap", pkg) }
func (tap) Uninstall(pkg string) error { return run("brew", "untap", pkg) }

func (tap) Installed() []string {
	var out []string
	for _, t := range lines(capture("brew", "tap")) {
		if t != "homebrew/core" && t != "homebrew/cask" && t != "homebrew/bundle" {
			out = append(out, t)
		}
	}
	return out
}

// ---- go ---------------------------------------------------------------------

type golang struct{}

func (golang) Name() string             { return "go" }
func (golang) Install(pkg string) error { return run("go", "install", pkg+"@latest") }

func (golang) Uninstall(pkg string) error {
	gopath := strings.TrimSpace(capture("go", "env", "GOPATH"))
	if gopath == "" {
		return fmt.Errorf("could not resolve GOPATH")
	}
	return os.Remove(filepath.Join(gopath, "bin", filepath.Base(pkg)))
}

// Installed recovers each go-installed binary's main package path via
// `go version -m`, which is what `go install <path>@latest` needs.
func (golang) Installed() []string {
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

// ---- uv ---------------------------------------------------------------------

type uvTool struct{}

func (uvTool) Name() string               { return "uv" }
func (uvTool) Install(pkg string) error   { return run("uv", "tool", "install", pkg) }
func (uvTool) Uninstall(pkg string) error { return run("uv", "tool", "uninstall", pkg) }
func (uvTool) UpgradeAll() error          { return run("uv", "tool", "upgrade", "--all") }

// Installed parses `uv tool list`: "name v1.2.3" headers with "- <exe>"
// bullets under each.
func (uvTool) Installed() []string {
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
}

// ---- npm --------------------------------------------------------------------

type npmPkg struct{}

func (npmPkg) Name() string               { return "npm" }
func (npmPkg) Install(pkg string) error   { return run("npm", "install", "-g", pkg) }
func (npmPkg) Uninstall(pkg string) error { return run("npm", "uninstall", "-g", pkg) }
func (npmPkg) Upgrade(pkg string) error   { return run("npm", "install", "-g", pkg+"@latest") }

// Installed parses the parseable global list: one node_modules path per line;
// scoped names keep their @scope/.
func (npmPkg) Installed() []string {
	var out []string
	for _, l := range lines(capture("npm", "ls", "-g", "--depth=0", "--parseable")) {
		if _, n, ok := strings.Cut(l, "node_modules/"); ok && n != "" && n != "npm" && n != "corepack" {
			out = append(out, n)
		}
	}
	return out
}

func (npmPkg) Search(query string) []string {
	var out []string
	for _, l := range lines(capture("npm", "search", query, "--parseable")) {
		if name, _, _ := strings.Cut(l, "\t"); name != "" {
			out = append(out, name)
		}
	}
	return out
}

// Outdated parses `npm -g outdated --parseable`:
// <path>:<name>@<wanted>:<name>@<current>:<name>@<latest>
func (npmPkg) Outdated() []string {
	var out []string
	for _, l := range lines(capture("npm", "-g", "outdated", "--parseable")) {
		parts := strings.Split(l, ":")
		if len(parts) >= 2 {
			if i := strings.LastIndex(parts[1], "@"); i > 0 {
				out = append(out, parts[1][:i])
			}
		}
	}
	return out
}

// ---- cargo ------------------------------------------------------------------

type cargo struct{}

func (cargo) Name() string               { return "cargo" }
func (cargo) Install(pkg string) error   { return run("cargo", "install", pkg) }
func (cargo) Uninstall(pkg string) error { return run("cargo", "uninstall", pkg) }

func (cargo) UpgradeAll() error {
	if _, err := exec.LookPath("cargo-install-update"); err != nil {
		return fmt.Errorf("install cargo-update first: cargo install cargo-update")
	}
	return run("cargo", "install-update", "-a")
}

// Installed parses `cargo install --list` headers: `eza v0.18.0:` (the
// indented binary lines under them don't end with ':').
func (cargo) Installed() []string {
	var out []string
	for _, l := range lines(capture("cargo", "install", "--list")) {
		if strings.HasSuffix(l, ":") {
			out = append(out, strings.Fields(l)[0])
		}
	}
	return out
}

// Search parses `cargo search`: name = "version"   # description
func (cargo) Search(query string) []string {
	var out []string
	for _, l := range lines(capture("cargo", "search", query)) {
		if i := strings.Index(l, " = "); i > 0 {
			out = append(out, strings.TrimSpace(l[:i]))
		}
	}
	return out
}
