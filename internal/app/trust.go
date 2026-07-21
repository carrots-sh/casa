// Trusted taps: which third-party taps brew bundle may manage unprompted.
package app

import (
	"fmt"
	"slices"
	"sort"
	"strings"

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
	all := append(append([]string{}, plain...), trusted...)
	if len(all) == 0 {
		fmt.Println("no taps recorded yet")
		return nil
	}
	sort.Strings(all)
	want, err := ui.MultiSelect("which taps are trusted? (their formulae update without prompting)", all, trusted...)
	if err != nil {
		return err
	}
	wantSet := map[string]bool{}
	for _, t := range want {
		wantSet[t] = true
	}
	changed := 0
	for _, t := range all {
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
	if changed == 0 {
		fmt.Println("nothing to change.")
		return nil
	}
	fmt.Printf("✓ trusted taps: %s\n", strings.Join(want, ", "))
	offerSave("casa: update trusted taps")
	return nil
}

// UpdateTools lists outdated packages and upgrades the chosen ones. sh tools
