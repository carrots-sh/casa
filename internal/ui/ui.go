// Package ui wraps charm's huh forms for casa's interactive prompts.
package ui

import (
	"errors"
	"os"

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

// SelectDefault is Select with the cursor starting on def (when present).
func SelectDefault(title string, opts []string, def string) (string, error) {
	v := def
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(14).
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
			Height(14).
			Filterable(true).
			Value(&v),
	)).WithTheme(theme()))
	if errors.Is(err, errCancel) {
		return nil, nil
	}
	return v, err
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

// Confirm prompts yes/no.
func Confirm(title string) (bool, error) {
	return ConfirmDefault(title, false)
}

// ConfirmDefault is Confirm starting on the given answer.
func ConfirmDefault(title string, def bool) (bool, error) {
	v := def
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(&v),
	)).WithTheme(theme()))
	if errors.Is(err, errCancel) {
		return false, nil
	}
	return v, err
}
