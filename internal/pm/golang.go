// go install.
package pm

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type golang struct{}

func (golang) Name() string             { return "go" }
func (golang) Install(pkg string) error { return run("go", "install", pkg+"@latest") }

func (golang) Uninstall(pkg string) error {
	gopath := strings.TrimSpace(capture("go", "env", "GOPATH"))
	if gopath == "" {
		return fmt.Errorf("could not resolve GOPATH")
	}
	return os.Remove(filepath.Join(gopath, "bin", filepath.Base(pkg)))
}

// Installed recovers each go-installed binary's main package path via
// `go version -m`, which is what `go install <path>@latest` needs.
func (golang) Installed() []string {
	gobin := strings.TrimSpace(capture("go", "env", "GOBIN"))
	if gobin == "" {
		if gp := strings.TrimSpace(capture("go", "env", "GOPATH")); gp != "" {
			gobin = filepath.Join(gp, "bin")
		}
	}
	if gobin == "" {
		return nil
	}
	ents, err := os.ReadDir(gobin)
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		if e.IsDir() {
			continue
		}
		for _, l := range lines(capture("go", "version", "-m", filepath.Join(gobin, e.Name()))) {
			if f := strings.Fields(l); len(f) >= 2 && f[0] == "path" {
				out = append(out, f[1])
				break
			}
		}
	}
	return out
}
