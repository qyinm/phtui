package mcpsrv

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type Config struct {
	Port               string
	AllowedOrigins     []string
	Stateless          bool
	EnableSearch       bool
	EnableAdmin        bool
	RPS                float64
	Burst              int
	SessionTimeout     time.Duration
	CacheClearInterval time.Duration
}

func LoadConfig() Config {
	port := strings.TrimSpace(os.Getenv("PORT"))
	if port == "" {
		port = "8080"
	}

	cfg := Config{
		Port:               port,
		AllowedOrigins:     parseCSV(os.Getenv("PHTUI_MCP_ALLOWED_ORIGINS")),
		Stateless:          parseBool(os.Getenv("PHTUI_MCP_STATELESS"), false),
		EnableSearch:       parseBool(os.Getenv("PHTUI_MCP_ENABLE_SEARCH"), false),
		EnableAdmin:        parseBool(os.Getenv("PHTUI_MCP_ENABLE_ADMIN"), false),
		RPS:                parseFloat(os.Getenv("PHTUI_MCP_RPS"), 2),
		Burst:              parseInt(os.Getenv("PHTUI_MCP_BURST"), 5),
		SessionTimeout:     parseDuration(os.Getenv("PHTUI_MCP_SESSION_TIMEOUT"), 15*time.Minute),
		CacheClearInterval: parseDuration(os.Getenv("PHTUI_MCP_CACHE_CLEAR_INTERVAL"), 30*time.Minute),
	}

	if cfg.RPS <= 0 {
		cfg.RPS = 2
	}
	if cfg.Burst <= 0 {
		cfg.Burst = 5
	}

	return cfg
}

func StreamableOptions(cfg Config) *mcp.StreamableHTTPOptions {
	return &mcp.StreamableHTTPOptions{
		Stateless:      cfg.Stateless,
		SessionTimeout: cfg.SessionTimeout,
	}
}

func parseCSV(raw string) []string {
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		v := strings.TrimSpace(p)
		if v != "" {
			out = append(out, v)
		}
	}
	return out
}

func parseBool(raw string, fallback bool) bool {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func parseInt(raw string, fallback int) int {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return fallback
	}
	return n
}

func parseFloat(raw string, fallback float64) float64 {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fallback
	}
	n, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return fallback
	}
	return n
}

func parseDuration(raw string, fallback time.Duration) time.Duration {
	v := strings.TrimSpace(raw)
	if v == "" {
		return fallback
	}
	d, err := time.ParseDuration(v)
	if err != nil {
		return fallback
	}
	return d
}
