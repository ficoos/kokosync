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
	KomgaAPIRoot  *url.URL
	KomgaAPIKey   string
	ListenAddress string
}

func ConfigFromEnvironment() (*Config, error) {
	dbPath := strings.TrimSpace(os.Getenv(EnvPrefix + "DB"))
	if len(dbPath) == 0 {
		dbPath = "./data.db"
	}
	rawKomgaAPIRoot := strings.TrimSpace(os.Getenv(EnvPrefix + "KOMGA_API_ROOT"))
	komgaAPIKey := strings.TrimSpace(os.Getenv(EnvPrefix + "KOMGA_API_KEY"))
	listenAddress := strings.TrimSpace(os.Getenv(EnvPrefix + "KOMGA_LISTEN_ADDRESS"))
	if listenAddress == "" {
		listenAddress = "0.0.0.0:8889"
	}

	komgaAPIRoot, err := url.Parse(rawKomgaAPIRoot)
	if err != nil {
		return nil, fmt.Errorf("parse komga api root: %s", err)
	}

	return &Config{
		DBPath:        dbPath,
		KomgaAPIRoot:  komgaAPIRoot,
		KomgaAPIKey:   komgaAPIKey,
		ListenAddress: listenAddress,
	}, nil
}
