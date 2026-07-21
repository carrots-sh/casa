package ui

import (
	"slices"
	"testing"

	tea "charm.land/bubbletea/v2"
)

func keys(m *fzfMulti, presses ...string) {
	for _, p := range presses {
		var msg tea.KeyPressMsg
		switch p {
		case "space":
			msg = tea.KeyPressMsg{Code: tea.KeySpace, Text: " "}
		case "backspace":
			msg = tea.KeyPressMsg{Code: tea.KeyBackspace}
		case "enter":
			msg = tea.KeyPressMsg{Code: tea.KeyEnter}
		default: // single rune
			msg = tea.KeyPressMsg{Code: rune(p[0]), Text: p}
		}
		m.Update(msg)
	}
}

func TestFzfMultiSelectionsPersistAcrossFilters(t *testing.T) {
	m := newFzfMulti("t", []string{"brew helix", "brew just", "go gopls", "npm ruflo"}, nil)

	keys(m, "h", "e", "l") // filter → "brew helix"
	if len(m.visible) != 1 || m.visible[0] != "brew helix" {
		t.Fatalf("filter hel → %v", m.visible)
	}
	keys(m, "space")                                     // select helix
	keys(m, "backspace", "backspace", "backspace")       // clear filter
	keys(m, "j", "u")                                    // filter → "brew just"
	keys(m, "space")                                     // select just
	keys(m, "backspace", "backspace", "g", "o", "space") // and gopls ("go" matches gopls first)

	keys(m, "enter")
	got := m.selected()
	want := []string{"brew helix", "brew just", "go gopls"}
	if !slices.Equal(got, want) {
		t.Fatalf("selected = %v, want %v", got, want)
	}
	if !m.done {
		t.Fatal("enter did not submit")
	}
}

func TestFzfMultiEscBacksOutAndCtrlAToggle(t *testing.T) {
	m := newFzfMulti("t", []string{"a1", "a2", "b1"}, []string{"b1"})
	keys(m, "a")
	m.Update(tea.KeyPressMsg{Code: 'a', Mod: tea.ModCtrl}) // ctrl+a → all visible
	got := m.selected()
	if !slices.Equal(got, []string{"a1", "a2", "b1"}) {
		t.Fatalf("ctrl+a over filtered = %v", got)
	}
	m.Update(tea.KeyPressMsg{Code: tea.KeyEscape})
	if !m.back {
		t.Fatal("esc did not back out")
	}
}
