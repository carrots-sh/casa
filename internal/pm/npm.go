// npm global packages.
package pm

import (
	"os/exec"
	"strings"
)

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

// Search: bun installs from the same registry and is the preferred
// installer, so npm's search stays silent when bun is present — the hits
// show up as bun rows instead (no duplicates, safer default installs).
func (npmPkg) Search(query string) []string {
	if _, err := exec.LookPath("bun"); err == nil {
		return nil
	}
	return npmRegistrySearch(query)
}

// npmRegistrySearch queries the npm registry via the npm CLI.
func npmRegistrySearch(query string) []string {
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
