package config

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Cache    CacheConfig     `toml:"cache"`
	Style    StyleConfig     `toml:"style"`
	Layout   LayoutConfig    `toml:"layout"`
	Segments []SegmentConfig `toml:"segments"`
	Theme    string          `toml:"theme"`
}

type CacheConfig struct {
	GitSeconds         int `toml:"git_seconds"`
	EnvironmentSeconds int `toml:"environment_seconds"`
}

func DefaultCacheConfig() CacheConfig {
	return CacheConfig{GitSeconds: 3, EnvironmentSeconds: 60}
}

func (c CacheConfig) GitTTL() time.Duration { return time.Duration(c.GitSeconds) * time.Second }
func (c CacheConfig) EnvironmentTTL() time.Duration {
	return time.Duration(c.EnvironmentSeconds) * time.Second
}

type LayoutConfig struct {
	Lines []LayoutLine `toml:"lines"`
	Task  []SegmentID  `toml:"task"`
}

type LayoutLine struct {
	Segments []SegmentID `toml:"segments"`
}

type StyleConfig struct {
	Separator string `toml:"separator"`
}

type SegmentConfig struct {
	ID      SegmentID       `toml:"id"`
	Enabled bool            `toml:"enabled"`
	Icon    IconConfig      `toml:"icon"`
	Colors  ColorConfig     `toml:"colors"`
	Styles  TextStyleConfig `toml:"styles"`
	Options map[string]any  `toml:"options"`
}

type IconConfig struct {
	Emoji string `toml:"emoji"`
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
	SegmentCache         SegmentID = "cache"
	SegmentCacheColdCost SegmentID = "cache.cold_cost"
	SegmentTaskLabel     SegmentID = "task.label"
	SegmentTaskName      SegmentID = "task.name"
	SegmentTaskStatus    SegmentID = "task.status"
	SegmentTaskElapsed   SegmentID = "task.elapsed"
	SegmentTaskTrend     SegmentID = "task.trend"
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
	cfg := Config{Cache: DefaultCacheConfig()}
	if _, err := toml.DecodeFile(path, &cfg); err != nil {
		return DefaultConfig(), err
	}

	maxSeconds := int64((1<<63 - 1) / time.Second)
	for _, seconds := range []int{cfg.Cache.GitSeconds, cfg.Cache.EnvironmentSeconds} {
		if seconds < 0 || int64(seconds) > maxSeconds {
			return DefaultConfig(), fmt.Errorf("cache durations must be between 0 and %d seconds", maxSeconds)
		}
	}
	applyConfigDefaults(&cfg)
	return cfg, nil
}

func ConfigPath() string {
	return filepath.Join(CclineDir(), "config.toml")
}

func DefaultConfig() Config {
	return Config{
		Cache:    DefaultCacheConfig(),
		Style:    StyleConfig{Separator: " | "},
		Layout:   defaultLayout(),
		Segments: buildDefaultSegments(),
		Theme:    "default",
	}
}

func defaultLayout() LayoutConfig {
	return LayoutConfig{
		Lines: []LayoutLine{
			{Segments: []SegmentID{SegmentModel, SegmentEffort, SegmentDirectory, SegmentGit, SegmentContextWindow}},
			{Segments: []SegmentID{SegmentEnvironment, SegmentCache, SegmentCacheColdCost, SegmentCost, SegmentSession}},
		},
		Task: []SegmentID{SegmentTaskStatus, SegmentTaskName, SegmentTaskLabel, SegmentTaskElapsed, SegmentTaskTrend},
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
	if cfg.Style.Separator == "" {
		cfg.Style.Separator = " | "
	}

	if len(cfg.Segments) == 0 {
		cfg.Segments = buildDefaultSegments()
	}
	if len(cfg.Layout.Lines) == 0 {
		cfg.Layout.Lines = defaultLayout().Lines
	}
	if cfg.Layout.Task == nil {
		cfg.Layout.Task = defaultLayout().Task
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

func paletteColor(value uint8) *AnsiColor {
	if value < 16 {
		return &AnsiColor{C16: &value}
	}
	return &AnsiColor{C256: &value}
}

type segmentDefaults struct {
	id        SegmentID
	enabled   bool
	emoji     string
	iconColor uint8
	textColor uint8
	options   map[string]any
}

var defaultSegmentDefs = []segmentDefaults{
	{SegmentModel, true, "🤖", 14, 14, nil},
	{SegmentEffort, true, "", 14, 14, nil},
	{SegmentDirectory, true, "📁", 11, 10, nil},
	{SegmentGit, true, "🐙", 12, 12, nil},
	{SegmentContextWindow, true, "💿", 13, 13, nil},
	{SegmentCost, false, "💰", 3, 3, nil},
	{SegmentSession, false, "⏱️", 2, 2, nil},
	{SegmentEnvironment, true, "🧰", 250, 250, nil},
	{SegmentCache, true, "🔥", 208, 208, nil},
	{SegmentCacheColdCost, true, "💲", 3, 3, nil},
	{SegmentTaskLabel, true, "↳", 11, 11, nil},
	{SegmentTaskName, true, "👤", 14, 14, nil},
	{SegmentTaskStatus, true, "", 10, 10, nil},
	{SegmentTaskElapsed, true, "⏱️", 11, 11, nil},
	{SegmentTaskTrend, true, "💿", 13, 13, nil},
}

func buildDefaultSegments() []SegmentConfig {
	result := make([]SegmentConfig, len(defaultSegmentDefs))
	for i, d := range defaultSegmentDefs {
		result[i] = defaultSegment(d)
	}
	return result
}

func defaultSegment(d segmentDefaults) SegmentConfig {
	options := d.options
	if options == nil {
		options = map[string]any{}
	}
	return SegmentConfig{
		ID: d.id, Enabled: d.enabled, Icon: IconConfig{Emoji: d.emoji},
		Colors:  ColorConfig{Icon: paletteColor(d.iconColor), Text: paletteColor(d.textColor)},
		Options: options,
	}
}
