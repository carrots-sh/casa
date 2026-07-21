package app

import (
	"fmt"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

// item is one leaf action in the flat menu: what you see, what it runs.
type item struct {
	name, desc, hint string
	run              func() error
}

// section is a visual cluster: the name renders once, as a gutter label on the
// cluster's first row. Still one flat pick — nothing to select twice.
type section struct {
	name  string
	items []item
}

// Menu is casa's interactive home: one flat, filterable list of every action,
// visually clustered by what the commands act on.
func Menu() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	if !chez.HasRepo() {
		fmt.Println("looks like a fresh machine — let's set up your dotfiles.")
		return Setup("")
	}
	for {
		s := computeStatus()
		sections := []section{
			{"configs", []item{
				{"edit", "pick + edit a config", "", func() error { return EditConfig("") }},
				{"track", "start managing a file", "", func() error { return TrackFile("") }},
				{"untrack", "stop managing a file", "", func() error { return UntrackFile("") }},
				{"storage", "change how a file is stored", "", func() error { return ChangeStorage("") }},
				{"list configs", "list managed files", "", func() error { return ListConfigs() }},
			}},
			{"tools", []item{
				{"add", "install a tool", "", func() error { return AddTool("", "") }},
				{"update", "upgrade outdated tools", hint(s.updates, "updates"), func() error { return UpdateTools() }},
				{"remove", "uninstall tools", "", func() error { return RemoveTools() }},
				{"import", "record what's installed here", hint(s.unrecorded, "to record"), func() error { return ImportTools() }},
				{"trust", "pick which taps are trusted", "", func() error { return TrustTaps() }},
				{"list tools", "list recorded tools", "", func() error { return ListTools() }},
			}},
			{"secrets", []item{
				{"secret", "edit an encrypted file", "", func() error { return EditSecret("") }},
				{"encrypt", "add an encrypted file", "", func() error { return AddSecret("") }},
				{"list secrets", "list encrypted files", "", func() error { return ListSecrets() }},
			}},
			{"machine", []item{
				{"save", "publish your changes", hint(s.toSave, "to save"), func() error { return Save("") }},
				{"sync", "update this machine", hint(s.behind, "behind"), func() error { return Sync() }},
				{"status", "full overview", "", func() error { return Status() }},
				{"answers", "change your setup answers", "", func() error { return Answers("") }},
				{"question", "add a setup question", "", func() error { return AddQuestion() }},
				{"undo", "revert the last save", "", func() error { return Undo() }},
				{"setup", "provision from a dotfiles repo", "", func() error { return Setup("") }},
				{"doctor", "health check", "", func() error { return Doctor() }},
				{"info", "machine + repo basics", "", func() error { return Info() }},
			}},
			{"casa", []item{
				{"upgrade", "update casa itself", upgradeHint(s.upgrade), func() error { return UpgradeSelf() }},
				{"quit", "", "", nil},
			}},
		}

		var labels []string
		run := map[string]func() error{}
		byName := map[string]string{}
		for _, sec := range sections {
			for i, it := range sec.items {
				gutter := ""
				if i == 0 {
					gutter = sec.name
				}
				label := fmt.Sprintf("%-8s  %-12s", gutter, it.name)
				if it.desc != "" {
					label += " · " + it.desc
				}
				label += it.hint
				labels = append(labels, label)
				run[label] = it.run
				byName[it.name] = label
			}
		}

		// urgency: the cursor starts on the most pressing action (order intact).
		def := ""
		switch {
		case s.toSave > 0:
			def = byName["save"]
		case s.behind > 0:
			def = byName["sync"]
		case s.upgrade != "":
			def = byName["upgrade"]
		}

		clearScreen()
		choice, err := ui.SelectDefault("casa · "+s.machine, labels, def)
		if err != nil || choice == "" || choice == byName["quit"] {
			return err
		}
		if action := run[choice]; action != nil {
			clearScreen()
			report(action())
			pause()
		}
	}
}

// clearScreen wipes the visible screen so each menu/action starts on a clean
// page instead of piling frames up. Scrollback from before casa started is
// left alone.
func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

// pause holds the action's output on screen until the user is done reading.
func pause() {
	fmt.Print("\n  enter to go back ")
	var s string
	_, _ = fmt.Scanln(&s)
}

func hint(n int, unit string) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("   (%d %s)", n, unit)
}

func upgradeHint(tag string) string {
	if tag == "" {
		return ""
	}
	return "   (" + tag + " is out)"
}

func report(err error) {
	if err != nil {
		fmt.Println("✗", err)
	}
}
