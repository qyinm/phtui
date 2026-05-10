package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/qyinm/phtui/types"
)

func (m Model) renderProductList() string {
	available := m.height - 4 // tab + status + help
	if available < 1 {
		available = 1
	}

	itemHeight := 3
	visibleCount := available / itemHeight
	if visibleCount < 1 {
		visibleCount = 1
	}

	start := 0
	if m.selected >= visibleCount {
		start = m.selected - visibleCount + 1
	}
	end := start + visibleCount
	if end > len(m.products) {
		end = len(m.products)
		start = end - visibleCount
		if start < 0 {
			start = 0
		}
	}

	var b strings.Builder
	for i := start; i < end; i++ {
		b.WriteString(renderProductItem(m.products[i], i == m.selected, m.width))
		if i < end-1 {
			b.WriteString("\n")
		}
	}

	return b.String()
}

func renderProductItem(product types.Product, isSelected bool, width int) string {
	// Line 1: Rank + Name + Votes
	rankStr := fmt.Sprintf("#%-2d", product.Rank())
	nameStr := product.Name()
	voteDisplay := fmt.Sprintf("▲ %s", formatVoteCount(product.VoteCount()))

	rankWidth := lipgloss.Width(rankStr)
	voteWidth := lipgloss.Width(voteDisplay) + 1
	availableForName := width - rankWidth - voteWidth
	if availableForName <= 1 {
		availableForName = 0
	}
	nameStr = padOrTruncate(nameStr, availableForName)

	var line1 string
	if isSelected {
		rankStyle := lipgloss.NewStyle().Foreground(DraculaCyan).Bold(true)
		nameStyle := lipgloss.NewStyle().Foreground(DraculaPink).Bold(true)
		voteStyle := lipgloss.NewStyle().Foreground(DraculaGreen).Bold(true)
		line1 = lipgloss.JoinHorizontal(lipgloss.Left, rankStyle.Render(rankStr), nameStyle.Render(nameStr), voteStyle.Render(voteDisplay))
	} else {
		rankStyle := lipgloss.NewStyle().Foreground(DraculaComment)
		nameStyle := lipgloss.NewStyle().Foreground(DraculaCyan)
		voteStyle := lipgloss.NewStyle().Foreground(DraculaGreen)
		line1 = lipgloss.JoinHorizontal(lipgloss.Left, rankStyle.Render(rankStr), nameStyle.Render(nameStr), voteStyle.Render(voteDisplay))
	}

	// Line 2: Tagline
	tagline := product.Tagline()
	taglineIndent := "    "
	taglineAvailable := width - lipgloss.Width(taglineIndent)
	if taglineAvailable < 0 {
		taglineAvailable = 0
	}
	tagline = truncateToWidth(tagline, taglineAvailable)
	line2 := taglineIndent + lipgloss.NewStyle().Foreground(DraculaForeground).Render(tagline)

	// Line 3: Categories
	categoryStr := strings.Join(product.Categories(), " • ")
	categoryIndent := "    "
	categoryAvailable := width - lipgloss.Width(categoryIndent)
	if categoryAvailable < 0 {
		categoryAvailable = 0
	}
	categoryStr = truncateToWidth(categoryStr, categoryAvailable)
	line3 := categoryIndent + lipgloss.NewStyle().Foreground(DraculaComment).Render(categoryStr)

	output := line1 + "\n" + line2 + "\n" + line3
	if isSelected {
		return SelectedItemStyle.Render(output)
	}
	return output
}

// renderDetailContent formats ProductDetail for the viewport
func (m Model) renderDetailContent() string {
	d := m.detail
	p := d.Product()

	var b strings.Builder

	b.WriteString(DetailTitleStyle.Render(p.Name()))
	b.WriteString("\n")
	b.WriteString(DetailTaglineStyle.Render(p.Tagline()))
	b.WriteString("\n\n")

	stats := fmt.Sprintf("⭐ %.1f (%d reviews) • %s followers",
		d.Rating(), d.ReviewCount(), formatVoteCount(d.FollowerCount()))
	b.WriteString(stats)
	b.WriteString("\n")

	if !d.LaunchDate().IsZero() {
		b.WriteString(fmt.Sprintf("🚀 Launched: %s\n", d.LaunchDate().Format("January 2, 2006")))
	}

	if d.MakerName() != "" {
		maker := fmt.Sprintf("👤 Maker: %s", d.MakerName())
		if d.MakerProfileURL() != "" {
			maker += fmt.Sprintf(" (%s)", d.MakerProfileURL())
		}
		b.WriteString(maker + "\n")
	}

	if d.PricingInfo() != "" {
		b.WriteString(fmt.Sprintf("💰 %s\n", d.PricingInfo()))
	}

	if d.WebsiteURL() != "" {
		b.WriteString(fmt.Sprintf("🌐 %s\n", d.WebsiteURL()))
	}

	b.WriteString("\n")

	if d.Description() != "" {
		b.WriteString(d.Description())
		b.WriteString("\n")
	}

	if d.MakerComment() != "" {
		b.WriteString("\n--- Maker Comment ---\n")
		b.WriteString(d.MakerComment())
		b.WriteString("\n")
	}

	if len(d.ProConTags()) > 0 {
		var pros, cons, others []string
		for _, tag := range d.ProConTags() {
			label := fmt.Sprintf("%s (%d)", tag.Name(), tag.Count())
			switch tag.TagType() {
			case "Positive":
				pros = append(pros, label)
			case "Negative":
				cons = append(cons, label)
			default:
				others = append(others, label)
			}
		}
		if len(pros) > 0 {
			b.WriteString("\n👍 Pros:\n")
			for _, p := range pros {
				b.WriteString("  + " + p + "\n")
			}
		}
		if len(cons) > 0 {
			b.WriteString("\n👎 Cons:\n")
			for _, c := range cons {
				b.WriteString("  - " + c + "\n")
			}
		}
		if len(others) > 0 {
			b.WriteString("\nℹ️ Other:\n")
			for _, o := range others {
				b.WriteString("  * " + o + "\n")
			}
		}
	}

	if len(d.Categories()) > 0 {
		catStyle := lipgloss.NewStyle().Foreground(DraculaCyan).Underline(true)
		b.WriteString("\nCategories: ")
		for i, cat := range d.Categories() {
			if i > 0 {
				b.WriteString(" • ")
			}
			b.WriteString(catStyle.Render(cat))
		}
		b.WriteString("  (press 4 to browse categories)")
		b.WriteString("\n")
	}

	if len(d.SocialLinks()) > 0 {
		b.WriteString("\nSocial:\n")
		for _, link := range d.SocialLinks() {
			b.WriteString("- ")
			b.WriteString(link)
			b.WriteString("\n")
		}
	}

	return b.String()
}
