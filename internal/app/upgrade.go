package app

import (
	"fmt"
	"strings"

	"github.com/carrots-sh/casa/internal/selfupdate"
	"github.com/carrots-sh/casa/internal/ui"
)

// Version is the running build's version, set by main at startup.
var Version = "dev"

// UpgradeSelf replaces the running casa binary with the newest GitHub release.
func UpgradeSelf() error {
	fmt.Println("checking github releases...")
	latest, err := selfupdate.Latest()
	if err != nil {
		return fmt.Errorf("couldn't check releases: %w", err)
	}
	cur := strings.TrimPrefix(Version, "v")
	switch {
	case cur == strings.TrimPrefix(latest, "v"):
		fmt.Printf("✓ already the newest release (%s)\n", latest)
		return nil
	case !selfupdate.Newer(Version, latest):
		// a dev/source build, or somehow ahead of the latest release —
		// replacing it would be a downgrade, so ask first
		ok, err := ui.Confirm(fmt.Sprintf(
			"you're on %s (not an older release) — replace it with %s?", Version, latest))
		if err != nil || !ok {
			return err
		}
	}
	fmt.Printf("upgrading casa %s → %s ...\n", Version, latest)
	path, err := selfupdate.Upgrade(latest)
	if err != nil {
		return err
	}
	fmt.Printf("✓ upgraded to %s  (%s)\n", latest, path)
	return nil
}
