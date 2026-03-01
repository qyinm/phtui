# @qxinm/phtui-mcp

Local stdio MCP launcher for `phtui`.

## Usage

```bash
npx -y @qxinm/phtui-mcp
```

The launcher runs:

```bash
go run github.com/qyinm/phtui/cmd/phtui-mcp-stdio@main
```

Make sure Go is installed and available in `PATH`.

For local testing from this repository, you can override the Go target:

```bash
PHTUI_MCP_GO_TARGET=./cmd/phtui-mcp-stdio npx -y @qxinm/phtui-mcp
```
