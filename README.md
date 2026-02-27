# phtui

Product Hunt TUI - browse the Product Hunt leaderboard from your terminal.

![phtui logo](assets/logo.png)

![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go)

## Features

- Browse Daily / Weekly / Monthly leaderboards
- Browse by category (248 categories with search/filter)
- Clickable date navigation bar (mouse support)
- Product detail view with ratings, reviews, pros/cons, pricing, and links
- Open products in your browser with `o`
- Vim-style keyboard navigation
- Dracula color theme (16-color ANSI)
- In-memory caching for fast navigation

## Install

### Homebrew

```bash
brew install qyinm/tap/phtui
```

### Go

```bash
go install github.com/qyinm/phtui@latest
```

### Build from source

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
| `Tab` | Cycle period (Daily/Weekly/Monthly/Categories) |
| `1` `2` `3` `4` | Switch to Daily/Weekly/Monthly/Categories |
| `h` / `l` | Previous/next date (or category) |
| `/` | Search (global product search, or filter categories) |
| `o` | Open in browser |
| `r` | Refresh |
| `?` | Toggle help |
| `q` | Quit |

Mouse clicks are supported on the period tabs and date bar.
Use `/` to open search input, type a query, then press `Enter` to run global search.
Press `4` or `Tab` to open the category selector, browse with `j`/`k`, and press `Enter` to view products. Use `/` to filter categories by name.

## Architecture

```
types/          Core types (Product, ProductDetail, ProductSource interface)
scraper/        HTTP scraper + HTML/SSR parser + cache
ui/             Bubbletea TUI (model, styles, keys, commands, delegate)
main.go         Entry point
```

Built with [Bubbletea](https://github.com/charmbracelet/bubbletea), [Bubbles](https://github.com/charmbracelet/bubbles), [Lipgloss](https://github.com/charmbracelet/lipgloss), and [goquery](https://github.com/PuerkitoBio/goquery).

## MCP Server

v1 is local-first. Run MCP server on localhost and connect agents to it.

Run:

```bash
PORT=8080 go run ./cmd/phtui-mcp
```

Endpoints:

- MCP: `http://localhost:8080/mcp`
- Health: `http://localhost:8080/healthz`

Quick local test:

```bash
curl -i http://localhost:8080/healthz
```

Core tools enabled by default (v1):

- `leaderboard_get`
- `product_get_detail`
- `category_list`
- `category_get_products`

Optional tools (off by default):

- `search_products` (`PHTUI_MCP_ENABLE_SEARCH=true`)
- `cache_clear` (`PHTUI_MCP_ENABLE_ADMIN=true` and `PHTUI_MCP_API_KEY` set)

Local client setup examples:

One-command setup script:

```bash
./scripts/install-mcp-local.sh
```

Options:

```bash
./scripts/install-mcp-local.sh --codex-only
./scripts/install-mcp-local.sh --claude-only
./scripts/install-mcp-local.sh --name phtui-local --url http://localhost:8080/mcp
```

### Codex (local)

```bash
codex mcp remove phtui-local
codex mcp add phtui-local --url http://localhost:8080/mcp
```

### Claude Code (local)

```bash
claude mcp add -t http phtui-local http://localhost:8080/mcp
```

Environment variables:

| Variable | Default | Description |
|---|---|---|
| `PORT` | `8080` | HTTP listen port |
| `PHTUI_MCP_API_KEY` | empty | Enables auth when set (`Authorization: Bearer <key>` or `X-API-Key`) |
| `PHTUI_MCP_ALLOWED_ORIGINS` | empty | Comma-separated allowed `Origin` values |
| `PHTUI_MCP_STATELESS` | `false` | Stateless streamable mode (`GET /mcp` returns `405`) |
| `PHTUI_MCP_ENABLE_SEARCH` | `false` | Enable `search_products` tool |
| `PHTUI_MCP_ENABLE_ADMIN` | `false` | Enable admin tool `cache_clear` (requires API key) |
| `PHTUI_MCP_RPS` | `2` | Global rate-limit tokens per second |
| `PHTUI_MCP_BURST` | `5` | Global rate-limit burst |
| `PHTUI_MCP_SESSION_TIMEOUT` | `15m` | Stateful session idle timeout |
| `PHTUI_MCP_CACHE_CLEAR_INTERVAL` | `30m` | Periodic scraper cache clear; `0` disables |

Remote deployment is optional and out of v1 scope. If you deploy publicly, enable `PHTUI_MCP_API_KEY` and use Bearer auth.

## License

MIT
