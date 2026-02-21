package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	General       GeneralConfig       `toml:"general"`
	Notifications NotificationsConfig `toml:"notifications"`
}

type GeneralConfig struct {
	Interval int    `toml:"interval"`
	Timezone string `toml:"timezone"`
	Language string `toml:"language"`
}

type NotificationsConfig struct {
	Enabled bool `toml:"enabled"`
	Bell    bool `toml:"bell"`
}

func DefaultConfig() Config {
	return Config{
		General: GeneralConfig{
			Interval: 10,
			Timezone: "UTC",
			Language: "en",
		},
		Notifications: NotificationsConfig{
			Enabled: true,
			Bell:    true,
		},
	}
}

func DefaultPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return "config.toml"
	}
	return filepath.Join(home, ".config", "claude-smi", "config.toml")
}

func Load(path string) (Config, error) {
	cfg := DefaultConfig()
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil // use defaults
	}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return cfg, fmt.Errorf("decode config %s: %w", path, err)
	}
	return cfg, nil
}

func Save(cfg Config, path string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("create config dir: %w", err)
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("open config file: %w", err)
	}
	defer f.Close()
	if err := toml.NewEncoder(f).Encode(cfg); err != nil {
		return fmt.Errorf("encode config: %w", err)
	}
	return nil
}
