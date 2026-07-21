package app

import (
	"os"
	"path/filepath"
	"slices"
	"testing"

	"github.com/carrots-sh/casa/internal/manifest"
)

func TestRecordedSectionsIncludesExtras(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "packages.toml")
	toml := `[packages]
brew = ["jq"]
brew_darwin = ["mas"]
extra = ['brew "ruby", link: false', 'tap "sst/tap", "https://x.git"', 'brew "anomalyco/tap/opencode", trusted: true']
extra_darwin = ['uv "mlx-lm"']
`
	if err := os.WriteFile(p, []byte(toml), 0o644); err != nil {
		t.Fatal(err)
	}
	m := manifest.Manifest{Path: p}

	brew := recordedSections(m, "brew")
	for _, want := range []string{"jq", "mas", "ruby", "anomalyco/tap/opencode", "opencode"} {
		if !slices.Contains(brew, want) {
			t.Errorf("brew recorded missing %q (got %v)", want, brew)
		}
	}
	if !slices.Contains(recordedSections(m, "taps"), "sst/tap") {
		t.Errorf("tap extra not recorded: %v", recordedSections(m, "taps"))
	}
	if !slices.Contains(recordedSections(m, "uv"), "mlx-lm") {
		t.Errorf("uv extra_darwin not recorded: %v", recordedSections(m, "uv"))
	}
}
