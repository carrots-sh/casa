// Drift review: files whose on-disk state differs from the repo — view each
// diff, then keep the local version (record it) or restore the repo's.
package app

import (
	"fmt"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/home"
	"github.com/carrots-sh/casa/internal/ui"
)

// driftedTargets parses `chezmoi status` into home-relative target paths.
// Pending run scripts ('R' status) are not file drift — they re-run on the
// next apply and have nothing to keep or restore.
func driftedTargets() ([]string, error) {
	lines, err := chez.Status()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range lines {
		if len(l) > 3 && l[0] != 'R' && l[1] != 'R' {
			out = append(out, l[3:])
		}
	}
	return out, nil
}

// Drift walks the drifted files one at a time: diff first, then choose.
func Drift() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	for {
		targets, err := driftedTargets()
		if err != nil {
			return err
		}
		if len(targets) == 0 {
			fmt.Println("✓ nothing drifted — this machine matches the repo")
			return nil
		}
		labels := make([]string, len(targets))
		byLabel := map[string]string{}
		for i, t := range targets {
			labels[i] = home.Tilde(t)
			byLabel[labels[i]] = t
		}
		sel, err := ui.Select("review which drifted file?", labels)
		if err != nil || sel == "" {
			return err
		}
		if err := resolveDrift(byLabel[sel], sel); err != nil {
			return err
		}
	}
}

// resolveDrift shows the diff for one target (colored, straight to the
// terminal — no line-cursor pager), then applies the chosen side.
func resolveDrift(target, display string) error {
	fmt.Printf("── %s ──  (- lines = your local change)\n\n", display)
	if err := chez.DiffShow(home.Path(target)); err != nil {
		return err
	}
	fmt.Println()
	const (
		keep    = "keep my local version · record it in the repo"
		restore = "restore the repo version · overwrite my local change"
		skip    = "skip · decide later"
	)
	sel, err := ui.Select(display, []string{keep, restore, skip})
	if err != nil || sel == "" || sel == skip {
		return err
	}
	switch sel {
	case keep:
		if err := chez.Add(home.Path(target)); err != nil {
			return err
		}
		invalidateStatus()
		fmt.Printf("✓ recorded your local %s\n", display)
		offerSave("casa: update " + display)
	case restore:
		if err := chez.ApplyForce(home.Path(target)); err != nil {
			return err
		}
		invalidateStatus()
		fmt.Printf("✓ restored %s from the repo\n", display)
	}
	return nil
}
