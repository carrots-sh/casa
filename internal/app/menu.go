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
			// noun clusters, one verb vocabulary: add/edit/remove/list mean the
			// same thing in every cluster — no synonym verbs (track, encrypt,
			// untrack are gone). Frequency order: daily verb first, list second,
			// destructive last.
			{"files", []item{
				{"edit", "pick + edit a file — encrypted handled", "", func() error { return EditConfig("") }, false},
				{"list", "managed files", "", func() error {
					l, err := configLines()
					return page("managed files", l, err)
				}, true},
				{"add", "start managing a file", "", func() error { return TrackFile("") }, false},
				{"storage", "how a file is stored", "", func() error { return ChangeStorage("") }, false},
				{"drift", "review files changed outside casa", hint(s.drift, "drifted"), func() error { return Drift() }, false},
				{"remove", "stop managing a file", "", func() error { return UntrackFile("") }, false},
			}},
			{"tools", []item{
				{"add", "install a tool (search or paste)", "", func() error { return AddTool("", "") }, false},
				{"list", "recorded tools", "", func() error {
					l, err := toolLines()
					return page("recorded tools", l, err)
				}, true},
				{"update", "upgrade outdated tools", hint(s.updates, "updates"), func() error { return UpdateTools() }, false},
				{"import", "record what's installed here", hint(s.unrecorded, "to record"), func() error { return ImportTools() }, false},
				{"trust", "trusted taps", "", func() error { return TrustTaps() }, false},
				{"remove", "uninstall tools", "", func() error { return RemoveTools() }, false},
			}},
			{"secrets", []item{
				{"edit", "edit an encrypted file", "", func() error { return EditSecret("") }, false},
				{"list", "encrypted files", "", func() error {
					l, err := secretLines()
					return page("encrypted files", l, err)
				}, true},
				{"add", "encrypt + manage a file", "", func() error { return AddSecret("") }, false},
				{"keys", "encryption keys", "", func() error { return Keys() }, false},
				{"remove", "stop managing a secret", "", func() error { return RemoveSecret() }, false},
			}},
			{"machine", []item{
				{"save", "publish your changes", hint(s.toSave, "to save"), func() error { return Save("") }, false},
				{"sync", "update this machine", hint(s.behind, "behind"), func() error { return Sync() }, false},
				{"status", "full overview", "", func() error { return Status() }, false},
				{"answers", "your setup answers", "", func() error { return Answers("") }, false},
				{"question", "add a setup question", "", func() error { return AddQuestion() }, false},
				{"undo", "revert the last save", "", func() error { return Undo() }, false},
				{"setup", "provision from a dotfiles repo", "", func() error { return Setup("") }, false},
				{"doctor", "health check", "", func() error { return Doctor() }, false},
				{"info", "machine + repo basics", "", func() error { return Info() }, false},
			}},
			{"casa", []item{
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
