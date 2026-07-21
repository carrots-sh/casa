// Homebrew taps.
package pm

type tap struct{}

func (tap) Name() string               { return "tap" }
func (tap) Install(pkg string) error   { return run("brew", "tap", pkg) }
func (tap) Uninstall(pkg string) error { return run("brew", "untap", pkg) }

func (tap) Installed() []string {
	var out []string
	for _, t := range lines(capture("brew", "tap")) {
		if t != "homebrew/core" && t != "homebrew/cask" && t != "homebrew/bundle" {
			out = append(out, t)
		}
	}
	return out
}
