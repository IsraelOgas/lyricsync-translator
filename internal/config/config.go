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
	Server      ServerConfig      `yaml:"server" json:"server"`
	Player      PlayerConfig      `yaml:"player" json:"player"`
	Lyrics      LyricsConfig      `yaml:"lyrics" json:"lyrics"`
	Translation TranslationConfig `yaml:"translation" json:"translation"`
	Cache       CacheConfig       `yaml:"cache" json:"cache"`

	loadedFrom string `yaml:"-" json:"-"` // path the config was loaded from (internal, never serialized)
}

// LoadedFrom returns the path the config was loaded from, or "" if defaults were used.
func (c *Config) LoadedFrom() string { return c.loadedFrom }

type ServerConfig struct {
	Host string `yaml:"host" json:"host"`
	Port int    `yaml:"port" json:"port"`
}

type PlayerConfig struct {
	PollIntervalMs int    `yaml:"poll_interval_ms" json:"poll_interval_ms"`
	PlayerctlPath  string `yaml:"playerctl_path" json:"playerctl_path"`
}

type LyricsConfig struct {
	Provider string       `yaml:"provider" json:"provider"`
	LRCLib   LRCLibConfig `yaml:"lrclib" json:"lrclib"`
}

type LRCLibConfig struct {
	BaseURL    string `yaml:"base_url" json:"base_url"`
	TimeoutSec int    `yaml:"timeout_sec" json:"timeout_sec"`
}

type DeepSeekConfig struct {
	BaseURL    string `yaml:"base_url" json:"base_url"`
	APIKey     string `yaml:"api_key" json:"api_key"`
	Model      string `yaml:"model" json:"model"`
	TimeoutSec int    `yaml:"timeout_sec" json:"timeout_sec"`
}

type TranslationConfig struct {
	Provider       string               `yaml:"provider" json:"provider"`
	TargetLang     string               `yaml:"target_lang" json:"target_lang"`
	LibreTranslate LibreTranslateConfig `yaml:"libretranslate" json:"libretranslate"`
	DeepSeek       DeepSeekConfig       `yaml:"deepseek" json:"deepseek"`
	Romanization   RomanizationConfig   `yaml:"romanization" json:"romanization"`
}

type LibreTranslateConfig struct {
	BaseURL    string `yaml:"base_url" json:"base_url"`
	TimeoutSec int    `yaml:"timeout_sec" json:"timeout_sec"`
	APIKey     string `yaml:"api_key" json:"api_key"`
}

type RomanizationConfig struct {
	Enabled   bool     `yaml:"enabled" json:"enabled"`
	Languages []string `yaml:"languages" json:"languages"`
}

type CacheConfig struct {
	DBPath string `yaml:"db_path" json:"db_path"`
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
			Provider:   "libretranslate",
			TargetLang: "es",
			LibreTranslate: LibreTranslateConfig{
				BaseURL:    "http://127.0.0.1:5000",
				TimeoutSec: 30,
				APIKey:     "",
			},
			DeepSeek: DeepSeekConfig{
				BaseURL:    "https://api.deepseek.com",
				Model:      "deepseek-v4-flash",
				TimeoutSec: 60,
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
// Supported env vars: LYRIC_HOST, LYRIC_PORT, LIBRETRANSLATE_URL, LIBRETRANSLATE_API_KEY, DEEPSEEK_API_KEY, LYRIC_DB_PATH
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
	if v := os.Getenv("DEEPSEEK_API_KEY"); v != "" {
		cfg.Translation.DeepSeek.APIKey = v
	}
	if v := os.Getenv("LYRIC_TARGET_LANG"); v != "" {
		cfg.Translation.TargetLang = v
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

	cfg.loadedFrom = path

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
	candidates := []string{"config.yaml", "config.yml"}

	// User config dir takes priority — secrets live here.
	home, err := os.UserHomeDir()
	if err == nil {
		for _, c := range candidates {
			p := filepath.Join(home, ".config", "lyricsync", c)
			if _, err := os.Stat(p); err == nil {
				return p
			}
		}
	}

	// Fall back to repo-local config.yaml (template, no secrets).
	for _, c := range candidates {
		if _, err := os.Stat(c); err == nil {
			return c
		}
	}

	return ""
}

// Save writes the config to the user config directory (~/.config/lyricsync/config.yaml).
// It never writes back to a repo-local config.yaml to avoid accidentally
// committing secrets. The repo-local file is treated as a read-only template.
func Save(cfg *Config) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return fmt.Errorf("getting home dir for config save: %w", err)
	}
	dir := filepath.Join(home, ".config", "lyricsync")
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating config dir: %w", err)
	}
	path := filepath.Join(dir, "config.yaml")

	data, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}

// SanitizedForAPI returns a shallow copy with sensitive fields masked.
// Safe to serialize as JSON and expose to the frontend.
func (c *Config) SanitizedForAPI() *Config {
	safe := *c
	safe.Translation.DeepSeek.APIKey = maskKey(c.Translation.DeepSeek.APIKey)
	safe.Translation.LibreTranslate.APIKey = maskKey(c.Translation.LibreTranslate.APIKey)
	return &safe
}

// maskKey returns a masked version of an API key for display.
func maskKey(key string) string {
	if key == "" {
		return ""
	}
	return "••••••••"
}

// Address returns the listening address string.
func (s ServerConfig) Address() string {
	return fmt.Sprintf("%s:%d", s.Host, s.Port)
}

// APIBase returns the full HTTP base URL for API calls.
func (s ServerConfig) APIBase() string {
	return fmt.Sprintf("http://%s:%d", s.Host, s.Port)
}

// WindowConfig stores persisted window position, size, and fullscreen state.
type WindowConfig struct {
	X          int  `yaml:"x"`
	Y          int  `yaml:"y"`
	Width      int  `yaml:"width"`
	Height     int  `yaml:"height"`
	Fullscreen bool `yaml:"fullscreen"`
}

// windowStatePath returns ~/.lyricsync/window-state.yaml.
func windowStatePath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("getting home dir: %w", err)
	}
	return filepath.Join(home, ".lyricsync", "window-state.yaml"), nil
}

// LoadWindowState reads window state from ~/.lyricsync/window-state.yaml.
// Returns defaults (centered 1024×768) if the file doesn't exist.
func LoadWindowState() (*WindowConfig, error) {
	path, err := windowStatePath()
	if err != nil {
		return nil, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &WindowConfig{Width: 1024, Height: 768}, nil
		}
		return nil, fmt.Errorf("reading window state: %w", err)
	}

	var wc WindowConfig
	if err := yaml.Unmarshal(data, &wc); err != nil {
		return nil, fmt.Errorf("parsing window state: %w", err)
	}
	return &wc, nil
}

// SaveWindowState writes window state to ~/.lyricsync/window-state.yaml.
// Creates parent directories if they don't exist.
func SaveWindowState(wc *WindowConfig) error {
	path, err := windowStatePath()
	if err != nil {
		return err
	}

	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating window state dir: %w", err)
	}

	data, err := yaml.Marshal(wc)
	if err != nil {
		return fmt.Errorf("marshaling window state: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}
