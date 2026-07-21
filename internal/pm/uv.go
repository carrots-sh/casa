// uv tools.
package pm

import (
	"strings"
)

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
