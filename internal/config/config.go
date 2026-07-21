// Package config reads casa's optional, committed .casa.toml and falls back to
// sensible defaults so casa works against any chezmoi repo.
package config

import (
	"path/filepath"

	"github.com/BurntSushi/toml"
	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/manifest"
)

type Config struct {
	Pkg struct {
		Manifest string `toml:"manifest"` // source-relative path of the package manifest
	} `toml:"pkg"`
	Setup struct {
		Repo string `toml:"repo"` // default repo for `casa machine setup`
	} `toml:"setup"`
}

// Load reads .casa.toml from the chezmoi source dir, applying defaults.
func Load() Config {
	var c Config
	src := chez.SourceDir()
	_, _ = toml.DecodeFile(filepath.Join(src, ".casa.toml"), &c)
	if c.Pkg.Manifest == "" {
		c.Pkg.Manifest = manifest.DefaultRel
	}
	return c
}

// ManifestPath returns the absolute path of the package manifest.
func (c Config) ManifestPath() string {
	return filepath.Join(chez.SourceDir(), c.Pkg.Manifest)
}
