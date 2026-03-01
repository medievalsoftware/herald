package config

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
)

const (
	defaultTimestamp    = "3:04 PM"
	defaultUsersWidth   = 20
	defaultHistoryLimit = 100
)

// Theme holds color settings for UI elements. Values are lipgloss color
// strings — ANSI numbers ("12") or hex ("#83a598").
type Theme struct {
	BarBg  string   `toml:"bar_bg"`
	BarFg  string   `toml:"bar_fg"`
	Border string   `toml:"border"`
	Accent string   `toml:"accent"`
	Green  string   `toml:"green"`
	Yellow string   `toml:"yellow"`
	Nicks  []string `toml:"nicks"`
}

// DefaultTheme returns the built-in ANSI color defaults.
func DefaultTheme() Theme {
	return Theme{
		BarBg:  "235",
		BarFg:  "252",
		Border: "240",
		Accent: "12",
		Green:  "10",
		Yellow: "11",
		Nicks: []string{
			"1", "2", "3", "4", "5", "6",
			"9", "10", "11", "12", "13", "14",
		},
	}
}

// KeysConfig holds user key binding overrides in Helix notation.
type KeysConfig struct {
	Normal map[string]string `toml:"normal"`
	Insert map[string]string `toml:"insert"`
}

// Config holds application configuration loaded from TOML.
type Config struct {
	Timestamp    string     `toml:"timestamp"`
	UsersWidth   int        `toml:"users_width"`
	HistoryLimit int        `toml:"history_limit"`
	Theme        string     `toml:"theme,omitempty"`
	Keys         KeysConfig `toml:"keys"`
}

// Load reads config from $XDG_CONFIG_HOME/herald/config.toml, falling back to
// ~/.config/herald/config.toml. A missing file returns defaults; a malformed
// file returns an error.
func Load() (Config, error) {
	cfg := Config{
		Timestamp:    defaultTimestamp,
		UsersWidth:   defaultUsersWidth,
		HistoryLimit: defaultHistoryLimit,
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
	if cfg.UsersWidth <= 0 {
		cfg.UsersWidth = defaultUsersWidth
	}
	if cfg.HistoryLimit <= 0 {
		cfg.HistoryLimit = defaultHistoryLimit
	}

	return cfg, nil
}

// Setting describes a configurable option.
type Setting struct {
	Name string
	Desc string
}

// AvailableSettings returns metadata for all configurable settings.
func AvailableSettings() []Setting {
	return []Setting{
		{Name: "timestamp", Desc: "Time format for messages"},
		{Name: "users_width", Desc: "Width of the users panel"},
		{Name: "history_limit", Desc: "Messages to fetch on join"},
	}
}

// Get returns the current value of a setting by name.
func (c *Config) Get(key string) string {
	switch key {
	case "timestamp":
		return c.Timestamp
	case "users_width":
		return strconv.Itoa(c.UsersWidth)
	case "history_limit":
		return strconv.Itoa(c.HistoryLimit)
	default:
		return ""
	}
}

// Set updates a setting by name, validating the value.
func (c *Config) Set(key, value string) error {
	switch key {
	case "timestamp":
		if value == "" {
			return fmt.Errorf("timestamp cannot be empty")
		}
		c.Timestamp = value
	case "users_width":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("users_width must be a positive integer")
		}
		c.UsersWidth = n
	case "history_limit":
		n, err := strconv.Atoi(value)
		if err != nil || n <= 0 {
			return fmt.Errorf("history_limit must be a positive integer")
		}
		c.HistoryLimit = n
	default:
		return fmt.Errorf("unknown setting: %s", key)
	}
	return nil
}

// Save writes the current config back to the TOML file.
func (c Config) Save() error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	encErr := toml.NewEncoder(f).Encode(c)
	if closeErr := f.Close(); encErr == nil {
		encErr = closeErr
	}
	return encErr
}

// ThemesDir returns the path to the themes directory.
func ThemesDir() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "herald", "themes")
}

// LoadTheme loads a named theme from the themes directory. If name is empty,
// DefaultTheme is returned. Zero-value fields are filled from defaults.
func LoadTheme(name string) (Theme, error) {
	if name == "" {
		return DefaultTheme(), nil
	}

	path := filepath.Join(ThemesDir(), name+".toml")
	var t Theme
	if _, err := toml.DecodeFile(path, &t); err != nil {
		return Theme{}, fmt.Errorf("loading theme %q: %w", name, err)
	}

	defaults := DefaultTheme()
	if t.BarBg == "" {
		t.BarBg = defaults.BarBg
	}
	if t.BarFg == "" {
		t.BarFg = defaults.BarFg
	}
	if t.Border == "" {
		t.Border = defaults.Border
	}
	if t.Accent == "" {
		t.Accent = defaults.Accent
	}
	if t.Green == "" {
		t.Green = defaults.Green
	}
	if t.Yellow == "" {
		t.Yellow = defaults.Yellow
	}
	if len(t.Nicks) == 0 {
		t.Nicks = defaults.Nicks
	}

	return t, nil
}

// AvailableThemes returns the names of theme files in the themes directory,
// sorted alphabetically. Returns nil if the directory is missing or empty.
func AvailableThemes() []string {
	entries, err := os.ReadDir(ThemesDir())
	if err != nil {
		return nil
	}

	var names []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if n, ok := strings.CutSuffix(e.Name(), ".toml"); ok {
			names = append(names, n)
		}
	}
	sort.Strings(names)
	return names
}

func configPath() string {
	dir := os.Getenv("XDG_CONFIG_HOME")
	if dir == "" {
		home, _ := os.UserHomeDir()
		dir = filepath.Join(home, ".config")
	}
	return filepath.Join(dir, "herald", "config.toml")
}
