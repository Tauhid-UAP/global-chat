package config

import (
	"os"
)

type Config struct {
	Debug bool
	StaticAssetBaseURL string
}

func Load() *Config {
	debug := os.Getenv("DEBUG") == "true"

	staticAssetBaseURL := os.Getenv("STATIC_ASSET_BASE_URL")

	return &Config {
		Debug: debug,
		StaticAssetBaseURL: staticAssetBaseURL,
	}
}
