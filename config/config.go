package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

type Config struct {
	AnthropicAPIKey string            `yaml:"anthropic_api_key"`
	VaultPath       string            `yaml:"vault_path"`
	VaultFolder     string            `yaml:"vault_folder"`
	CronSchedule    string            `yaml:"cron_schedule"`
	BatchSize       int               `yaml:"batch_size"`
	MaxPDFChars     int               `yaml:"max_pdf_chars"`
	Watchlist       []string          `yaml:"watchlist"`
	NewsSources     map[string]string `yaml:"news_sources"`
}

func Load(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open config: %w", err)
	}
	defer f.Close()

	var cfg Config
	if err := yaml.NewDecoder(f).Decode(&cfg); err != nil {
		return nil, fmt.Errorf("decode config: %w", err)
	}

	if cfg.CronSchedule == "" {
		cfg.CronSchedule = "0 7 * * 1-5"
	}
	if cfg.BatchSize == 0 {
		cfg.BatchSize = 20
	}
	if cfg.MaxPDFChars == 0 {
		cfg.MaxPDFChars = 3000
	}

	return &cfg, nil
}
