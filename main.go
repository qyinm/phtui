package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qyinm/phtui/scraper"
	"github.com/qyinm/phtui/ui"
)

func main() {
	source := scraper.New()
	m := ui.NewModel(source)
	p := tea.NewProgram(m, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
