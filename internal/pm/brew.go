// brew formulae.
package pm

type brew struct{}

func (brew) Name() string             { return "brew" }
func (brew) Install(pkg string) error { return run("brew", "install", pkg) }
func (brew) Uninstall(pkg string) error {
	return run("brew", "uninstall", pkg)
}

func (brew) Installed() []string {
	if out := lines(capture("brew", "leaves", "--installed-on-request")); len(out) > 0 {
		return out
	}
	return lines(capture("brew", "leaves"))
}

func (brew) Search(query string) []string {
	return lines(capture("brew", "search", "--formula", query))
}

func (brew) Outdated() []string {
	return lines(capture("brew", "outdated", "--formula", "--quiet"))
}
func (brew) Upgrade(pkg string) error { return run("brew", "upgrade", "--formula", pkg) }
