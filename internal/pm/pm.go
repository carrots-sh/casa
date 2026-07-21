// Package pm drives the supported package managers behind a uniform interface.
package pm

import (
	"os"
	"os/exec"
	"strings"
	"sync"
)

// Manager is one package manager casa can drive. Extra abilities are optional
// interfaces asserted where they're used: Searcher, Outdater, BulkUpgrader.
type Manager interface {
	Name() string
	Install(pkg string) error
	Uninstall(pkg string) error
	Installed() []string // best-effort top-level packages, for `tools import`
}

// Searcher is a Manager whose CLI exposes a usable package search.
type Searcher interface {
	Search(query string) []string
}

// Outdater is a Manager that can report and upgrade individual packages.
type Outdater interface {
	Outdated() []string
	Upgrade(pkg string) error
}

// BulkUpgrader is a Manager that only upgrades everything at once.
type BulkUpgrader interface {
	UpgradeAll() error
}

// Managers is the supported set, in display order.
var Managers = []Manager{brew{}, cask{}, tap{}, golang{}, uvTool{}, npmPkg{}, bunPkg{}, cargo{}}

// ByName returns the manager with the given name.
func ByName(name string) (Manager, bool) {
	for _, m := range Managers {
		if m.Name() == name {
			return m, true
		}
	}
	return nil, false
}

// Names lists every manager name, in display order.
func Names() []string {
	out := make([]string, len(Managers))
	for i, m := range Managers {
		out[i] = m.Name()
	}
	return out
}

// Searchable lists the names of managers that implement Searcher.
func Searchable() []string {
	var out []string
	for _, m := range Managers {
		if _, ok := m.(Searcher); ok {
			out = append(out, m.Name())
		}
	}
	return out
}

// Outdated returns "mgr name" items with updates, from every Outdater.
func Outdated() []string {
	var items []string
	for _, m := range Managers {
		o, ok := m.(Outdater)
		if !ok {
			continue
		}
		for _, n := range o.Outdated() {
			items = append(items, m.Name()+" "+n)
		}
	}
	return items
}

// Result is a single search hit: which manager offers the package, and its name.
type Result struct{ Mgr, Name string }

// SearchAll runs every Searcher in parallel and returns the combined hits,
// grouped in manager order.
func SearchAll(query string) []Result {
	hits := make([][]string, len(Managers))
	var wg sync.WaitGroup
	for i, m := range Managers {
		s, ok := m.(Searcher)
		if !ok {
			continue
		}
		wg.Add(1)
		go func(i int, s Searcher) {
			defer wg.Done()
			hits[i] = s.Search(query)
		}(i, s)
	}
	wg.Wait()
	var out []Result
	for i, m := range Managers {
		for _, n := range hits[i] {
			out = append(out, Result{m.Name(), n})
		}
	}
	return out
}

// UpgradeAllBrew upgrades every outdated formula and cask in a single brew call.
func UpgradeAllBrew() error { return run("brew", "upgrade") }

func run(name string, args ...string) error {
	c := exec.Command(name, args...)
	c.Stdout, c.Stderr, c.Stdin = os.Stdout, os.Stderr, os.Stdin
	return c.Run()
}

func capture(name string, args ...string) string {
	out, _ := exec.Command(name, args...).Output()
	return string(out)
}

func lines(s string) []string {
	var out []string
	for l := range strings.SplitSeq(s, "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out
}
