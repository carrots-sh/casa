// Trusted taps: which third-party taps brew bundle may manage unprompted.
package app

import (
	"fmt"
	"os/exec"
	"slices"
	"sort"
	"strings"

	"github.com/carrots-sh/casa/internal/manifest"
	"github.com/carrots-sh/casa/internal/ui"
)

// TrustTaps picks which taps brew bundle may manage without prompting —
// trusted taps render as `tap "…", trusted: true`, so their formulae stop
// producing "tap formula is not trusted" warnings on install/cleanup.
func TrustTaps() error {
	m := mf()
	if !m.Configured() {
		fmt.Println("no manifest yet — try: casa tools add")
		return nil
	}
	plain, _ := m.List("taps")
	trusted, _ := m.List("taps_trusted")
	extras := m.ExtraTaps() // custom-URL taps live as raw extra lines
	extraByName := map[string]manifest.ExtraTap{}
	all := append(append([]string{}, plain...), trusted...)
	preselect := append([]string{}, trusted...)
	for _, e := range extras {
		extraByName[e.Name] = e
		all = append(all, e.Name)
		if e.Trusted {
			preselect = append(preselect, e.Name)
		}
	}
	if len(all) == 0 {
		fmt.Println("no taps recorded yet")
		return nil
	}
	sort.Strings(all)
	all = slices.Compact(all)
	want, err := ui.MultiSelect("which taps are trusted? (their formulae update without prompting)", all, preselect...)
	if err != nil {
		return err
	}
	wantSet := map[string]bool{}
	for _, t := range want {
		wantSet[t] = true
	}
	changed := 0
	for _, t := range all {
		if e, ok := extraByName[t]; ok {
			if wantSet[t] != e.Trusted {
				if err := m.SetExtraTapTrust(t, wantSet[t]); err != nil {
					return err
				}
				changed++
			}
			continue
		}
		was := slices.Contains(trusted, t)
		switch {
		case wantSet[t] && !was:
			_ = m.Remove("taps", t)
			if err := m.Add("taps_trusted", t); err != nil {
				return err
			}
			changed++
		case !wantSet[t] && was:
			_ = m.Remove("taps_trusted", t)
			if err := m.Add("taps", t); err != nil {
				return err
			}
			changed++
		}
	}
	// brew keeps its own machine-level trust store (~/.homebrew/trust.json)
	// that the Brewfile directive never touches — non-bundle commands (brew
	// upgrade, brew install) warn from it. Converge it to the selection even
	// when the manifest didn't change.
	syncMachineTrust(all, wantSet)
	if changed == 0 {
		fmt.Println("manifest unchanged; this machine's brew trust store synced.")
		return nil
	}
	fmt.Printf("✓ trusted taps: %s\n", strings.Join(want, ", "))
	offerSave("casa: update trusted taps")
	return nil
}

// syncMachineTrust applies brew's per-machine trust to match the selection.
// Older brews without `brew trust` just error, which is ignored.
func syncMachineTrust(all []string, want map[string]bool) {
	if _, err := exec.LookPath("brew"); err != nil {
		return
	}
	for _, t := range all {
		verb := "untrust"
		if want[t] {
			verb = "trust"
		}
		_ = exec.Command("brew", verb, t).Run()
	}
}

// UpdateTools lists outdated packages and upgrades the chosen ones. sh tools
