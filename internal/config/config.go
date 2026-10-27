package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Style    StyleConfig     `toml:"style"`
	Layout   LayoutConfig    `toml:"layout"`
	Segments []SegmentConfig `toml:"segments"`
	Theme    string          `toml:"theme"`
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
	SegmentModel            SegmentID = "model"
	SegmentEffort           SegmentID = "effort"
	SegmentDirectory        SegmentID = "directory"
	SegmentGit              SegmentID = "git"
	SegmentContextWindow    SegmentID = "context_window"
	SegmentCost             SegmentID = "cost"
	SegmentSession          SegmentID = "session"
	SegmentEnvironment      SegmentID = "environment"
	SegmentContextBreakdown SegmentID = "context.breakdown"
	SegmentCache            SegmentID = "cache"
	SegmentCacheColdCost    SegmentID = "cache.cold_cost"
	SegmentTimingTTFT       SegmentID = "timing.ttft"
	SegmentTimingTPS        SegmentID = "timing.tps"
	SegmentActivityTools    SegmentID = "activity.tools"
	SegmentActivityAgents   SegmentID = "activity.agents"
	SegmentTaskLabel        SegmentID = "task.label"
	SegmentTaskName         SegmentID = "task.name"
	SegmentTaskStatus       SegmentID = "task.status"
	SegmentTaskElapsed      SegmentID = "task.elapsed"
	SegmentTaskTrend        SegmentID = "task.trend"
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
		Style:    StyleConfig{Separator: " | "},
		Layout:   defaultLayout(),
		Segments: buildDefaultSegments(),
		Theme:    "default",
	}
}

func defaultLayout() LayoutConfig {
	return LayoutConfig{
		Lines: []LayoutLine{
			{Segments: []SegmentID{SegmentModel, SegmentEffort, SegmentDirectory, SegmentGit, SegmentEnvironment}},
			{Segments: []SegmentID{SegmentContextWindow, SegmentContextBreakdown, SegmentCache, SegmentCacheColdCost, SegmentCost, SegmentSession, SegmentTimingTTFT, SegmentTimingTPS}},
			{Segments: []SegmentID{SegmentActivityTools, SegmentActivityAgents}},
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

func enabledSegmentIDs(segments []SegmentConfig) []SegmentID {
	ids := make([]SegmentID, 0, len(segments))
	for _, segment := range segments {
		if segment.Enabled {
			ids = append(ids, segment.ID)
		}
	}
	return ids
}

func applyConfigDefaults(cfg *Config) {
	if cfg.Style.Separator == "" {
		cfg.Style.Separator = " | "
	}

	segmentsWereMissing := len(cfg.Segments) == 0
	if segmentsWereMissing {
		cfg.Segments = buildDefaultSegments()
	}
	if len(cfg.Layout.Lines) == 0 {
		if segmentsWereMissing {
			cfg.Layout.Lines = defaultLayout().Lines
		} else {
			cfg.Layout.Lines = []LayoutLine{{Segments: enabledSegmentIDs(cfg.Segments)}}
		}
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

func color16(value uint8) *AnsiColor {
	return &AnsiColor{C16: &value}
}

type segmentDefaults struct {
	id      SegmentID
	enabled bool
	emoji   string
	iconC16 uint8
	textC16 uint8
	options map[string]any
}

var defaultSegmentDefs = []segmentDefaults{
	{SegmentModel, true, "🤖", 14, 14, nil},
	{SegmentEffort, true, "", 13, 13, nil},
	{SegmentDirectory, true, "📁", 11, 10, nil},
	{SegmentGit, true, "🐙", 12, 12, nil},
	{SegmentContextWindow, true, "💿", 13, 13, nil},
	{SegmentCost, false, "💰", 3, 3, nil},
	{SegmentSession, false, "⏱️", 2, 2, nil},
	{SegmentEnvironment, true, "🧰", 14, 14, nil},
	{SegmentContextBreakdown, true, "🔈", 13, 13, nil},
	{SegmentCache, true, "🔥", 6, 6, nil},
	{SegmentCacheColdCost, true, "💲", 3, 3, nil},
	{SegmentTimingTTFT, true, "⌛", 11, 11, nil},
	{SegmentTimingTPS, true, "⚡", 14, 14, nil},
	{SegmentActivityTools, true, "✏️", 12, 12, nil},
	{SegmentActivityAgents, true, "👥", 14, 14, nil},
	{SegmentTaskLabel, true, "↳", 11, 11, nil},
	{SegmentTaskName, true, "👤", 14, 14, nil},
	{SegmentTaskStatus, true, "", 10, 10, nil},
	{SegmentTaskElapsed, true, "⏱️", 11, 11, nil},
	{SegmentTaskTrend, true, "📈", 13, 13, nil},
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
			Icon:    IconConfig{Emoji: d.emoji},
			Colors:  ColorConfig{Icon: color16(d.iconC16), Text: color16(d.textC16)},
			Options: opts,
		}
	}
	return result
}
