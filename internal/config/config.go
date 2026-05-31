package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"gopkg.in/yaml.v3"
)

// Config is the top-level application configuration.
type Config struct {
	Server      ServerConfig      `yaml:"server"`
	Player      PlayerConfig      `yaml:"player"`
	Lyrics      LyricsConfig      `yaml:"lyrics"`
	Translation TranslationConfig `yaml:"translation"`
	Cache       CacheConfig       `yaml:"cache"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type PlayerConfig struct {
	PollIntervalMs int    `yaml:"poll_interval_ms"`
	PlayerctlPath  string `yaml:"playerctl_path"`
}

type LyricsConfig struct {
	Provider string       `yaml:"provider"`
	LRCLib   LRCLibConfig `yaml:"lrclib"`
}

type LRCLibConfig struct {
	BaseURL    string `yaml:"base_url"`
	TimeoutSec int    `yaml:"timeout_sec"`
}

type TranslationConfig struct {
	Provider    string              `yaml:"provider"`
	LibreTranslate LibreTranslateConfig `yaml:"libretranslate"`
	Romanization RomanizationConfig `yaml:"romanization"`
}

type LibreTranslateConfig struct {
	BaseURL    string `yaml:"base_url"`
	TimeoutSec int    `yaml:"timeout_sec"`
	APIKey     string `yaml:"api_key"`
}

type RomanizationConfig struct {
	Enabled   bool     `yaml:"enabled"`
	Languages []string `yaml:"languages"`
}

type CacheConfig struct {
	DBPath string `yaml:"db_path"`
}

// DefaultConfig returns a configuration with sensible defaults.
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			Host: "127.0.0.1",
			Port: 8090,
		},
		Player: PlayerConfig{
			PollIntervalMs: 500,
			PlayerctlPath:  "playerctl",
		},
		Lyrics: LyricsConfig{
			Provider: "lrclib",
			LRCLib: LRCLibConfig{
				BaseURL:    "https://lrclib.net/api",
				TimeoutSec: 15,
			},
		},
		Translation: TranslationConfig{
			Provider: "libretranslate",
			LibreTranslate: LibreTranslateConfig{
				BaseURL:    "http://127.0.0.1:5000",
				TimeoutSec: 30,
				APIKey:     "",
			},
			Romanization: RomanizationConfig{
				Enabled:   true,
				Languages: []string{"ja", "zh", "ko"},
			},
		},
		Cache: CacheConfig{
			DBPath: "~/.lyricsync/cache.db",
		},
	}
}

// applyEnvOverrides applies environment variable overrides to the config.
// Called after YAML loading so env vars always take precedence over file values.
// Supported env vars: LYRIC_HOST, LYRIC_PORT, LIBRETRANSLATE_URL, LIBRETRANSLATE_API_KEY, LYRIC_DB_PATH
func applyEnvOverrides(cfg *Config) {
	if v := os.Getenv("LYRIC_HOST"); v != "" {
		cfg.Server.Host = v
	}
	if v := os.Getenv("LYRIC_PORT"); v != "" {
		if p, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = p
		}
	}
	if v := os.Getenv("LIBRETRANSLATE_URL"); v != "" {
		cfg.Translation.LibreTranslate.BaseURL = v
	}
	if v := os.Getenv("LIBRETRANSLATE_API_KEY"); v != "" {
		cfg.Translation.LibreTranslate.APIKey = v
	}
	if v := os.Getenv("LYRIC_DB_PATH"); v != "" {
		cfg.Cache.DBPath = v
	}
}

// Load reads config from the given path. If path is empty, it tries
// default locations: ./config.yaml, ~/.config/lyricsync/config.yaml.
func Load(path string) (*Config, error) {
	cfg := DefaultConfig()

	if path == "" {
		path = findConfig()
	}

	if path == "" {
		applyEnvOverrides(cfg)
		return cfg, nil // no config file found, use defaults + env
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading config: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	// Apply env vars AFTER YAML — env beats file every time.
	applyEnvOverrides(cfg)

	// Expand ~ in DB path
	if strings.HasPrefix(cfg.Cache.DBPath, "~") {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("getting home dir: %w", err)
		}
		cfg.Cache.DBPath = filepath.Join(home, cfg.Cache.DBPath[1:])
	}

	return cfg, nil
}

func findConfig() string {
	candidates := []string{
		"config.yaml",
		"config.yml",
	}
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}
	// Check ~/.config/lyricsync/
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	for _, c := range candidates {
		p := filepath.Join(home, ".config", "lyricsync", c)
		if _, err := os.Stat(p); err == nil {
			return p
		}
	}
	return ""
}

// Address returns the listening address string.
func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}
