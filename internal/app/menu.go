package app

import (
	"fmt"
	"os"

	"github.com/charmbracelet/x/term"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

// item is one leaf action in the flat menu: what you see, what it runs.
// paged actions present their output inside the TUI (scroll + filter) and
// return to the menu on their own — no "enter to go back" pause.
type item struct {
	name, desc, hint string
	run              func() error
	paged            bool
}

// page shows lines inside a scrollable, filterable list — the same widget as
// every picker, so the controls match; enter or esc returns to the menu.
func page(title string, lines []string, err error) error {
	if err != nil {
		return err
	}
	if len(lines) == 0 {
		lines = []string{"(nothing yet)"}
	}
	_, perr := ui.Select(title, lines)
	return perr
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
			// action-first: the top-level question is "what do I want to do".
			// edit and list are type-smart (encrypted files route automatically;
			// list shows everything), so no action needs a second type pick.
			{"act", []item{
				{"edit", "any managed file — encrypted handled", "", func() error { return EditConfig("") }, false},
				{"add", "install a tool (search or paste)", "", func() error { return AddTool("", "") }, false},
				{"track", "start managing a file", "", func() error { return TrackFile("") }, false},
				{"encrypt", "add an encrypted file", "", func() error { return AddSecret("") }, false},
				{"save", "publish your changes", hint(s.toSave, "to save"), func() error { return Save("") }, false},
				{"sync", "update this machine", hint(s.behind, "behind"), func() error { return Sync() }, false},
			}},
			{"see", []item{
				{"list", "everything — files, tools, secrets", "", func() error {
					files, err := configLines()
					if err != nil {
						return err
					}
					tools, _ := toolLines()
					return page("everything managed", append(files, tools...), nil)
				}, true},
				{"status", "full overview", "", func() error { return Status() }, false},
				{"info", "machine + repo basics", "", func() error { return Info() }, false},
			}},
			{"change", []item{
				{"update", "upgrade outdated tools", hint(s.updates, "updates"), func() error { return UpdateTools() }, false},
				{"import", "record what's installed here", hint(s.unrecorded, "to record"), func() error { return ImportTools() }, false},
				{"storage", "how a file is stored", "", func() error { return ChangeStorage("") }, false},
				{"answers", "your setup answers", "", func() error { return Answers("") }, false},
				{"keys", "encryption keys", "", func() error { return Keys() }, false},
				{"trust", "trusted taps", "", func() error { return TrustTaps() }, false},
				{"question", "add a setup question", "", func() error { return AddQuestion() }, false},
			}},
			{"undo", []item{
				{"untrack", "stop managing a file", "", func() error { return UntrackFile("") }, false},
				{"remove", "uninstall tools", "", func() error { return RemoveTools() }, false},
				{"undo", "revert the last save", "", func() error { return Undo() }, false},
			}},
			{"casa", []item{
				{"setup", "provision from a dotfiles repo", "", func() error { return Setup("") }, false},
				{"doctor", "health check", "", func() error { return Doctor() }, false},
				{"upgrade", "update casa itself", upgradeHint(s.upgrade), func() error { return UpgradeSelf() }, false},
				{"quit", "", "", nil, false},
			}},
		}

		var labels []string
		run := map[string]func() error{}
		paged := map[string]bool{}
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
				paged[label] = it.paged
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
			err := action()
			report(err)
			if !paged[choice] || err != nil {
				pause()
			}
		}
	}
}

// clearScreen wipes the visible screen so each menu/action starts on a clean
// page instead of piling frames up. Scrollback from before casa started is
// left alone. It also resets terminal state the TUI stack may leave behind —
// a dangling synchronized-output mode or scroll region makes some terminals
// repaint scrolled output as duplicated chunks.
func clearScreen() {
	fmt.Print("\033[?2026l" + // end synchronized output, if stuck
		"\033[r" + // reset any scroll region to full screen
		"\033[H\033[2J")
}

// pause holds the action's output on screen until the user is done reading.
// Raw mode, no echo: only enter continues (ctrl+c still quits casa); every
// other key is swallowed silently.
func pause() {
	fmt.Print("\n  enter to go back ")
	fd := os.Stdin.Fd()
	old, err := term.MakeRaw(fd)
	if err != nil { // not a terminal — fall back to a plain line read
		var s string
		_, _ = fmt.Scanln(&s)
		return
	}
	defer term.Restore(fd, old) //nolint:errcheck
	buf := make([]byte, 1)
	for {
		if n, err := os.Stdin.Read(buf); err != nil || n == 0 {
			return
		}
		switch buf[0] {
		case '\r', '\n':
			return
		case 3: // ctrl+c
			_ = term.Restore(fd, old)
			os.Exit(0)
		}
	}
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
