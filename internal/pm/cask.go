// Homebrew casks (macOS apps + fonts).
package pm

type cask struct{}

func (cask) Name() string               { return "cask" }
func (cask) Install(pkg string) error   { return run("brew", "install", "--cask", pkg) }
func (cask) Uninstall(pkg string) error { return run("brew", "uninstall", "--cask", pkg) }
func (cask) Installed() []string        { return lines(capture("brew", "list", "--cask")) }
func (cask) Search(query string) []string {
	return lines(capture("brew", "search", "--cask", query))
}

func (cask) Outdated() []string {
	return lines(capture("brew", "outdated", "--cask", "--quiet"))
}
func (cask) Upgrade(pkg string) error { return run("brew", "upgrade", "--cask", pkg) }
