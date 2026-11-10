package config

import (
	"os"
	"strings"
)

// Config holds operator runtime settings.
type Config struct {
	DryRun bool

	// WatchNamespaces limits reconciliation; empty means all namespaces.
	WatchNamespaces []string
}

func envBool(key string, defaultVal bool) bool {
	v, ok := os.LookupEnv(key)
	if !ok {
		return defaultVal
	}
	return v != "false" && v != "0"
}

// Load reads configuration from the environment.
func Load() Config {
	cfg := Config{
		DryRun: envBool("DRY_RUN", false),
	}
	if ns := strings.TrimSpace(os.Getenv("WATCH_NAMESPACE")); ns != "" {
		cfg.WatchNamespaces = []string{ns}
	} else if legacy := strings.TrimSpace(os.Getenv("NAMESPACE")); legacy != "" {
		cfg.WatchNamespaces = []string{legacy}
	}
	return cfg
}
