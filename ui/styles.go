package ui

import "github.com/charmbracelet/lipgloss"

// 16-color ANSI Dracula palette (identical to lazyadmin)
var (
	DraculaBackground = lipgloss.AdaptiveColor{Light: "0", Dark: "0"}
	DraculaForeground = lipgloss.AdaptiveColor{Light: "255", Dark: "255"}
	DraculaPurple     = lipgloss.AdaptiveColor{Light: "5", Dark: "5"}
	DraculaPink       = lipgloss.AdaptiveColor{Light: "13", Dark: "13"}
	DraculaCyan       = lipgloss.AdaptiveColor{Light: "14", Dark: "14"}
	DraculaGreen      = lipgloss.AdaptiveColor{Light: "10", Dark: "10"}
	DraculaComment    = lipgloss.AdaptiveColor{Light: "7", Dark: "7"}
	DraculaOrange     = lipgloss.AdaptiveColor{Light: "3", Dark: "3"}
	DraculaRed        = lipgloss.AdaptiveColor{Light: "1", Dark: "1"}

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

	SelectedItemStyle = lipgloss.NewStyle().
				BorderLeft(true).
				BorderStyle(lipgloss.NormalBorder()).
				BorderForeground(DraculaPink).
				PaddingLeft(1)

	// Date bar styles
	DateArrowStyle = lipgloss.NewStyle().
			Foreground(DraculaComment)
	DateItemStyle = lipgloss.NewStyle().
			Foreground(DraculaCyan)
	DateItemActiveStyle = lipgloss.NewStyle().
				Foreground(DraculaPink).
				Bold(true)
	DateItemDimStyle = lipgloss.NewStyle().
				Foreground(DraculaComment)
)
