package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/qyinm/phtui/cli"
	"github.com/qyinm/phtui/scraper"
	"github.com/qyinm/phtui/ui"
)

func main() {
	source := scraper.New()
	if len(os.Args) > 1 && cli.IsCommand(os.Args[1]) {
		if err := cli.Run(os.Args[1:], os.Stdout, source); err != nil {
			fmt.Fprintf(os.Stderr, "Error: %v\n", err)
			os.Exit(1)
		}
		return
	}

	m := ui.NewModel(source)
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
