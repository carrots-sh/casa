// Package home is the one place that knows about ~ — expanding it for the
// filesystem and folding it back in for display.
package home

import (
	"os"
	"path/filepath"
	"strings"
)

// Dir returns the user's home directory ("" if unknown).
func Dir() string {
	h, _ := os.UserHomeDir()
	return h
}

// Expand turns ~ and ~/x into absolute paths. A plain prefix swap — trailing
// slashes and . segments survive, unlike filepath.Join. Everything else
// passes through.
func Expand(p string) string {
	h := Dir()
	if h == "" {
		return p
	}
	if p == "~" {
		return h
	}
	if strings.HasPrefix(p, "~/") {
		return h + p[1:]
	}
	return p
}

// Tilde renders a path for display as ~/…: home-relative paths get the
// prefix, absolute paths under home get shortened, anything else is left
// alone.
func Tilde(p string) string {
	h := Dir()
	if h != "" && strings.HasPrefix(p, h+string(os.PathSeparator)) {
		return "~" + p[len(h):]
	}
	if p == "" || filepath.IsAbs(p) || strings.HasPrefix(p, "~") {
		return p
	}
	return "~/" + p
}

// Path joins a home-relative path onto the home directory.
func Path(rel string) string {
	return filepath.Join(Dir(), rel)
}
