package ui

import "github.com/charmbracelet/lipgloss"

// 16-color ANSI Dracula palette (identical to lazyadmin)
var (
	DraculaBackground = lipgloss.Color("0")
	DraculaForeground = lipgloss.Color("255")
	DraculaPurple     = lipgloss.Color("5")
	DraculaPink       = lipgloss.Color("13")
	DraculaCyan       = lipgloss.Color("14")
	DraculaGreen      = lipgloss.Color("10")
	DraculaComment    = lipgloss.Color("7")
	DraculaOrange     = lipgloss.Color("3")
	DraculaRed        = lipgloss.Color("1")

	// Tab bar styles
	ActiveTabStyle = lipgloss.NewStyle().
			Foreground(DraculaPink).
			Bold(true).
			Padding(0, 1)
	InactiveTabStyle = lipgloss.NewStyle().
				Foreground(DraculaComment).
				Padding(0, 1)

	// List styles
	TitleStyle = lipgloss.NewStyle().
			Foreground(DraculaPink).
			Bold(true).
			Padding(0, 1)

	// Detail view styles
	DetailTitleStyle = lipgloss.NewStyle().
				Foreground(DraculaPink).
				Bold(true)
	DetailTaglineStyle = lipgloss.NewStyle().
				Foreground(DraculaCyan).
				Italic(true)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Foreground(DraculaComment)
	ErrorStyle = lipgloss.NewStyle().
			Foreground(DraculaRed)

	// Help
	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(DraculaPink).
			Bold(true)
	HelpDescStyle = lipgloss.NewStyle().
			Foreground(DraculaForeground)
)
