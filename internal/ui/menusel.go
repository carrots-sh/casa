// Section-aware single-select for the main menu: same fzf-style typing as
// the multiselect, but the section gutter is computed per visible row — it
// renders once per contiguous cluster, whether the list is filtered or not.
package ui

import (
	"fmt"
	"os"
	"strings"

	tea "charm.land/bubbletea/v2"
)

// MenuRow is one action row: Section renders as a deduped gutter label,
// Name/Desc/Hint make up the row text. Filtering matches all of them.
type MenuRow struct {
	Section, Name, Desc, Hint string
}

func (r MenuRow) text() string {
	s := fmt.Sprintf("%-12s", r.Name)
	if r.Desc != "" {
		s += " · " + r.Desc
	}
	return s + r.Hint
}

// Menu shows the section-clustered menu and returns the picked row index,
// or -1 when the user backs out (esc/←).
func Menu(title string, rows []MenuRow, defaultIdx int) (int, error) {
	m := &menuSel{title: title, rows: rows, height: 24}
	m.refilter()
	for i, v := range m.visible {
		if v == defaultIdx {
			m.cursor = i
		}
	}
	m.scroll()
	final, err := tea.NewProgram(m).Run()
	if err != nil {
		return -1, err
	}
	ms := final.(*menuSel)
	if ms.quit {
		os.Exit(0)
	}
	if ms.back || !ms.done || len(ms.visible) == 0 {
		return -1, nil
	}
	return ms.visible[ms.cursor], nil
}

type menuSel struct {
	title   string
	rows    []MenuRow
	filter  string
	visible []int // indices into rows
	cursor  int   // index into visible
	offset  int
	height  int
	done    bool
	back    bool
	quit    bool
}

func (m *menuSel) refilter() {
	m.visible = m.visible[:0]
	f := strings.ToLower(m.filter)
	for i, r := range m.rows {
		if f == "" || strings.Contains(strings.ToLower(r.Section+" "+r.text()), f) {
			m.visible = append(m.visible, i)
		}
	}
	if m.cursor >= len(m.visible) {
		m.cursor = max(len(m.visible)-1, 0)
	}
	m.scroll()
}

func (m *menuSel) scroll() {
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

func (m *menuSel) Init() tea.Cmd { return nil }

func (m *menuSel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	if w, ok := msg.(tea.WindowSizeMsg); ok {
		m.height = max(w.Height-5, 5) // title + filter + footer + margins
		m.scroll()
		return m, nil
	}
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
	default:
		if r := []rune(s); len(r) == 1 && r[0] >= ' ' && r[0] != 0x7f {
			m.filter += s
			m.refilter()
		}
	}
	return m, nil
}

func (m *menuSel) View() tea.View {
	if m.done || m.back || m.quit {
		return tea.NewView("")
	}
	var b strings.Builder
	bar := fzfBar.Render("┃") + " "
	fmt.Fprintf(&b, "%s%s\n", bar, fzfTitle.Render(m.title))
	fmt.Fprintf(&b, "%s%s%s%s\n", bar, fzfAccent.Render("/ "), m.filter, fzfAccent.Render("█"))
	end := min(m.offset+m.height, len(m.visible))
	for i := m.offset; i < end; i++ {
		r := m.rows[m.visible[i]]
		// gutter: section name only on the first visible row of each
		// contiguous cluster — once when unfiltered, per-group when filtered
		gutter := ""
		if i == 0 || m.rows[m.visible[i-1]].Section != r.Section {
			gutter = r.Section
		}
		cursor := "  "
		line := r.text()
		if i == m.cursor {
			cursor = fzfAccent.Render("> ")
			line = fzfAccent.Render(line)
		}
		fmt.Fprintf(&b, "%s%s%s  %s\n", bar, cursor, fzfDim.Render(fmt.Sprintf("%-8s", gutter)), line)
	}
	if len(m.visible) == 0 {
		fmt.Fprintf(&b, "%s%s\n", bar, fzfDim.Render("  (no matches)"))
	}
	fmt.Fprintf(&b, "\n%s", fzfDim.Render("type to filter · ↑↓ move · enter select · esc/← quit"))
	return tea.NewView(b.String())
}
