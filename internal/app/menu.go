package app

import (
	"fmt"
	"strings"

	"github.com/carrots-sh/casa/internal/chez"
	"github.com/carrots-sh/casa/internal/ui"
)

// Menu is casa's interactive home: a status-aware loop you navigate with arrows.
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
		opts := []string{
			"configs  · edit your dotfiles",
			"tools    · install, remove, update" + hint(s.updates, "updates"),
			"secrets  · encrypted files",
			"sync     · pull latest onto this machine" + hint(s.behind, "behind"),
			"save     · publish your changes" + hint(s.toSave, "to save"),
			"status   · full overview",
			"machine  · contexts, info, health",
			"quit",
		}
		choice, err := ui.Select("casa · "+s.machine, opts)
		if err != nil || choice == "" || strings.HasPrefix(choice, "quit") {
			return err
		}
		switch {
		case strings.HasPrefix(choice, "configs"):
			configsMenu()
		case strings.HasPrefix(choice, "tools"):
			toolsMenu()
		case strings.HasPrefix(choice, "secrets"):
			secretsMenu()
		case strings.HasPrefix(choice, "sync"):
			report(Sync())
		case strings.HasPrefix(choice, "save"):
			report(Save(""))
		case strings.HasPrefix(choice, "status"):
			report(Status())
		case strings.HasPrefix(choice, "machine"):
			machineMenu()
		}
	}
}

func toolsMenu() {
	for {
		c, err := ui.Select("tools", []string{"add a tool", "remove tool(s)", "update outdated", "list installed", "← back"})
		if err != nil || c == "" || c == "← back" {
			return
		}
		switch c {
		case "add a tool":
			report(AddTool("", ""))
		case "remove tool(s)":
			report(RemoveTools())
		case "update outdated":
			report(UpdateTools())
		case "list installed":
			report(ListTools())
		}
	}
}

func configsMenu() {
	for {
		c, err := ui.Select("configs", []string{"edit a config", "track a new file", "untrack a file", "list managed", "← back"})
		if err != nil || c == "" || c == "← back" {
			return
		}
		switch c {
		case "edit a config":
			report(EditConfig(""))
		case "track a new file":
			report(TrackFile(""))
		case "untrack a file":
			report(UntrackFile(""))
		case "list managed":
			report(ListConfigs())
		}
	}
}

func secretsMenu() {
	for {
		c, err := ui.Select("secrets", []string{"edit a secret", "add an encrypted file", "list secrets", "← back"})
		if err != nil || c == "" || c == "← back" {
			return
		}
		switch c {
		case "edit a secret":
			report(EditSecret())
		case "add an encrypted file":
			report(AddSecret(""))
		case "list secrets":
			report(ListSecrets())
		}
	}
}

func machineMenu() {
	for {
		c, err := ui.Select("machine", []string{"set up this machine", "change contexts", "undo last change", "info", "health check", "← back"})
		if err != nil || c == "" || c == "← back" {
			return
		}
		switch c {
		case "set up this machine":
			report(Setup(""))
		case "change contexts":
			report(Context())
		case "undo last change":
			report(Undo())
		case "info":
			report(Info())
		case "health check":
			report(Doctor())
		}
	}
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
