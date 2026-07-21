// fzf-style multiselect: printable keys narrow the list as you type, space
// toggles the highlighted row, and selections persist while the filter
// changes — select, retype, select again.
package ui

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
)

type fzfMulti struct {
	title   string
	opts    []string
	sel     map[string]bool
	filter  string
	visible []string
	cursor  int
	offset  int
	height  int
	done    bool // enter
	back    bool // esc / ←
	quit    bool // ctrl+c
}

func newFzfMulti(title string, opts []string, preselected []string) *fzfMulti {
	m := &fzfMulti{title: title, opts: opts, sel: map[string]bool{}, height: 10}
	for _, p := range preselected {
		m.sel[p] = true
	}
	m.refilter()
	return m
}

func (m *fzfMulti) refilter() {
	m.visible = m.visible[:0]
	f := strings.ToLower(m.filter)
	for _, o := range m.opts {
		if f == "" || strings.Contains(strings.ToLower(o), f) {
			m.visible = append(m.visible, o)
		}
	}
	if m.cursor >= len(m.visible) {
		m.cursor = max(len(m.visible)-1, 0)
	}
	m.scroll()
}

func (m *fzfMulti) scroll() {
	if m.cursor < m.offset {
		m.offset = m.cursor
	}
	if m.cursor >= m.offset+m.height {
		m.offset = m.cursor - m.height + 1
	}
	if m.offset < 0 {
		m.offset = 0
	}
}

func (m *fzfMulti) Init() tea.Cmd { return nil }

func (m *fzfMulti) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	k, ok := msg.(tea.KeyPressMsg)
	if !ok {
		return m, nil
	}
	switch s := k.String(); s {
	case "ctrl+c":
		m.quit = true
		return m, tea.Quit
	case "esc", "left":
		m.back = true
		return m, tea.Quit
	case "enter":
		m.done = true
		return m, tea.Quit
	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
		m.scroll()
	case "down", "ctrl+n":
		if m.cursor < len(m.visible)-1 {
			m.cursor++
		}
		m.scroll()
	case "pgup":
		m.cursor = max(m.cursor-m.height, 0)
		m.scroll()
	case "pgdown":
		m.cursor = min(m.cursor+m.height, max(len(m.visible)-1, 0))
		m.scroll()
	case "backspace":
		if m.filter != "" {
			m.filter = m.filter[:len(m.filter)-1]
			m.refilter()
		}
	case "ctrl+a": // toggle all currently visible
		all := true
		for _, o := range m.visible {
			if !m.sel[o] {
				all = false
				break
			}
		}
		for _, o := range m.visible {
			m.sel[o] = !all
		}
	case "space", " ", "tab":
		if len(m.visible) > 0 {
			o := m.visible[m.cursor]
			m.sel[o] = !m.sel[o]
		}
	default:
		if r := []rune(s); len(r) == 1 && r[0] >= ' ' && r[0] != 0x7f {
			m.filter += s
			m.refilter()
		}
	}
	return m, nil
}

var (
	fzfAccent = lipgloss.NewStyle().Foreground(lipgloss.Color("212"))
	fzfTitle  = lipgloss.NewStyle().Foreground(lipgloss.Color("99")).Bold(true)
	fzfCheck  = lipgloss.NewStyle().Foreground(lipgloss.Color("42"))
	fzfDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("241"))
	fzfBar    = lipgloss.NewStyle().Foreground(lipgloss.Color("238"))
)

func (m *fzfMulti) View() tea.View {
	if m.done || m.back || m.quit {
		return tea.NewView("")
	}
	var b strings.Builder
	bar := fzfBar.Render("┃") + " "
	fmt.Fprintf(&b, "%s%s\n", bar, fzfTitle.Render(m.title))
	fmt.Fprintf(&b, "%s%s%s%s\n", bar, fzfAccent.Render("/ "), m.filter, fzfAccent.Render("█"))
	end := min(m.offset+m.height, len(m.visible))
	for i := m.offset; i < end; i++ {
		o := m.visible[i]
		cursor, check := "  ", fzfDim.Render("•")
		if i == m.cursor {
			cursor = fzfAccent.Render("> ")
		}
		if m.sel[o] {
			check = fzfCheck.Render("✓")
		}
		line := o
		if i == m.cursor {
			line = fzfAccent.Render(o)
		}
		fmt.Fprintf(&b, "%s%s%s %s\n", bar, cursor, check, line)
	}
	if len(m.visible) == 0 {
		fmt.Fprintf(&b, "%s%s\n", bar, fzfDim.Render("  (no matches)"))
	}
	n := 0
	for _, o := range m.opts {
		if m.sel[o] {
			n++
		}
	}
	fmt.Fprintf(&b, "\n%s", fzfDim.Render(fmt.Sprintf(
		"%d selected · type to filter · space select · ctrl+a all · enter submit · esc/← back", n)))
	return tea.NewView(b.String())
}

// selected returns the picked options in original order.
func (m *fzfMulti) selected() []string {
	var out []string
	for _, o := range m.opts {
		if m.sel[o] {
			out = append(out, o)
		}
	}
	return out
}
