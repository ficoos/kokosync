package main

import (
	"fmt"
	"net/url"
	"os"
	"strings"
)

const EnvPrefix = "MKSYNC_"

type Config struct {
	DBPath        string
	UpstreamURL   *url.URL
	ListenAddress string
}

func ConfigFromEnvironment() (*Config, error) {
	dbPath := strings.TrimSpace(os.Getenv(EnvPrefix + "DB"))
	if len(dbPath) == 0 {
		dbPath = "./data.db"
	}
	rawUpstreamAPIRoot := strings.TrimSpace(os.Getenv(EnvPrefix + "UPSTREAM_API_ROOT"))
	listenAddress := strings.TrimSpace(os.Getenv(EnvPrefix + "LISTEN_ADDRESS"))
	if listenAddress == "" {
		listenAddress = "127.0.0.1:8889"
	}

	upstreamURL, err := url.Parse(rawUpstreamAPIRoot)
	if err != nil {
		return nil, fmt.Errorf("parse komga api root: %s", err)
	}

	return &Config{
		DBPath:        dbPath,
		UpstreamURL:   upstreamURL,
		ListenAddress: listenAddress,
	}, nil
}
