// casa with no arguments opens the interactive menu; the subcommands below are
// the same actions for scripting. Dispatch is a plain switch — casa's commands
// take at most two positional args and no flags, so a CLI framework isn't needed.
package main

import (
	"fmt"
	"os"
	"runtime/debug"
	"strings"

	"github.com/carrots-sh/casa/internal/app"
)

// Set via ldflags during a goreleaser build; filled from build info for `go install`.
var (
	version = "dev"
	commit  = "none"
)

const usage = `casa — easy chezmoi: manage your configs and tools from one friendly menu

usage: casa [command]           (no command opens the interactive menu)
shortcuts: casa edit [name] · casa save [msg] · casa sync · casa status

configs   edit [name]           pick and edit a config (encrypted ones handled transparently)
          track [path]          start managing an existing file (plain, template, or encrypted)
          storage [name]        change how a file is stored (template/encrypted/…)
          untrack [path]        stop managing a file (keeps it on disk)
          list                  list managed files
tools     add [manager] [name]  install a package and record it in your brewfile
          rm                    uninstall package(s) — pick across all managers
          update                upgrade outdated packages — one, many, or all
          list                  list recorded packages
secrets   add [path]            encrypt and start managing a file
          edit [name]           pick a secret, decrypt, edit, re-encrypt
          list                  list encrypted files
machine   setup [repo]          provision this machine from your dotfiles repo
          sync                  upgrade packages, then pull + apply dotfiles
          save [message]        commit + push your changes
          status                show what's changed, behind, or outdated
          answers [name]        change this machine's setup answers and re-apply
          question              add a setup question to your repo
          undo                  revert the last saved change and re-apply
          doctor                health check
          info                  machine + repo basics

help, version                   this text / version info
`

func main() {
	fillFromBuildInfo()
	if err := dispatch(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "✗", err)
		os.Exit(1)
	}
}

// arg returns a[i] or "" — every optional positional in one place.
func arg(a []string, i int) string {
	if i < len(a) {
		return a[i]
	}
	return ""
}

func dispatch(args []string) error {
	switch arg(args, 0) {
	case "":
		return app.Menu()
	case "help", "-h", "--help":
		fmt.Print(usage)
		return nil
	case "version", "-v", "--version":
		fmt.Printf("casa %s (%s)\n", version, commit)
		return nil
	case "edit":
		return app.EditConfig(arg(args, 1))
	case "save":
		return app.Save(arg(args, 1))
	case "sync":
		return app.Sync()
	case "status":
		return app.Status()
	case "configs":
		switch arg(args, 1) {
		case "edit":
			return app.EditConfig(arg(args, 2))
		case "track":
			return app.TrackFile(arg(args, 2))
		case "untrack":
			return app.UntrackFile(arg(args, 2))
		case "storage":
			return app.ChangeStorage(arg(args, 2))
		case "list":
			return app.ListConfigs()
		}
	case "tools":
		switch arg(args, 1) {
		case "add":
			return app.AddTool(arg(args, 2), arg(args, 3))
		case "rm":
			return app.RemoveTools()
		case "update":
			return app.UpdateTools()
		case "list":
			return app.ListTools()
		}
	case "secrets":
		switch arg(args, 1) {
		case "add":
			return app.AddSecret(arg(args, 2))
		case "edit":
			return app.EditSecret(arg(args, 2))
		case "list":
			return app.ListSecrets()
		}
	case "machine":
		switch arg(args, 1) {
		case "setup":
			return app.Setup(arg(args, 2))
		case "sync":
			return app.Sync()
		case "save":
			return app.Save(arg(args, 2))
		case "status":
			return app.Status()
		case "answers", "context": // context: the old name for the same screen
			return app.Answers(arg(args, 2))
		case "question":
			return app.AddQuestion()
		case "undo":
			return app.Undo()
		case "doctor":
			return app.Doctor()
		case "info":
			return app.Info()
		}
	}
	fmt.Print(usage)
	return fmt.Errorf("unknown command: casa %s", strings.Join(args, " "))
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
