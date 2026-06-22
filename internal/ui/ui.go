// Package ui wraps charm's huh forms for casa's interactive prompts.
package ui

import (
	"os"
	"sync"

	"charm.land/huh/v2"
	"charm.land/lipgloss/v2"
)

// theme is detected once, lazily, so non-interactive commands never query the
// terminal. It adapts to the terminal's actual background (fixes light mode).
var (
	themeOnce sync.Once
	themeVal  huh.Theme
)

func theme() huh.Theme {
	themeOnce.Do(func() {
		// Detect the real terminal background ourselves and pin it, since huh's
		// own detection can default to dark (washing out light terminals).
		isDark := lipgloss.HasDarkBackground(os.Stdin, os.Stdout)
		themeVal = huh.ThemeFunc(func(bool) *huh.Styles { return huh.ThemeCharm(isDark) })
	})
	return themeVal
}

// Select prompts for a single choice from a filterable list.
func Select(title string, opts []string) (string, error) {
	var v string
	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(14).
			Filtering(true).
			Value(&v),
	)).WithTheme(theme()).Run()
	return v, err
}

// MultiSelect prompts for zero or more choices from a filterable list.
func MultiSelect(title string, opts []string) ([]string, error) {
	var v []string
	err := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(14).
			Filterable(true).
			Value(&v),
	)).WithTheme(theme()).Run()
	return v, err
}

// MultiSelectPreselected is like MultiSelect but starts with some options ticked.
func MultiSelectPreselected(title string, opts, selected []string) ([]string, error) {
	v := append([]string{}, selected...)
	err := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().
			Title(title).
			Options(huh.NewOptions(opts...)...).
			Height(14).
			Filterable(true).
			Value(&v),
	)).WithTheme(theme()).Run()
	return v, err
}

// Input prompts for free text.
func Input(title string) (string, error) {
	var v string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Value(&v),
	)).WithTheme(theme()).Run()
	return v, err
}

// Confirm prompts yes/no.
func Confirm(title string) (bool, error) {
	var v bool
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(&v),
	)).WithTheme(theme()).Run()
	return v, err
}
