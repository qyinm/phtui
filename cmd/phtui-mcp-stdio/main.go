package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/qyinm/phtui/mcpsrv"
	"github.com/qyinm/phtui/scraper"
)

type cacheClearSource interface {
	ClearCache()
}

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg := mcpsrv.LoadConfig()
	source := scraper.New()
	server := mcpsrv.NewServer(source, "dev", &mcpsrv.ServerOptions{
		EnableSearch: cfg.EnableSearch,
		EnableAdmin:  cfg.EnableAdmin,
	})

	if cfg.CacheClearInterval > 0 {
		if clearable, ok := any(source).(cacheClearSource); ok {
			go func() {
				ticker := time.NewTicker(cfg.CacheClearInterval)
				defer ticker.Stop()
				for {
					select {
					case <-ticker.C:
						clearable.ClearCache()
					case <-ctx.Done():
						return
					}
				}
			}()
		}
	}

	if err := server.Run(ctx, &mcp.StdioTransport{}); err != nil {
		log.Fatalf("stdio mcp server failed: %v", err)
	}
}
