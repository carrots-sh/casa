package app

import (
	"path/filepath"
	"slices"
	"testing"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/manifest"
	"github.com/carrots-sh/casa/internal/ui"
)

// scriptedUI answers prompts from queues — the Prompter seam in action.
type scriptedUI struct {
	selects []string
	multis  [][]string
}

func (s *scriptedUI) Select(_ string, _ []string, _ string) (string, error) {
	if len(s.selects) == 0 {
		return "", nil
	}
	v := s.selects[0]
	s.selects = s.selects[1:]
	return v, nil
}

func (s *scriptedUI) MultiSelect(_ string, _ []string, _ []string) ([]string, error) {
	if len(s.multis) == 0 {
		return nil, nil
	}
	v := s.multis[0]
	s.multis = s.multis[1:]
	return v, nil
}

func (s *scriptedUI) Input(_, def string) (string, error) { return def, nil }
func (s *scriptedUI) Path(string) (string, error)         { return "", nil }
func (s *scriptedUI) Confirm(_ string, def bool) (bool, error) {
	return false, nil // declines offerSave's push prompt paths
}

func TestTrustTapsMovesSections(t *testing.T) {
	src := t.TempDir()
	chez.SetSource(src)
	mp := filepath.Join(src, manifest.DefaultRel)
	if _, err := manifest.Bootstrap(src, mp); err != nil {
		t.Fatal(err)
	}
	m := manifest.Manifest{Path: mp}
	for _, tap := range []string{"a/tap", "b/tap"} {
		if err := m.Add("taps", tap); err != nil {
			t.Fatal(err)
		}
	}
	if err := m.Add("taps_trusted", "c/tap"); err != nil {
		t.Fatal(err)
	}

	// trust b, untrust c, leave a plain
	restore := ui.SetPrompter(&scriptedUI{multis: [][]string{{"b/tap"}}})
	defer restore()
	if err := TrustTaps(); err != nil {
		t.Fatal(err)
	}

	trusted, _ := m.List("taps_trusted")
	plain, _ := m.List("taps")
	slices.Sort(trusted)
	slices.Sort(plain)
	if !slices.Equal(trusted, []string{"b/tap"}) {
		t.Errorf("trusted = %v", trusted)
	}
	if !slices.Equal(plain, []string{"a/tap", "c/tap"}) {
		t.Errorf("plain = %v", plain)
	}
}
