# phtui

Product Hunt TUI - browse the Product Hunt leaderboard from your terminal.

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)

## Features

- Browse Daily / Weekly / Monthly leaderboards
- Clickable date navigation bar (mouse support)
- Product detail view with ratings, reviews, and links
- Open products in your browser with `o`
- Vim-style keyboard navigation
- Dracula color theme (16-color ANSI)
- In-memory caching for fast navigation

## Install

```bash
go install github.com/qyinm/phtui@latest
```

Or build from source:

```bash
git clone https://github.com/qyinm/phtui.git
cd phtui
go build -o phtui .
./phtui
```

## Usage

```
phtui
```

### Key Bindings

| Key | Action |
|-----|--------|
| `j` / `k` | Navigate up/down |
| `Enter` | View product detail |
| `Esc` | Back to list |
| `Tab` | Cycle period (Daily/Weekly/Monthly) |
| `1` `2` `3` | Switch to Daily/Weekly/Monthly |
| `h` / `l` | Previous/next date |
| `o` | Open in browser |
| `r` | Refresh |
| `?` | Toggle help |
| `q` | Quit |

Mouse clicks are supported on the period tabs and date bar.

## Architecture

```
types/          Core types (Product, ProductDetail, ProductSource interface)
scraper/        HTTP scraper + HTML/SSR parser + cache
ui/             Bubbletea TUI (model, styles, keys, commands, delegate)
main.go         Entry point
```

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), [Lipgloss](https://github.com/charmbracelet/lipgloss), and [goquery](https://github.com/PuerkitoBio/goquery).

## License

MIT
