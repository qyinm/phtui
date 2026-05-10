package cli

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/qyinm/phtui/mcpsrv/dto"
	"github.com/qyinm/phtui/types"
)

type inspirationItem struct {
	Product           dto.Product `json:"product"`
	InspirationReason string      `json:"inspiration_reason"`
	FeatureSignals    []string    `json:"feature_signals"`
	MarketSignals     []string    `json:"market_signals"`
}

type ideasOutput struct {
	Source       string            `json:"source"`
	Period       string            `json:"period,omitempty"`
	Date         string            `json:"date,omitempty"`
	CategorySlug string            `json:"category_slug,omitempty"`
	Total        int               `json:"total"`
	Items        []inspirationItem `json:"items"`
}

// IsCommand reports whether arg is an agent-friendly non-interactive CLI command.
func IsCommand(arg string) bool {
	switch arg {
	case "ideas", "detail", "leaderboard", "help", "--help", "-h":
		return true
	default:
		return false
	}
}

// Run executes phtui's agent-friendly JSON CLI commands.
func Run(args []string, out io.Writer, source types.ProductSource) error {
	if len(args) == 0 || args[0] == "help" || args[0] == "--help" || args[0] == "-h" {
		writeUsage(out)
		return nil
	}

	switch args[0] {
	case "ideas":
		return runIdeas(args[1:], out, source)
	case "detail":
		return runDetail(args[1:], out, source)
	case "leaderboard":
		return runLeaderboard(args[1:], out, source)
	default:
		writeUsage(out)
		return fmt.Errorf("unknown command %q", args[0])
	}
}

func runIdeas(args []string, out io.Writer, source types.ProductSource) error {
	fs := flag.NewFlagSet("ideas", flag.ContinueOnError)
	fs.SetOutput(out)
	category := fs.String("category", "", "Product Hunt category slug")
	period := fs.String("period", "daily", "leaderboard period: daily, weekly, monthly")
	dateArg := fs.String("date", "", "leaderboard date YYYY-MM-DD")
	limit := fs.Int("limit", 5, "maximum products")
	if err := fs.Parse(args); err != nil {
		return err
	}

	max := normalizeLimit(*limit)
	if strings.TrimSpace(*category) != "" {
		products, _, err := source.GetCategoryProducts(strings.TrimSpace(*category))
		if err != nil {
			return err
		}
		products = applyLimit(products, max)
		return writeJSON(out, ideasOutput{
			Source:       "category",
			CategorySlug: strings.TrimSpace(*category),
			Total:        len(products),
			Items:        buildInspirationItems(products),
		})
	}

	p, err := parsePeriod(*period)
	if err != nil {
		return err
	}
	date, err := parseDate(*dateArg)
	if err != nil {
		return err
	}
	products, err := source.GetLeaderboard(p, date)
	if err != nil {
		return err
	}
	products = applyLimit(products, max)
	return writeJSON(out, ideasOutput{
		Source: "leaderboard",
		Period: p.String(),
		Date:   date.Format(time.DateOnly),
		Total:  len(products),
		Items:  buildInspirationItems(products),
	})
}

func runDetail(args []string, out io.Writer, source types.ProductSource) error {
	fs := flag.NewFlagSet("detail", flag.ContinueOnError)
	fs.SetOutput(out)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if fs.NArg() != 1 || strings.TrimSpace(fs.Arg(0)) == "" {
		return errors.New("detail requires a product slug")
	}
	detail, err := source.GetProductDetail(strings.TrimSpace(fs.Arg(0)))
	if err != nil {
		return err
	}
	return writeJSON(out, dto.FromProductDetail(detail))
}

func runLeaderboard(args []string, out io.Writer, source types.ProductSource) error {
	fs := flag.NewFlagSet("leaderboard", flag.ContinueOnError)
	fs.SetOutput(out)
	period := fs.String("period", "daily", "leaderboard period: daily, weekly, monthly")
	dateArg := fs.String("date", "", "leaderboard date YYYY-MM-DD")
	limit := fs.Int("limit", 10, "maximum products")
	if err := fs.Parse(args); err != nil {
		return err
	}
	p, err := parsePeriod(*period)
	if err != nil {
		return err
	}
	date, err := parseDate(*dateArg)
	if err != nil {
		return err
	}
	products, err := source.GetLeaderboard(p, date)
	if err != nil {
		return err
	}
	return writeJSON(out, map[string]any{
		"source": "leaderboard",
		"period": p.String(),
		"date":   date.Format(time.DateOnly),
		"total":  len(applyLimit(products, normalizeLimit(*limit))),
		"items":  dto.FromProducts(applyLimit(products, normalizeLimit(*limit))),
	})
}

func buildInspirationItems(products []types.Product) []inspirationItem {
	items := make([]inspirationItem, 0, len(products))
	for _, p := range products {
		items = append(items, inspirationItem{
			Product:           dto.FromProduct(p),
			InspirationReason: inspirationReason(p),
			FeatureSignals:    featureSignals(p),
			MarketSignals:     marketSignals(p),
		})
	}
	return items
}

func inspirationReason(p types.Product) string {
	if len(p.Categories()) > 0 {
		return fmt.Sprintf("Rank #%d with Product Hunt traction in %s; useful for service idea and MVP positioning.", p.Rank(), strings.Join(p.Categories(), ", "))
	}
	return fmt.Sprintf("Rank #%d Product Hunt launch; useful for service idea and MVP positioning.", p.Rank())
}

func featureSignals(p types.Product) []string {
	signals := make([]string, 0, 2+len(p.Categories()))
	if p.Tagline() != "" {
		signals = append(signals, "Positioning: "+p.Tagline())
	}
	for _, c := range p.Categories() {
		if strings.TrimSpace(c) != "" {
			signals = append(signals, "Category pattern: "+c)
		}
	}
	if len(signals) == 0 {
		signals = append(signals, "Open product detail for feature extraction")
	}
	return signals
}

func marketSignals(p types.Product) []string {
	signals := []string{fmt.Sprintf("Product Hunt rank #%d", p.Rank())}
	if p.VoteCount() > 0 {
		signals = append(signals, fmt.Sprintf("%d votes", p.VoteCount()))
	}
	if p.CommentCount() > 0 {
		signals = append(signals, fmt.Sprintf("%d comments", p.CommentCount()))
	}
	return signals
}

func parsePeriod(value string) (types.Period, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "daily":
		return types.Daily, nil
	case "weekly":
		return types.Weekly, nil
	case "monthly":
		return types.Monthly, nil
	default:
		return types.Daily, fmt.Errorf("invalid period %q", value)
	}
}

func parseDate(value string) (time.Time, error) {
	if strings.TrimSpace(value) == "" {
		return time.Now(), nil
	}
	return time.Parse(time.DateOnly, strings.TrimSpace(value))
}

func normalizeLimit(limit int) int {
	if limit <= 0 {
		return 5
	}
	if limit > 50 {
		return 50
	}
	return limit
}

func applyLimit(products []types.Product, limit int) []types.Product {
	if limit <= 0 || len(products) <= limit {
		return products
	}
	return products[:limit]
}

func writeJSON(out io.Writer, value any) error {
	encoder := json.NewEncoder(out)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func writeUsage(out io.Writer) {
	fmt.Fprintln(out, `phtui agent JSON commands:
  phtui ideas --category <slug> [--limit 5]
  phtui ideas --period daily|weekly|monthly [--date YYYY-MM-DD] [--limit 5]
  phtui detail <product-slug>
  phtui leaderboard --period daily|weekly|monthly [--date YYYY-MM-DD] [--limit 10]

Run phtui without a command to open the interactive TUI.`)
}
