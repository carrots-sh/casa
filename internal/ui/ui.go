// Package ui wraps charm's huh forms for casa's interactive prompts.
package ui

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"

	"github.com/carrots-sh/casa/internal/home"
)

// errCancel is the sentinel run reports when the user presses esc.
var errCancel = errors.New("cancelled")

// Prompter asks the user things. The default renders huh forms in the
// terminal; tests swap in a scripted implementation via SetPrompter. Backed
// out / cancelled prompts return zero values with a nil error.
type Prompter interface {
	Select(title string, opts []string, def string) (string, error)
	MultiSelect(title string, opts []string, preselected []string) ([]string, error)
	Input(title, def string) (string, error)
	Path(title string) (string, error)
	Confirm(title string, def bool) (bool, error)
}

var active Prompter = huhPrompter{}

// SetPrompter replaces the prompter (tests); returns a restore func.
func SetPrompter(p Prompter) func() {
	old := active
	active = p
	return func() { active = old }
}

// huhPrompter renders real terminal forms.
type huhPrompter struct{}

// run executes a form. Esc (and, on list fields, ←) cancels the current
// prompt (a soft back: callers treat the returned zero value as cancel);
// ctrl+c quits casa entirely from anywhere. All abort the form the same way
// in huh, so they share the Quit binding and a message filter remembers
// whether it was a soft back or a hard quit.
//
// Controls are kept consistent across every prompt: tab selects (toggles a
// row, accepts a completion), enter submits, esc/← goes back.
func run(f *huh.Form, list bool) error {
	back := false
	km := keymap(list)
	err := f.WithKeyMap(km).
		WithProgramOptions(tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if k, ok := msg.(tea.KeyPressMsg); ok {
				s := k.String()
				back = s == "esc" || s == "left"
			}
			return msg
		})).
		Run()
	if errors.Is(err, huh.ErrUserAborted) {
		if back {
			return errCancel
		}
		os.Exit(0)
	}
	return err
}

// keymap is huh's default keymap bent to casa's consistent scheme. list forms
// (select/multiselect) also take ← as back; text forms keep ← for the cursor.
func keymap(list bool) *huh.KeyMap {
	km := huh.NewDefaultKeyMap()
	quitKeys := []string{"ctrl+c", "esc"}
	if list {
		quitKeys = append(quitKeys, "left")
	}
	km.Quit = key.NewBinding(key.WithKeys(quitKeys...))

	// single-field forms: shift+tab "back" (previous field) is meaningless noise
	km.Select.Prev.SetEnabled(false)
	km.MultiSelect.Prev.SetEnabled(false)
	km.Input.Prev.SetEnabled(false)
	km.Confirm.Prev.SetEnabled(false)

	// select: enter picks the highlighted row; help carrier for esc/← back
	km.Select.Next = key.NewBinding(key.WithKeys("enter", "tab"), key.WithHelp("enter", "select"))
	km.Select.Submit = key.NewBinding(key.WithKeys("enter"), key.WithHelp("esc/←", "back"))

	// multiselect: tab toggles (space/x still work), enter submits
	km.MultiSelect.Toggle = key.NewBinding(key.WithKeys("tab", "space", "x"), key.WithHelp("tab", "select"))
	km.MultiSelect.Next = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit"))
	km.MultiSelect.Submit = key.NewBinding(key.WithKeys("enter"), key.WithHelp("esc/←", "back"))

	// input: free tab for completion (huh's default tab=next shadows it)
	km.Input.Next = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit"))
	km.Input.AcceptSuggestion = key.NewBinding(key.WithKeys("tab", "right", "ctrl+e"), key.WithHelp("tab", "complete"))

	// confirm: tab toggles like in lists
	km.Confirm.Toggle = key.NewBinding(key.WithKeys("tab", "h", "l", "right", "left"), key.WithHelp("tab/←/→", "toggle"))
	km.Confirm.Next = key.NewBinding(key.WithKeys("enter"), key.WithHelp("enter", "submit"))
	return km
}

// theme keeps charm's accent for the selected item but renders every other
// option in the terminal's own foreground, so it stays readable on any
// background (huh's auto color picks wash out in light terminals).
func theme() huh.Theme {
	return huh.ThemeFunc(func(isDark bool) *huh.Styles {
		s := huh.ThemeCharm(isDark)
		s.Focused.Option = s.Focused.Option.UnsetForeground()
		s.Focused.UnselectedOption = s.Focused.UnselectedOption.UnsetForeground()
		s.Blurred.Option = s.Blurred.Option.UnsetForeground()
		s.Blurred.UnselectedOption = s.Blurred.UnselectedOption.UnsetForeground()
		return s
	})
}

// Select prompts for a single choice from a filterable list.
func Select(title string, opts []string) (string, error) {
	return active.Select(title, opts, "")
}

// SelectDefault is Select with the cursor starting on def (when present).
func SelectDefault(title string, opts []string, def string) (string, error) {
	return active.Select(title, opts, def)
}

// MultiSelect prompts for zero or more choices from a filterable list.
func MultiSelect(title string, opts []string, selected ...string) ([]string, error) {
	return active.MultiSelect(title, opts, selected)
}

// Input prompts for free text.
func Input(title string) (string, error) { return active.Input(title, "") }

// InputDefault prompts for free text, prefilled with an editable default.
func InputDefault(title, def string) (string, error) { return active.Input(title, def) }

// PathInput prompts for a filesystem path with as-you-type completion.
func PathInput(title string) (string, error) { return active.Path(title) }

// Confirm prompts yes/no.
func Confirm(title string) (bool, error) { return active.Confirm(title, false) }

// ConfirmDefault is Confirm starting on the given answer.
func ConfirmDefault(title string, def bool) (bool, error) { return active.Confirm(title, def) }

// height caps long lists so they scroll instead of flooding the screen; short
// lists get 0 = unset, which makes huh size the viewport to the options (no
// blank filler rows).
func height(n int) int {
	if n > 10 {
		return 14
	}
	return 0
}

// multiHeight is height for MultiSelect, whose auto-size clips the last row
// (the title is subtracted from the options height), so short lists get an
// explicit options+title height instead of 0.
func multiHeight(n int) int {
	if n > 10 {
		return 14
	}
	return n + 1
}

func (huhPrompter) Select(title string, opts []string, def string) (string, error) {
	v := def
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(height(len(opts))).
			Filtering(true).
			Value(&v),
	)).WithTheme(theme()), true)
	if errors.Is(err, errCancel) {
		return "", nil
	}
	return v, err
}

func (huhPrompter) MultiSelect(title string, opts []string, preselected []string) ([]string, error) {
	v := append([]string{}, preselected...)
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(multiHeight(len(opts))).
			Filterable(true).
			Value(&v),
	)).WithTheme(theme()), true)
	if errors.Is(err, errCancel) {
		return nil, nil
	}
	return v, err
}

// Path prompts for a filesystem path with as-you-type completion
// (suggestions come from the directory being typed; tab/→ accepts).
func (huhPrompter) Path(title string) (string, error) {
	var v string
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(title).
			Description("tab or → to complete").
			SuggestionsFunc(func() []string { return pathSuggestions(v) }, &v).
			Value(&v),
	)).WithTheme(theme()), false)
	if errors.Is(err, errCancel) {
		return "", nil
	}
	return v, err
}

// pathSuggestions completes the typed path from the filesystem, preserving the
// user's spelling of the prefix (~ stays ~). Directories get a trailing /.
func pathSuggestions(typed string) []string {
	if typed == "" {
		return []string{"~/"}
	}
	i := strings.LastIndexByte(typed, '/')
	if i < 0 {
		if strings.HasPrefix("~/", typed) {
			return []string{"~/"}
		}
		return nil
	}
	typedDir := typed[:i+1] // up to and including the slash, exactly as typed
	base := typed[i+1:]
	ents, err := os.ReadDir(home.Expand(typedDir))
	if err != nil {
		return nil
	}
	var out []string
	for _, e := range ents {
		name := e.Name()
		if !strings.HasPrefix(name, base) {
			continue
		}
		s := typedDir + name
		if e.IsDir() {
			s += "/"
		}
		out = append(out, s)
		if len(out) == 12 {
			break
		}
	}
	return out
}

func (huhPrompter) Input(title, def string) (string, error) {
	v := def
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Value(&v),
	)).WithTheme(theme()), false)
	if errors.Is(err, errCancel) {
		return "", nil
	}
	return v, err
}

// Confirm asks yes/no. CASA_YES=1 answers yes without prompting (scripting).
func (huhPrompter) Confirm(title string, def bool) (bool, error) {
	if os.Getenv("CASA_YES") == "1" {
		fmt.Println(title + "  → yes (CASA_YES)")
		return true, nil
	}
	v := def
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(&v),
	)).WithTheme(theme()), false)
	if errors.Is(err, errCancel) {
		return false, nil
	}
	return v, err
}
