// Package ui wraps charm's huh forms for casa's interactive prompts.
package ui

import (
	"errors"
	"os"
	"sync"

	"charm.land/huh/v2"
)

// run executes a form; ctrl+c (abort) quits casa cleanly from anywhere.
func run(f *huh.Form) error {
	err := f.Run()
	if errors.Is(err, huh.ErrUserAborted) {
		os.Exit(0)
	}
	return err
}

var (
	themeOnce sync.Once
	themeVal  huh.Theme
)

// theme keeps charm's accent for the selected item but renders every other
// option in the terminal's own foreground, so it stays readable on any
// background (huh's auto color picks wash out in light terminals).
func theme() huh.Theme {
	themeOnce.Do(func() {
		themeVal = huh.ThemeFunc(func(isDark bool) *huh.Styles {
			s := huh.ThemeCharm(isDark)
			s.Focused.Option = s.Focused.Option.UnsetForeground()
			s.Focused.UnselectedOption = s.Focused.UnselectedOption.UnsetForeground()
			s.Blurred.Option = s.Blurred.Option.UnsetForeground()
			s.Blurred.UnselectedOption = s.Blurred.UnselectedOption.UnsetForeground()
			return s
		})
	})
	return themeVal
}

// Select prompts for a single choice from a filterable list.
func Select(title string, opts []string) (string, error) {
	var v string
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(14).
			Filtering(true).
			Value(&v),
	)).WithTheme(theme()))
	return v, err
}

// MultiSelect prompts for zero or more choices from a filterable list.
func MultiSelect(title string, opts []string) ([]string, error) {
	var v []string
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(14).
			Filterable(true).
			Value(&v),
	)).WithTheme(theme()))
	return v, err
}

// MultiSelectPreselected is like MultiSelect but starts with some options ticked.
func MultiSelectPreselected(title string, opts, selected []string) ([]string, error) {
	v := append([]string{}, selected...)
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(14).
			Filterable(true).
			Value(&v),
	)).WithTheme(theme()))
	return v, err
}

// Input prompts for free text.
func Input(title string) (string, error) {
	var v string
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Value(&v),
	)).WithTheme(theme()))
	return v, err
}

// Confirm prompts yes/no.
func Confirm(title string) (bool, error) {
	var v bool
	err := run(huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(&v),
	)).WithTheme(theme()))
	return v, err
}
