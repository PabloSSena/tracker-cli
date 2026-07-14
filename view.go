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
)

func (m model) View() string {
	switch m.state {
	case editorView:
		return m.editorViewRender()
	default:
		return m.listViewRender()
	}
}

// prettyDay converte YYYY-MM-DD para uma exibição amigável, marcando "hoje".
func prettyDay(day string) string {
	t, err := time.Parse("2006-01-02", day)
	if err != nil {
		return day
	}
	label := t.Format("02/01/2006 (Mon)")
	if day == time.Now().Format("2006-01-02") {
		return label + " · hoje"
	}
	return label
}

func (m model) listViewRender() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("📓 Tracker — suas anotações"))
	b.WriteString("\n")

	if len(m.notes) == 0 {
		b.WriteString(emptyStyle.Render("Nenhuma anotação ainda — pressione `n` para começar a de hoje."))
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

	b.WriteString(helpStyle.Render("↑/↓ navegar · enter abrir · n nova (hoje) · ctrl+c sair"))
	return b.String()
}

func (m model) editorViewRender() string {
	var b strings.Builder
	b.WriteString(titleStyle.Render("✍️  " + prettyDay(m.currentDay)))
	b.WriteString("\n")
	b.WriteString(m.editor.View())
	b.WriteString("\n")

	status := "salvamento automático"
	switch {
	case time.Since(m.copiedAt) < 2*time.Second:
		status = savedStyle.Render("copiado!")
	case m.saveErr != nil:
		status = savingStyle.Render("erro ao salvar!")
	case m.saving:
		status = savingStyle.Render("salvando…")
	case !m.savedAt.IsZero():
		status = savedStyle.Render("salvo às " + m.savedAt.Format("15:04:05"))
	}
	b.WriteString(helpStyle.Render(fmt.Sprintf("esc voltar · ctrl+y copiar · %s", status)))
	return b.String()
}

func truncate(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
