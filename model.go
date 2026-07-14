package main

import (
	"log"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	listView uint = iota
	editorView
)

const (
	debounceDelay = 1500 * time.Millisecond
	maxWait       = 7 * time.Second
)

// autoSaveMsg é disparado pelo debounce (tea.Tick). O seq garante que apenas
// o tick mais recente efetive o save (as teclas anteriores reagendam o timer).
type autoSaveMsg struct{ seq int }

// savedMsg reporta o resultado de um save assíncrono.
type savedMsg struct {
	at  time.Time
	err error
}

// clearCopiedMsg força um redraw para o aviso "copiado!" sumir sozinho.
type clearCopiedMsg struct{}

type model struct {
	state      uint
	store      *Store
	notes      []Note
	listIndex  int
	editor     textarea.Model
	currentDay string

	dirty     bool
	saveSeq   int       // sequência do debounce
	lastSaved time.Time // último save efetivado (para o throttle/max-wait)
	savedAt   time.Time // para exibir "salvo às HH:MM:SS"
	saving    bool
	saveErr   error
	copiedAt  time.Time // para exibir "copiado!" após Ctrl+Y

	width  int
	height int
}

func newModel(store *Store) model {
	notes, err := store.ListDays()
	if err != nil {
		log.Fatalf("Unable to get notes: %v", err)
	}

	ta := textarea.New()
	ta.Placeholder = "Escreva sua anotação do dia..."
	ta.CharLimit = 0            // sem limite
	ta.ShowLineNumbers = true   // mostra os números de linha na margem

	return model{
		state:  listView,
		store:  store,
		notes:  notes,
		editor: ta,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// openDay carrega a anotação de um dia e entra no editor.
func (m *model) openDay(day string) tea.Cmd {
	body, err := m.store.GetDay(day)
	if err != nil {
		log.Printf("Erro ao carregar dia %s: %v", day, err)
	}
	m.currentDay = day
	m.editor.SetValue(body) // posiciona o cursor no fim do conteúdo
	m.editor.Focus()
	m.state = editorView
	m.dirty = false
	m.saveErr = nil
	m.lastSaved = time.Now()
	m.resizeEditor()
	return textarea.Blink
}

// scheduleSave agenda um save por debounce e devolve o Cmd do tick.
func (m *model) scheduleSave() tea.Cmd {
	m.dirty = true
	m.saveSeq++
	seq := m.saveSeq
	return tea.Tick(debounceDelay, func(time.Time) tea.Msg {
		return autoSaveMsg{seq: seq}
	})
}

// saveCmd persiste o conteúdo atual de forma assíncrona.
func (m *model) saveCmd() tea.Cmd {
	day := m.currentDay
	body := m.editor.Value()
	store := m.store
	m.saving = true
	m.dirty = false
	m.lastSaved = time.Now()
	return func() tea.Msg {
		err := store.UpsertDay(day, body)
		return savedMsg{at: time.Now(), err: err}
	}
}

func (m *model) resizeEditor() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// reserva linhas para cabeçalho (2) e rodapé (2)
	m.editor.SetWidth(m.width - 2)
	h := m.height - 4
	if h < 3 {
		h = 3
	}
	m.editor.SetHeight(h)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.resizeEditor()
		return m, nil

	case savedMsg:
		m.saving = false
		m.saveErr = msg.err
		if msg.err == nil {
			m.savedAt = msg.at
		} else {
			log.Printf("Erro ao salvar: %v", msg.err)
		}
		return m, nil

	case clearCopiedMsg:
		// nada a fazer além de disparar o redraw que esconde o "copiado!".
		return m, nil

	case autoSaveMsg:
		// Só salva se este for o tick mais recente (usuário parou de digitar)
		// ou se estourou o max-wait (throttle) mesmo ainda digitando.
		if m.state != editorView || !m.dirty {
			return m, nil
		}
		if msg.seq == m.saveSeq || time.Since(m.lastSaved) >= maxWait {
			return m, m.saveCmd()
		}
		return m, nil
	}

	switch m.state {
	case listView:
		return m.updateList(msg)
	case editorView:
		return m.updateEditor(msg)
	}
	return m, nil
}

func (m model) updateList(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "up", "k":
			if m.listIndex > 0 {
				m.listIndex--
			}
		case "down", "j":
			if m.listIndex < len(m.notes)-1 {
				m.listIndex++
			}
		case "enter":
			if len(m.notes) > 0 {
				cmd := m.openDay(m.notes[m.listIndex].Day)
				return m, cmd
			}
		case "n":
			today := time.Now().Format("2006-01-02")
			cmd := m.openDay(today)
			return m, cmd
		}
	}
	return m, nil
}

func (m model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			// flush síncrono antes de sair, garantia anti-perda.
			_ = m.store.UpsertDay(m.currentDay, m.editor.Value())
			return m, tea.Quit
		case "ctrl+y":
			// copia o conteúdo puro da nota para o clipboard — sem números
			// de linha nem margem, ao contrário da seleção do terminal.
			if err := clipboard.WriteAll(m.editor.Value()); err != nil {
				log.Printf("Erro ao copiar: %v", err)
				return m, nil
			}
			m.copiedAt = time.Now()
			// agenda um redraw para o aviso "copiado!" sumir sozinho.
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return clearCopiedMsg{}
			})
		case "esc":
			// flush síncrono e volta para a lista, recarregando a listagem.
			_ = m.store.UpsertDay(m.currentDay, m.editor.Value())
			notes, err := m.store.ListDays()
			if err != nil {
				log.Printf("Erro ao recarregar lista: %v", err)
			} else {
				m.notes = notes
			}
			m.state = listView
			m.editor.Blur()
			m.dirty = false
			if m.listIndex >= len(m.notes) {
				m.listIndex = 0
			}
			return m, nil
		}
	}

	var cmd tea.Cmd
	prev := m.editor.Value()
	m.editor, cmd = m.editor.Update(msg)

	cmds := []tea.Cmd{cmd}
	if m.editor.Value() != prev {
		// conteúdo mudou → agenda debounce; se estourou max-wait, salva já.
		cmds = append(cmds, m.scheduleSave())
		if time.Since(m.lastSaved) >= maxWait {
			cmds = append(cmds, m.saveCmd())
		}
	}
	return m, tea.Batch(cmds...)

}
