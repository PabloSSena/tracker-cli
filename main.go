package main

import (
	"log"

	tea "github.com/charmbracelet/bubbletea"
)

func main() {
	store := &Store{}

	if err := store.Init(); err != nil {
		log.Fatalf("Error initializing store: %v", err)
	}

	m := newModel(store)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		log.Fatalf("Error running program: %v", err)
	}
}
