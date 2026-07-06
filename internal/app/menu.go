package app

import (
	"fmt"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

// entry is one leaf action in the flat menu: what you see, what it runs.
type entry struct {
	label string
	run   func() error
}

// Menu is casa's interactive home: one flat, filterable list of every action.
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
		entries := []entry{
			{row("edit", "pick + edit a config"), func() error { return EditConfig("") }},
			{row("save", "publish your changes") + hint(s.toSave, "to save"), func() error { return Save("") }},
			{row("sync", "update this machine") + hint(s.behind, "behind"), func() error { return Sync() }},
			{row("add", "install a tool"), func() error { return AddTool("", "") }},
			{row("update", "upgrade outdated tools") + hint(s.updates, "updates"), func() error { return UpdateTools() }},
			{row("track", "start managing a file"), func() error { return TrackFile("") }},
			{row("secret", "edit an encrypted file"), func() error { return EditSecret("") }},
			{row("encrypt", "add an encrypted file"), func() error { return AddSecret("") }},
			{row("status", "full overview"), func() error { return Status() }},
			{row("remove", "uninstall tools"), func() error { return RemoveTools() }},
			{row("untrack", "stop managing a file"), func() error { return UntrackFile("") }},
			{row("list configs", "list managed files"), func() error { return ListConfigs() }},
			{row("list tools", "list installed tools"), func() error { return ListTools() }},
			{row("list secrets", "list encrypted files"), func() error { return ListSecrets() }},
			{row("contexts", "toggle machine contexts"), func() error { return Context() }},
			{row("undo", "revert the last save"), func() error { return Undo() }},
			{row("setup", "provision from a dotfiles repo"), func() error { return Setup("") }},
			{row("doctor", "health check"), func() error { return Doctor() }},
			{row("info", "machine + repo basics"), func() error { return Info() }},
		}
		// urgency bubbling: sync jumps to the front when behind, save above it when unsaved.
		if s.behind > 0 {
			entries = front(entries, row("sync", "update this machine"))
		}
		if s.toSave > 0 {
			entries = front(entries, row("save", "publish your changes"))
		}

		labels := make([]string, 0, len(entries)+1)
		run := make(map[string]func() error, len(entries))
		for _, e := range entries {
			labels = append(labels, e.label)
			run[e.label] = e.run
		}
		labels = append(labels, "quit")

		choice, err := ui.Select("casa · "+s.machine, labels)
		if err != nil || choice == "" || choice == "quit" {
			return err
		}
		if action := run[choice]; action != nil {
			report(action())
		}
	}
}

// row keeps the aligned 'name · description' aesthetic.
func row(name, desc string) string {
	return fmt.Sprintf("%-8s · %s", name, desc)
}

// front moves the first entry whose label starts with prefix to the top.
func front(entries []entry, prefix string) []entry {
	for i, e := range entries {
		if strings.HasPrefix(e.label, prefix) {
			entries = append(entries[:i:i], entries[i+1:]...)
			return append([]entry{e}, entries...)
		}
	}
	return entries
}

func hint(n int, unit string) string {
	if n <= 0 {
		return ""
	}
	return fmt.Sprintf("   (%d %s)", n, unit)
}

func report(err error) {
	if err != nil {
		fmt.Println("✗", err)
	}
}
