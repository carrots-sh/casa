// bun global packages.
package pm

import (
	"regexp"
	"strings"
)

type bunPkg struct{}

func (bunPkg) Name() string               { return "bun" }
func (bunPkg) Install(pkg string) error   { return run("bun", "add", "-g", pkg) }
func (bunPkg) Uninstall(pkg string) error { return run("bun", "remove", "-g", pkg) }
func (bunPkg) UpgradeAll() error          { return run("bun", "update", "-g") }

func (bunPkg) Installed() []string {
	return parseBunList(capture("bun", "pm", "ls", "-g"))
}

var ansiSeq = regexp.MustCompile(`\x1b\[[0-9;]*m`)

// parseBunList reads `bun pm ls -g` tree rows ("└── cowsay@1.6.0"), which bun
// colors even when piped, so ANSI codes are stripped first.
func parseBunList(s string) []string {
	var out []string
	for _, l := range lines(ansiSeq.ReplaceAllString(s, "")) {
		_, entry, ok := strings.Cut(l, "── ")
		if !ok {
			continue
		}
		if i := strings.LastIndex(entry, "@"); i > 0 {
			out = append(out, strings.TrimSpace(entry[:i]))
		}
	}
	return out
}
