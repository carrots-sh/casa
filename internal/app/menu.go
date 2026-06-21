package app

import (
	"fmt"
	"strings"

	"github.com/carrots-sh/casa/internal/ui"
)

// Menu is casa's interactive home: a status-aware loop you navigate with arrows.
func Menu() error {
	if err := requireChezmoi(); err != nil {
		return err
	}
	for {
		s := computeStatus()
		opts := []string{
			"Configs  · edit your dotfiles",
			"Tools    · install, remove, update" + hint(s.updates, "updates"),
			"Secrets  · encrypted files",
			"Sync     · pull latest onto this machine" + hint(s.behind, "behind"),
			"Save     · publish your changes" + hint(s.toSave, "to save"),
			"Status   · full overview",
			"Machine  · contexts, info, health",
			"Quit",
		}
		choice, err := ui.Select("casa · "+s.machine, opts)
		if err != nil || choice == "" || strings.HasPrefix(choice, "Quit") {
			return err
		}
		switch {
		case strings.HasPrefix(choice, "Configs"):
			configsMenu()
		case strings.HasPrefix(choice, "Tools"):
			toolsMenu()
		case strings.HasPrefix(choice, "Secrets"):
			secretsMenu()
		case strings.HasPrefix(choice, "Sync"):
			report(Sync())
		case strings.HasPrefix(choice, "Save"):
			report(Save(""))
		case strings.HasPrefix(choice, "Status"):
			report(Status())
		case strings.HasPrefix(choice, "Machine"):
			machineMenu()
		}
	}
}

func toolsMenu() {
	for {
		c, err := ui.Select("Tools", []string{"Add a tool", "Remove tool(s)", "Update outdated", "List installed", "← Back"})
		if err != nil || c == "" || c == "← Back" {
			return
		}
		switch c {
		case "Add a tool":
			report(AddTool("", ""))
		case "Remove tool(s)":
			report(RemoveTools())
		case "Update outdated":
			report(UpdateTools())
		case "List installed":
			report(ListTools())
		}
	}
}

func configsMenu() {
	for {
		c, err := ui.Select("Configs", []string{"Edit a config", "Track a new file", "Untrack a file", "List managed", "← Back"})
		if err != nil || c == "" || c == "← Back" {
			return
		}
		switch c {
		case "Edit a config":
			report(EditConfig(""))
		case "Track a new file":
			report(TrackFile(""))
		case "Untrack a file":
			report(UntrackFile(""))
		case "List managed":
			report(ListConfigs())
		}
	}
}

func secretsMenu() {
	for {
		c, err := ui.Select("Secrets", []string{"Edit a secret", "Add an encrypted file", "List secrets", "← Back"})
		if err != nil || c == "" || c == "← Back" {
			return
		}
		switch c {
		case "Edit a secret":
			report(EditSecret())
		case "Add an encrypted file":
			report(AddSecret(""))
		case "List secrets":
			report(ListSecrets())
		}
	}
}

func machineMenu() {
	for {
		c, err := ui.Select("Machine", []string{"Set up this machine", "Change contexts", "Info", "Health check", "← Back"})
		if err != nil || c == "" || c == "← Back" {
			return
		}
		switch c {
		case "Set up this machine":
			report(Setup(""))
		case "Change contexts":
			report(Context())
		case "Info":
			report(Info())
		case "Health check":
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
