package ui

import (
	"fmt"
	"io"
	"strings"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/qyinm/phtui/types"
)

// ProductDelegate is a custom list delegate for rendering Product items
type ProductDelegate struct{}

// Height returns the height of a list item (3 lines)
func (d ProductDelegate) Height() int {
	return 3
}

// Spacing returns the spacing between list items
func (d ProductDelegate) Spacing() int {
	return 0
}

// Update handles updates for the delegate (no-op for products)
func (d ProductDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return nil
}

// Render renders a single product item
func (d ProductDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	product, ok := item.(types.Product)
	if !ok {
		return
	}

	// Determine if this item is selected
	isSelected := index == m.Index()

	// Line 1: Rank + Name + Vote Count
	rankStr := fmt.Sprintf("#%-2d", product.Rank())
	nameStr := product.Name()

	// Format vote count with K/M suffixes
	voteStr := formatVoteCount(product.VoteCount())
	voteDisplay := fmt.Sprintf("▲ %s", voteStr)

	// Calculate available width for name (accounting for rank, spacing, and vote count)
	// Format: "#1  Name                                    ▲ 1,422"
	rankWidth := 4                    // "#1  "
	voteWidth := len(voteDisplay) + 1 // " ▲ 1,422"
	availableForName := m.Width() - rankWidth - voteWidth

	if availableForName < 0 {
		availableForName = 0
	}

	// Truncate or pad name
	if len(nameStr) > availableForName {
		nameStr = nameStr[:availableForName-1] + "…"
	} else {
		nameStr = nameStr + strings.Repeat(" ", availableForName-len(nameStr))
	}

	// Style line 1
	var line1 string
	if isSelected {
		rankStyle := lipgloss.NewStyle().Foreground(DraculaCyan).Bold(true)
		nameStyle := lipgloss.NewStyle().Foreground(DraculaPink).Bold(true)
		voteStyle := lipgloss.NewStyle().Foreground(DraculaGreen).Bold(true)
		line1 = rankStyle.Render(rankStr) + nameStyle.Render(nameStr) + voteStyle.Render(voteDisplay)
	} else {
		rankStyle := lipgloss.NewStyle().Foreground(DraculaComment)
		nameStyle := lipgloss.NewStyle().Foreground(DraculaCyan)
		voteStyle := lipgloss.NewStyle().Foreground(DraculaGreen)
		line1 = rankStyle.Render(rankStr) + nameStyle.Render(nameStr) + voteStyle.Render(voteDisplay)
	}

	// Line 2: Tagline (indented)
	tagline := product.Tagline()
	taglineIndent := "    "
	taglineAvailable := m.Width() - len(taglineIndent)

	if taglineAvailable < 0 {
		taglineAvailable = 0
	}

	if len(tagline) > taglineAvailable {
		tagline = tagline[:taglineAvailable-3] + "…"
	}

	var line2 string
	if isSelected {
		taglineStyle := lipgloss.NewStyle().Foreground(DraculaForeground)
		line2 = taglineIndent + taglineStyle.Render(tagline)
	} else {
		taglineStyle := lipgloss.NewStyle().Foreground(DraculaForeground)
		line2 = taglineIndent + taglineStyle.Render(tagline)
	}

	// Line 3: Categories (indented, dimmed)
	categories := product.Categories()
	categoryStr := strings.Join(categories, " • ")
	categoryIndent := "    "
	categoryAvailable := m.Width() - len(categoryIndent)

	if categoryAvailable < 0 {
		categoryAvailable = 0
	}

	if len(categoryStr) > categoryAvailable {
		categoryStr = categoryStr[:categoryAvailable-3] + "…"
	}

	var line3 string
	if isSelected {
		categoryStyle := lipgloss.NewStyle().Foreground(DraculaComment)
		line3 = categoryIndent + categoryStyle.Render(categoryStr)
	} else {
		categoryStyle := lipgloss.NewStyle().Foreground(DraculaComment)
		line3 = categoryIndent + categoryStyle.Render(categoryStr)
	}

	// Combine lines and write to output
	output := line1 + "\n" + line2 + "\n" + line3
	fmt.Fprint(w, output)
}

// formatVoteCount formats vote count with K/M suffixes
// 1000 -> "1.0K", 1422 -> "1.4K", 1000000 -> "1.0M"
func formatVoteCount(count int) string {
	if count >= 1000000 {
		return fmt.Sprintf("%.1fM", float64(count)/1000000)
	}
	if count >= 1000 {
		return fmt.Sprintf("%.1fK", float64(count)/1000)
	}
	return fmt.Sprintf("%d", count)
}
