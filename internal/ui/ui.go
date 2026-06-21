// Package ui wraps charm's huh forms for casa's interactive prompts.
package ui

import "charm.land/huh/v2"

// Select prompts for a single choice.
func Select(title string, opts []string) (string, error) {
	var v string
	err := huh.NewForm(huh.NewGroup(
		huh.NewSelect[string]().Title(title).Options(huh.NewOptions(opts...)...).Value(&v),
	)).Run()
	return v, err
}

// MultiSelect prompts for zero or more choices.
func MultiSelect(title string, opts []string) ([]string, error) {
	var v []string
	err := huh.NewForm(huh.NewGroup(
		huh.NewMultiSelect[string]().Title(title).Options(huh.NewOptions(opts...)...).Value(&v),
	)).Run()
	return v, err
}

// Input prompts for free text.
func Input(title string) (string, error) {
	var v string
	err := huh.NewForm(huh.NewGroup(
		huh.NewInput().Title(title).Value(&v),
	)).Run()
	return v, err
}

// Confirm prompts yes/no.
func Confirm(title string) (bool, error) {
	var v bool
	err := huh.NewForm(huh.NewGroup(
		huh.NewConfirm().Title(title).Value(&v),
	)).Run()
	return v, err
}
