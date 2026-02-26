package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

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
		EnableAdmin:  cfg.EnableAdmin && cfg.APIKey != "",
		APIKey:       cfg.APIKey,
	})

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})
	mcpHandler := mcpsrv.NewHandler(server, mcpsrv.StreamableOptions(cfg))
	mux.Handle("/mcp", mcpsrv.WrapMCPHandler(mcpHandler, cfg))

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

	httpServer := &http.Server{
		Addr:              ":" + strings.TrimSpace(cfg.Port),
		Handler:           mux,
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      0,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			log.Printf("shutdown error: %v", err)
		}
	}()

	log.Printf("phtui-mcp listening on %s", httpServer.Addr)
	err := httpServer.ListenAndServe()
	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		log.Fatalf("server failed: %v", err)
	}
}
