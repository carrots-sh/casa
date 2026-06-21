package main

import (
	"runtime/debug"

	"github.com/carrots-sh/casa/cmd"
)

// Set via ldflags during a goreleaser build; filled from build info for `go install`.
var (
	version = "dev"
	commit  = "none"
)

func main() {
	fillFromBuildInfo()
	cmd.Execute(version, commit)
}

func fillFromBuildInfo() {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return
	}
	if version == "dev" && info.Main.Version != "" && info.Main.Version != "(devel)" {
		version = info.Main.Version
	}
	if commit == "none" {
		for _, s := range info.Settings {
			if s.Key == "vcs.revision" && s.Value != "" {
				if len(s.Value) > 7 {
					commit = s.Value[:7]
				} else {
					commit = s.Value
				}
			}
		}
	}
}
