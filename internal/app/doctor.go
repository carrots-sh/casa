// Health check: which manager dependencies are present (and where), then
// chezmoi's own doctor.
package app

import (
	"fmt"
	"os/exec"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/home"
)

// dep is one binary casa leans on, with how to get it when missing.
type dep struct{ bin, why, hint string }

// deps lists everything casa shells out to. brew is the keystone: once it's
// there, every other manager is one brew install away (macOS and Linux).
var deps = []dep{
	{"git", "versioning your dotfiles", "xcode-select --install (macOS) / apt install git (Linux)"},
	{"chezmoi", "the engine behind casa", "casa installs it on first run"},
	{"brew", "brew/cask/tap packages", "casa installs it during machine setup (or see brew.sh)"},
	{"age", "secret encryption", "brew install age"},
	{"go", "go installs", "brew install go"},
	{"uv", "uv tools", "brew install uv"},
	{"npm", "npm globals", "brew install node"},
	{"bun", "bun globals", "brew install oven-sh/bun/bun"},
	{"cargo", "cargo installs", "brew install rustup && rustup default stable"},
}

// Doctor reports casa's dependencies, then runs chezmoi's own health check.
// Managers you don't use are fine to leave missing — casa skips them.
func Doctor() error {
	fmt.Println("deps (missing managers are fine — casa skips them):")
	for _, d := range deps {
		if p, err := exec.LookPath(d.bin); err == nil {
			fmt.Printf("  ✓ %-8s %-24s %s\n", d.bin, d.why, home.Tilde(p))
		} else {
			fmt.Printf("  ✗ %-8s %-24s → %s\n", d.bin, d.why, d.hint)
		}
	}
	fmt.Println()
	if err := requireChezmoi(); err != nil {
		return err
	}
	return chez.Doctor()
}
