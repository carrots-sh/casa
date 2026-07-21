// cargo installs.
package pm

import (
	"fmt"
	"os/exec"
	"strings"
)

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
