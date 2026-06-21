// Package config reads casa's optional, committed .casa.toml and falls back to
// sensible auto-detected defaults so casa works against any chezmoi repo.
package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/carrots-sh/casa/internal/chez"
)

type Config struct {
	Pkg struct {
		Brewfile string `toml:"brewfile"` // source template that records packages
		Anchors  string `toml:"anchors"`  // "# casa:<manager>" anchor prefix word
	} `toml:"pkg"`
	Setup struct {
		Repo string `toml:"repo"` // default repo for `casa machine setup`
	} `toml:"setup"`
}

// Load reads .casa.toml from the chezmoi source dir, applying defaults.
func Load() Config {
	var c Config
	if src, err := chez.SourceDir(); err == nil {
		_, _ = toml.DecodeFile(filepath.Join(src, ".casa.toml"), &c)
		if c.Pkg.Brewfile == "" {
			for _, cand := range []string{"dot_Brewfile.tmpl", "dot_Brewfile", "Brewfile.tmpl", "Brewfile"} {
				if _, err := os.Stat(filepath.Join(src, cand)); err == nil {
					c.Pkg.Brewfile = cand
					break
				}
			}
		}
	}
	if c.Pkg.Anchors == "" {
		c.Pkg.Anchors = "casa"
	}
	return c
}

// BrewfileTmpl returns the absolute path to the Brewfile source template, or "".
func (c Config) BrewfileTmpl() string {
	if c.Pkg.Brewfile == "" {
		return ""
	}
	src, err := chez.SourceDir()
	if err != nil {
		return ""
	}
	return filepath.Join(src, c.Pkg.Brewfile)
}
