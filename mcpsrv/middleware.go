package mcpsrv

import (
	"net/http"
	"strings"
	"sync"
	"time"
)

func WrapMCPHandler(next http.Handler, cfg Config) http.Handler {
	rps := cfg.RPS
	if rps <= 0 {
		rps = 2
	}
	burst := cfg.Burst
	if burst <= 0 {
		burst = 5
	}

	allowedOrigins := make(map[string]struct{}, len(cfg.AllowedOrigins))
	for _, origin := range cfg.AllowedOrigins {
		allowedOrigins[origin] = struct{}{}
	}

	limiter := newTokenBucket(rps, burst)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := strings.TrimSpace(r.Header.Get("Origin"))
		if origin != "" {
			if len(allowedOrigins) == 0 {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			if _, ok := allowedOrigins[origin]; !ok {
				http.Error(w, "origin not allowed", http.StatusForbidden)
				return
			}
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Vary", "Origin")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept, Mcp-Protocol-Version, Mcp-Session-Id")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
		}

		if !limiter.Allow() {
			http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
			return
		}

		next.ServeHTTP(w, r)
	})
}

type tokenBucket struct {
	mu     sync.Mutex
	rps    float64
	burst  float64
	tokens float64
	last   time.Time
}

func newTokenBucket(rps float64, burst int) *tokenBucket {
	b := float64(burst)
	now := time.Now()
	return &tokenBucket{
		rps:    rps,
		burst:  b,
		tokens: b,
		last:   now,
	}
}

func (b *tokenBucket) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.last).Seconds()
	b.last = now
	b.tokens += elapsed * b.rps
	if b.tokens > b.burst {
		b.tokens = b.burst
	}
	if b.tokens < 1 {
		return false
	}
	b.tokens -= 1
	return true
}
