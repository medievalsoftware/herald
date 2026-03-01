package config

import (
	"errors"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const defaultTimestamp = "15:04"

// Config holds application configuration loaded from TOML.
type Config struct {
	Timestamp string `toml:"timestamp"`
}

// Load reads config from $XDG_CONFIG_HOME/herald/config.toml, falling back to
// ~/.config/herald/config.toml. A missing file returns defaults; a malformed
// file returns an error.
func Load() (Config, error) {
	cfg := Config{
		Timestamp: defaultTimestamp,
	}

	path := configPath()
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return cfg, nil
		}
		return Config{}, err
	}

	if cfg.Timestamp == "" {
		cfg.Timestamp = defaultTimestamp
	}

	return cfg, nil
}

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "herald", "config.toml")
}
