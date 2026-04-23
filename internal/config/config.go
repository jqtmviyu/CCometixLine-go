package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Style    StyleConfig     `toml:"style"`
	Segments []SegmentConfig `toml:"segments"`
	Theme    string          `toml:"theme"`
}

type StyleConfig struct {
	Mode      StyleMode `toml:"mode"`
	Separator string    `toml:"separator"`
}

type StyleMode string

const (
	StyleModePlain     StyleMode = "plain"
	StyleModeNerdFont  StyleMode = "nerd_font"
	StyleModePowerline StyleMode = "powerline"
)

type SegmentConfig struct {
	ID      SegmentID       `toml:"id"`
	Enabled bool            `toml:"enabled"`
	Icon    IconConfig      `toml:"icon"`
	Colors  ColorConfig     `toml:"colors"`
	Styles  TextStyleConfig `toml:"styles"`
	Options map[string]any  `toml:"options"`
}

type IconConfig struct {
	Plain    string `toml:"plain"`
	NerdFont string `toml:"nerd_font"`
}

type ColorConfig struct {
	Icon       *AnsiColor `toml:"icon"`
	Text       *AnsiColor `toml:"text"`
	Background *AnsiColor `toml:"background"`
}

type TextStyleConfig struct {
	TextBold bool `toml:"text_bold"`
}

type AnsiColor struct {
	C16  *uint8 `toml:"c16,omitempty"`
	C256 *uint8 `toml:"c256,omitempty"`
	R    *uint8 `toml:"r,omitempty"`
	G    *uint8 `toml:"g,omitempty"`
	B    *uint8 `toml:"b,omitempty"`
}

type SegmentID string

const (
	SegmentModel         SegmentID = "model"
	SegmentEffort        SegmentID = "effort"
	SegmentDirectory     SegmentID = "directory"
	SegmentGit           SegmentID = "git"
	SegmentContextWindow SegmentID = "context_window"
	SegmentCost          SegmentID = "cost"
	SegmentSession       SegmentID = "session"
	SegmentEnvironment   SegmentID = "environment"
)

func Load() (Config, error) {
	path := ConfigPath()
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return DefaultConfig(), nil
		}
		return DefaultConfig(), err
	}

	return LoadFromPath(path)
}

func LoadFromPath(path string) (Config, error) {
	var cfg Config
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return DefaultConfig(), err
	}

	applyConfigDefaults(&cfg)
	return cfg, nil
}

func ConfigPath() string {
	return filepath.Join(CclineDir(), "config.toml")
}

func DefaultConfig() Config {
	return Config{
		Style: StyleConfig{
			Mode:      StyleModePlain,
			Separator: " | ",
		},
		Segments: buildDefaultSegments(),
		Theme:    "default",
	}
}

func Option[T comparable](options map[string]any, key string, fallback T) T {
	if options == nil {
		return fallback
	}

	value, ok := options[key]
	if !ok {
		return fallback
	}

	result, ok := value.(T)
	if !ok {
		return fallback
	}

	return result
}

func IntOption(options map[string]any, key string, fallback int) int {
	if options == nil {
		return fallback
	}

	value, ok := options[key]
	if !ok {
		return fallback
	}

	switch typed := value.(type) {
	case int64:
		return int(typed)
	case int32:
		return int(typed)
	case int:
		return typed
	case float64:
		return int(typed)
	default:
		return fallback
	}
}

func applyConfigDefaults(cfg *Config) {
	if cfg.Style.Mode == "" {
		cfg.Style.Mode = StyleModePlain
	}

	if cfg.Style.Separator == "" {
		cfg.Style.Separator = " | "
	}

	if len(cfg.Segments) == 0 {
		cfg.Segments = buildDefaultSegments()
	}

	if cfg.Theme == "" {
		cfg.Theme = "default"
	}

	for i := range cfg.Segments {
		if cfg.Segments[i].Options == nil {
			cfg.Segments[i].Options = map[string]any{}
		}
	}
}

func color16(value uint8) *AnsiColor {
	return &AnsiColor{C16: &value}
}

type segmentDefaults struct {
	id       SegmentID
	enabled  bool
	plain    string
	nerdFont string
	iconC16  uint8
	textC16  uint8
	options  map[string]any
}

var defaultSegmentDefs = []segmentDefaults{
	{SegmentModel, true, "🤖", "\uE26D", 14, 14, nil},
	{SegmentEffort, true, "🧠", "\uF0A39", 13, 13, nil},
	{SegmentDirectory, true, "📁", "\uF024B", 11, 10, nil},
	{SegmentGit, true, "🌿", "\uF02A2", 12, 12, map[string]any{"show_sha": false}},
	{SegmentContextWindow, true, "⚡️", "\uF49B", 13, 13, nil},
	{SegmentCost, false, "💰", "\uEEC1", 3, 3, nil},
	{SegmentSession, false, "⏱️", "\uF19BB", 2, 2, nil},
	{SegmentEnvironment, true, "🛠️", "\uF0AD", 14, 14, nil},
}

func buildDefaultSegments() []SegmentConfig {
	result := make([]SegmentConfig, len(defaultSegmentDefs))
	for i, d := range defaultSegmentDefs {
		opts := map[string]any{}
		if d.options != nil {
			opts = d.options
		}
		result[i] = SegmentConfig{
			ID:      d.id,
			Enabled: d.enabled,
			Icon:    IconConfig{Plain: d.plain, NerdFont: d.nerdFont},
			Colors:  ColorConfig{Icon: color16(d.iconC16), Text: color16(d.textC16)},
			Options: opts,
		}
	}
	return result
}
