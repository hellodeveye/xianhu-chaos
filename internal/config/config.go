package config

import (
	"encoding/json"
	"fmt"
	"os"
)

type Config struct {
	Server struct {
		Addr string `json:"addr"`
	} `json:"server"`
	ProvidersDir    string `json:"providersDir"`
	RequestLogLimit int    `json:"requestLogLimit"`
}

func Load(path string) (Config, error) {
	var cfg Config
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, fmt.Errorf("read config %q: %w", path, err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return cfg, fmt.Errorf("parse config %q: %w", path, err)
	}
	if cfg.Server.Addr == "" {
		cfg.Server.Addr = ":18080"
	}
	if cfg.ProvidersDir == "" {
		cfg.ProvidersDir = "configs/providers"
	}
	if cfg.RequestLogLimit <= 0 {
		cfg.RequestLogLimit = 100
	}
	return cfg, nil
}
