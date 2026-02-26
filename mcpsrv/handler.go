package mcpsrv

import (
	"net/http"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewHandler(server *mcp.Server, opts *mcp.StreamableHTTPOptions) http.Handler {
	return mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return server
	}, opts)
}
