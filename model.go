package main

import (
	"log"
	"strings"
	"time"

	"github.com/atotto/clipboard"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

const (
	listView uint = iota
	editorView
	pinnedView
	promptView
)

// promptMode tells the title prompt what to do on confirm.
type promptMode uint

const (
	promptCreate promptMode = iota // new empty pinned note
	promptRename                   // rename the selected pinned note
	promptPinDay                   // pin a copy of the selected day
)

const (
	debounceDelay = 1500 * time.Millisecond
	maxWait       = 7 * time.Second
)

// autoSaveMsg is fired by the debounce (tea.Tick). seq ensures only the
// latest tick actually saves (earlier keystrokes reschedule the timer).
type autoSaveMsg struct{ seq int }

// savedMsg reports the result of an async save.
type savedMsg struct {
	at  time.Time
	err error
}

// clearCopiedMsg forces a redraw so the "copied!" notice goes away.
type clearCopiedMsg struct{}

type model struct {
	state      uint
	store      *Store
	notes      []Note
	listIndex  int
	editor     textarea.Model
	currentDay string

	pinned        []Pinned
	pinnedIndex   int
	editingPinned int64  // id of the pinned note in the editor; 0 = editing a day
	pinnedTitle   string // title shown in the editor header
	returnTo      uint   // tab to return to from the editor/prompt
	savedBody     string // last persisted body, to skip no-op saves

	prompt        textinput.Model
	promptMode    promptMode
	pendingBody   string // day body copied when pinning
	confirmDelete bool

	dirty     bool
	saveSeq   int       // debounce sequence
	lastSaved time.Time // last actual save (for the throttle/max-wait)
	savedAt   time.Time // to show "saved at HH:MM:SS"
	saving    bool
	saveErr   error
	copiedAt  time.Time // to show "copied!" after Ctrl+Y

	width  int
	height int
}

func newModel(store *Store) model {
	notes, err := store.ListDays()
	if err != nil {
		log.Fatalf("Unable to get notes: %v", err)
	}
	pinned, err := store.ListPinned()
	if err != nil {
		log.Fatalf("Unable to get pinned notes: %v", err)
	}

	ta := textarea.New()
	ta.Placeholder = "Write your note for the day..."
	ta.CharLimit = 0          // no limit
	ta.ShowLineNumbers = true // show line numbers in the gutter

	ti := textinput.New()
	ti.CharLimit = 120

	return model{
		state:  listView,
		store:  store,
		notes:  notes,
		pinned: pinned,
		editor: ta,
		prompt: ti,
	}
}

func (m model) Init() tea.Cmd {
	return nil
}

// openEditor loads body into the editor; used for both days and pinned notes.
func (m *model) openEditor(body string) tea.Cmd {
	m.editor.SetValue(body) // puts the cursor at the end of the content
	m.editor.Focus()
	m.state = editorView
	m.savedBody = body
	m.dirty = false
	m.saveErr = nil
	m.savedAt = time.Time{}
	m.lastSaved = time.Now()
	m.resizeEditor()
	return textarea.Blink
}

// openDay loads a day's note and enters the editor.
func (m *model) openDay(day string) tea.Cmd {
	body, err := m.store.GetDay(day)
	if err != nil {
		log.Printf("Error loading day %s: %v", day, err)
	}
	m.currentDay = day
	m.editingPinned = 0
	m.returnTo = listView
	m.editor.Placeholder = "Write your note for the day..."
	return m.openEditor(body)
}

// openPinned opens a pinned note in the editor.
func (m *model) openPinned(p Pinned) tea.Cmd {
	m.editingPinned = p.ID
	m.pinnedTitle = p.Title
	m.returnTo = pinnedView
	m.editor.Placeholder = "Write your pinned note..."
	return m.openEditor(p.Body)
}

// persist writes body to the editor's target (day or pinned note).
func persist(store *Store, day string, pinnedID int64, body string) error {
	if pinnedID != 0 {
		return store.UpdatePinnedBody(pinnedID, body)
	}
	return store.UpsertDay(day, body)
}

// flush saves synchronously, only if changed since the last save.
func (m *model) flush() {
	body := m.editor.Value()
	if body == m.savedBody {
		return
	}
	if err := persist(m.store, m.currentDay, m.editingPinned, body); err != nil {
		log.Printf("Error saving: %v", err)
	}
}

// scheduleSave schedules a debounced save and returns the tick Cmd.
func (m *model) scheduleSave() tea.Cmd {
	m.dirty = true
	m.saveSeq++
	seq := m.saveSeq
	return tea.Tick(debounceDelay, func(time.Time) tea.Msg {
		return autoSaveMsg{seq: seq}
	})
}

// saveCmd persists the current content asynchronously.
func (m *model) saveCmd() tea.Cmd {
	day := m.currentDay
	pinnedID := m.editingPinned
	body := m.editor.Value()
	store := m.store
	m.saving = true
	m.dirty = false
	m.lastSaved = time.Now()
	m.savedBody = body
	return func() tea.Msg {
		err := persist(store, day, pinnedID, body)
		return savedMsg{at: time.Now(), err: err}
	}
}

// reloadLists reloads days and pinned notes, clamping the indexes.
func (m *model) reloadLists() {
	if notes, err := m.store.ListDays(); err != nil {
		log.Printf("Error reloading list: %v", err)
	} else {
		m.notes = notes
	}
	if pinned, err := m.store.ListPinned(); err != nil {
		log.Printf("Error reloading pinned notes: %v", err)
	} else {
		m.pinned = pinned
	}
	if m.listIndex >= len(m.notes) {
		m.listIndex = 0
	}
	if m.pinnedIndex >= len(m.pinned) {
		m.pinnedIndex = max(len(m.pinned)-1, 0)
	}
}

// openPrompt shows the title prompt, prefilled with value.
func (m *model) openPrompt(mode promptMode, value string) tea.Cmd {
	m.returnTo = m.state
	m.promptMode = mode
	m.prompt.SetValue(value)
	m.prompt.CursorEnd()
	m.state = promptView
	return m.prompt.Focus()
}

func (m *model) resizeEditor() {
	if m.width == 0 || m.height == 0 {
		return
	}
	// reserve lines for the header (2) and footer (2)
	m.editor.SetWidth(m.width - 2)
	m.editor.SetHeight(max(m.height-4, 3))
	m.prompt.Width = m.width - 4
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
			log.Printf("Error saving: %v", msg.err)
		}
		return m, nil

	case clearCopiedMsg:
		// nothing to do besides triggering the redraw that hides "copied!".
		return m, nil

	case autoSaveMsg:
		// Only save if this is the latest tick (the user stopped typing)
		// or max-wait (throttle) was exceeded while still typing.
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
	case pinnedView:
		return m.updatePinned(msg)
	case promptView:
		return m.updatePrompt(msg)
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
		case "tab":
			m.state = pinnedView
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
		case "p":
			if len(m.notes) > 0 {
				note := m.notes[m.listIndex]
				m.pendingBody = note.Body
				return m, m.openPrompt(promptPinDay, note.Preview())
			}
		}
	}
	return m, nil
}

func (m model) updatePinned(msg tea.Msg) (tea.Model, tea.Cmd) {
	key, ok := msg.(tea.KeyMsg)
	if !ok {
		return m, nil
	}

	// second key of the delete confirmation: only "y" deletes.
	if m.confirmDelete {
		m.confirmDelete = false
		if key.String() == "y" && len(m.pinned) > 0 {
			if err := m.store.DeletePinned(m.pinned[m.pinnedIndex].ID); err != nil {
				log.Printf("Error deleting: %v", err)
			}
			m.reloadLists()
		}
		return m, nil
	}

	switch key.String() {
	case "ctrl+c", "q":
		return m, tea.Quit
	case "tab":
		m.state = listView
	case "up", "k":
		if m.pinnedIndex > 0 {
			m.pinnedIndex--
		}
	case "down", "j":
		if m.pinnedIndex < len(m.pinned)-1 {
			m.pinnedIndex++
		}
	case "enter":
		if len(m.pinned) > 0 {
			return m, m.openPinned(m.pinned[m.pinnedIndex])
		}
	case "n":
		m.pendingBody = ""
		return m, m.openPrompt(promptCreate, "")
	case "r":
		if len(m.pinned) > 0 {
			return m, m.openPrompt(promptRename, m.pinned[m.pinnedIndex].Title)
		}
	case "d":
		if len(m.pinned) > 0 {
			m.confirmDelete = true
		}
	}
	return m, nil
}

func (m model) updatePrompt(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "esc":
			m.prompt.Blur()
			m.state = m.returnTo
			return m, nil
		case "enter":
			title := strings.TrimSpace(m.prompt.Value())
			if title == "" {
				return m, nil
			}
			m.prompt.Blur()

			if m.promptMode == promptRename {
				if err := m.store.RenamePinned(m.pinned[m.pinnedIndex].ID, title); err != nil {
					log.Printf("Error renaming: %v", err)
				}
				m.reloadLists()
				m.pinnedIndex = 0 // renaming counts as an edit: it moves to the top
				m.state = pinnedView
				return m, nil
			}

			// create (empty or a copy of the day) and open it in the editor.
			id, err := m.store.CreatePinned(title, m.pendingBody)
			if err != nil {
				log.Printf("Error creating pinned note: %v", err)
				m.state = m.returnTo
				return m, nil
			}
			m.reloadLists()
			m.pinnedIndex = 0 // the new note is the most recent
			return m, m.openPinned(Pinned{ID: id, Title: title, Body: m.pendingBody})
		}
	}

	var cmd tea.Cmd
	m.prompt, cmd = m.prompt.Update(msg)
	return m, cmd
}

func (m model) updateEditor(msg tea.Msg) (tea.Model, tea.Cmd) {
	if key, ok := msg.(tea.KeyMsg); ok {
		switch key.String() {
		case "ctrl+c":
			// synchronous flush before quitting, to avoid data loss.
			m.flush()
			return m, tea.Quit
		case "ctrl+y":
			// copy the raw note content to the clipboard — no line numbers
			// or gutter, unlike a terminal selection.
			if err := clipboard.WriteAll(m.editor.Value()); err != nil {
				log.Printf("Error copying: %v", err)
				return m, nil
			}
			m.copiedAt = time.Now()
			// schedule a redraw so the "copied!" notice goes away.
			return m, tea.Tick(2*time.Second, func(time.Time) tea.Msg {
				return clearCopiedMsg{}
			})
		case "esc":
			// synchronous flush, then back to the originating tab with fresh lists.
			m.flush()
			m.reloadLists()
			if m.editingPinned != 0 {
				m.pinnedIndex = indexOfPinned(m.pinned, m.editingPinned)
			}
			m.state = m.returnTo
			m.editor.Blur()
			m.dirty = false
			return m, nil
		}
	}

	var cmd tea.Cmd
	prev := m.editor.Value()
	m.editor, cmd = m.editor.Update(msg)

	cmds := []tea.Cmd{cmd}
	if m.editor.Value() != prev {
		// content changed → schedule debounce; if max-wait was exceeded, save now.
		cmds = append(cmds, m.scheduleSave())
		if time.Since(m.lastSaved) >= maxWait {
			cmds = append(cmds, m.saveCmd())
		}
	}
	return m, tea.Batch(cmds...)

}

// indexOfPinned returns the position of note id in the list (0 if missing),
// so the selection stays on it after the order changes.
func indexOfPinned(pinned []Pinned, id int64) int {
	for i, p := range pinned {
		if p.ID == id {
			return i
		}
	}
	return 0
}
