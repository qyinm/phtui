package ui

import (
	"fmt"
	"io"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/qyinm/phtui/types"
)

type ProductDelegate struct {
	base list.DefaultDelegate
}

func NewProductDelegate() ProductDelegate {
	d := list.NewDefaultDelegate()
	d.ShowDescription = false
	d.SetSpacing(0)
	d.SetHeight(3)
	return ProductDelegate{base: d}
}

// Height returns the height of a list item (3 lines)
func (d ProductDelegate) Height() int {
	return d.base.Height()
}

// Spacing returns the spacing between list items
func (d ProductDelegate) Spacing() int {
	return d.base.Spacing()
}

// Update handles updates for the delegate (no-op for products)
func (d ProductDelegate) Update(msg tea.Msg, m *list.Model) tea.Cmd {
	return d.base.Update(msg, m)
}

// Render renders a single product item using the shared renderProductItem.
func (d ProductDelegate) Render(w io.Writer, m list.Model, index int, item list.Item) {
	product, ok := item.(types.Product)
	if !ok {
		return
	}

	isSelected := index == m.Index()
	output := renderProductItem(product, isSelected, m.Width())
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
