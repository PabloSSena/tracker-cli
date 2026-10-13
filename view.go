package main

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205")).
			MarginBottom(1)

	selectedStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("231")).
			Background(lipgloss.Color("57")).
			Padding(0, 1)

	itemStyle = lipgloss.NewStyle().
			Padding(0, 1)

	previewStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245"))

	helpStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("241")).
			MarginTop(1)

	emptyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("245")).
			Italic(true)

	savedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("42"))

	savingStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("214"))

	activeTabStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("205"))

	inactiveTabStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("241"))

	pinTitleStyle = lipgloss.NewStyle().Bold(true)
)

func (m model) View() string {
	switch m.state {
	case editorView:
		return m.editorViewRender()
	case pinnedView:
		return m.pinnedViewRender()
	case promptView:
		return m.promptViewRender()
	default:
		return m.listViewRender()
	}
}

// prettyDay turns YYYY-MM-DD into a friendly label, marking "today".
func prettyDay(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	label := t.Format("Jan 02, 2006 (Mon)")
	if day == time.Now().Format("2006-01-02") {
		return label + " · today"
	}
	return label
}

func (m model) listViewRender() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📓 Tracker — your notes"))
	b.WriteString("\n")
	b.WriteString(m.tabsRender())
	b.WriteString("\n\n")

	if len(m.notes) == 0 {
		b.WriteString(emptyStyle.Render("No notes yet — press `n` to start today's."))
		b.WriteString("\n")
	} else {
		for i, note := range m.notes {
			line := prettyDay(note.Day)
			if preview := note.Preview(); preview != "" {
				line += "  " + previewStyle.Render(truncate(preview, 50))
			}
			if i == m.listIndex {
				b.WriteString(selectedStyle.Render("› " + line))
			} else {
				b.WriteString(itemStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
	}

	b.WriteString(helpStyle.Render("↑/↓ navigate · enter open · n new (today) · p pin · tab pinned · ctrl+c quit"))
	return b.String()
}

func (m model) editorViewRender() string {
	var b strings.Builder
	header := "✍️  " + prettyDay(m.currentDay)
	if m.editingPinned != 0 {
		header = "📌 " + m.pinnedTitle
	}
	b.WriteString(titleStyle.Render(header))
	b.WriteString("\n")
	b.WriteString(m.editor.View())
	b.WriteString("\n")

	status := "autosave"
	switch {
	case time.Since(m.copiedAt) < 2*time.Second:
		status = savedStyle.Render("copied!")
	case m.saveErr != nil:
		status = savingStyle.Render("save failed!")
	case m.saving:
		status = savingStyle.Render("saving…")
	case !m.savedAt.IsZero():
		status = savedStyle.Render("saved at " + m.savedAt.Format("15:04:05"))
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf("esc back · ctrl+y copy · %s", status)))
	return b.String()
}

// tabsRender draws the tabs; the pinned count is always visible.
func (m model) tabsRender() string {
	days, pins := inactiveTabStyle, inactiveTabStyle
	if m.state == pinnedView {
		pins = activeTabStyle
	} else {
		days = activeTabStyle
	}
	return days.Render("Days") + inactiveTabStyle.Render("  ·  ") +
		pins.Render(fmt.Sprintf("📌 %d pinned", len(m.pinned)))
}

func (m model) pinnedViewRender() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📓 Tracker — your notes"))
	b.WriteString("\n")
	b.WriteString(m.tabsRender())
	b.WriteString("\n\n")

	if len(m.pinned) == 0 {
		b.WriteString(emptyStyle.Render("No pinned notes — press `n` to create one."))
		b.WriteString("\n")
	} else {
		for i, p := range m.pinned {
			if i == m.pinnedIndex {
				line := truncate(p.Title, 40)
				if preview := p.Preview(); preview != "" {
					line += "  " + truncate(preview, 50)
				}
				b.WriteString(selectedStyle.Render("› " + line))
			} else {
				line := pinTitleStyle.Render(truncate(p.Title, 40))
				if preview := p.Preview(); preview != "" {
					line += "  " + previewStyle.Render(truncate(preview, 50))
				}
				b.WriteString(itemStyle.Render("  " + line))
			}
			b.WriteString("\n")
		}
	}

	help := "↑/↓ navigate · enter open · n new · r rename · d delete · tab days · ctrl+c quit"
	if m.confirmDelete && len(m.pinned) > 0 {
		help = savingStyle.Render(fmt.Sprintf("delete \"%s\"? (y/n)", m.pinned[m.pinnedIndex].Title))
	}
	b.WriteString(helpStyle.Render(help))
	return b.String()
}

func (m model) promptViewRender() string {
	label := "📌 Pinned note title:"
	if m.promptMode == promptRename {
		label = "📌 Rename pinned note:"
	}
	var b strings.Builder
	b.WriteString(titleStyle.Render(label))
	b.WriteString("\n")
	b.WriteString(m.prompt.View())
	b.WriteString("\n")
	b.WriteString(helpStyle.Render("enter confirm · esc cancel"))
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
