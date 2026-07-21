// Package ui wraps charm's huh forms for casa's interactive prompts.
package ui

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/key"
	tea "charm.land/bubbletea/v2"
	"charm.land/huh/v2"
)

// errCancel is the sentinel run reports when the user presses esc.
var errCancel = errors.New("cancelled")

// run executes a form. Esc cancels the current prompt (a soft back: callers
// treat the returned zero value as cancel); ctrl+c quits casa entirely from
// anywhere. Both keys abort the form the same way in huh, so we bind esc to
// Quit alongside ctrl+c and use a message filter to remember which one fired.
func run(f *huh.Form) error {
	esc := false
	km := huh.NewDefaultKeyMap()
	km.Quit = key.NewBinding(key.WithKeys("ctrl+c", "esc"))
	err := f.WithKeyMap(km).
		WithProgramOptions(tea.WithFilter(func(_ tea.Model, msg tea.Msg) tea.Msg {
			if k, ok := msg.(tea.KeyPressMsg); ok {
				esc = k.String() == "esc"
			}
			return msg
		})).
		Run()
	if errors.Is(err, huh.ErrUserAborted) {
		if esc {
			return errCancel
		}
		os.Exit(0)
	}
	return err
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
	return SelectDefault(title, opts, "")
}

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

// SelectDefault is Select with the cursor starting on def (when present).
func SelectDefault(title string, opts []string, def string) (string, error) {
	v := def
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(height(len(opts))).
			Filtering(true).
			Value(&v),
	)).WithTheme(theme()))
	if errors.Is(err, errCancel) {
		return "", nil
	}
	return v, err
}

// MultiSelect prompts for zero or more choices from a filterable list.
func MultiSelect(title string, opts []string, selected ...string) ([]string, error) {
	v := append([]string{}, selected...)
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(multiHeight(len(opts))).
			Filterable(true).
			Value(&v),
	)).WithTheme(theme()))
	if errors.Is(err, errCancel) {
		return nil, nil
	}
	return v, err
}

// PathInput prompts for a filesystem path with as-you-type completion
// (suggestions come from the directory being typed; tab/→ accepts).
func PathInput(title string) (string, error) {
	var v string
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewInput().
			Title(title).
			Description("tab or → to complete").
			SuggestionsFunc(func() []string { return pathSuggestions(v) }, &v).
			Value(&v),
	)).WithTheme(theme()))
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
	real := typed
	if strings.HasPrefix(typed, "~/") {
		h, _ := os.UserHomeDir()
		real = filepath.Join(h, typed[2:])
		if strings.HasSuffix(typed, "/") {
			real += "/"
		}
	}
	dir, base := filepath.Split(real)
	if dir == "" {
		dir = "."
	}
	ents, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	typedDir := typed[:len(typed)-len(base)]
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

// Input prompts for free text.
func Input(title string) (string, error) {
	return InputDefault(title, "")
}

// InputDefault prompts for free text, prefilled with an editable default.
func InputDefault(title, def string) (string, error) {
	v := def
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Value(&v),
	)).WithTheme(theme()))
	if errors.Is(err, errCancel) {
		return "", nil
	}
	return v, err
}

// Confirm prompts yes/no. CASA_YES=1 answers yes without prompting (scripting/tests).
func Confirm(title string) (bool, error) {
	return ConfirmDefault(title, false)
}

// ConfirmDefault is Confirm starting on the given answer.
func ConfirmDefault(title string, def bool) (bool, error) {
	if os.Getenv("CASA_YES") == "1" {
		fmt.Println(title + "  → yes (CASA_YES)")
		return true, nil
	}
	v := def
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(&v),
	)).WithTheme(theme()))
	if errors.Is(err, errCancel) {
		return false, nil
	}
	return v, err
}
