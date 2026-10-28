package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestConfigPathUsesClaudeConfigDir(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)

	path := ConfigPath()
	want := filepath.Join(root, "ccline", "config.toml")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestConfigPathDefaultsToClaudeDir(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", "")

	path := ConfigPath()
	want := filepath.Join(ClaudeDir(), "ccline", "config.toml")
	if path != want {
		t.Fatalf("expected %q, got %q", want, path)
	}
}

func TestDefaultConfigEnablesEnvironmentSegmentWithToolIcon(t *testing.T) {
	cfg := DefaultConfig()

	for _, segment := range cfg.Segments {
		if segment.ID != SegmentEnvironment {
			continue
		}
		if !segment.Enabled {
			t.Fatalf("expected environment enabled by default")
		}
		if segment.Icon.Emoji != "🧰" {
			t.Fatalf("expected plain icon %q, got %q", "🧰", segment.Icon.Emoji)
		}
		return
	}

	t.Fatalf("environment segment not found")
}

func TestDefaultConfigUsesSelectedIcons(t *testing.T) {
	cfg := DefaultConfig()
	for id, want := range map[SegmentID]string{
		SegmentModel:         "🤖",
		SegmentGit:           "🐙",
		SegmentContextWindow: "💿",
	} {
		segment := findSegmentConfig(t, cfg, id)
		if segment.Icon.Emoji != want {
			t.Errorf("unexpected emoji for %q: got %q, want %q", id, segment.Icon.Emoji, want)
		}
	}

	line := cfg.Layout.Lines[0].Segments
	if len(line) < 2 || line[0] != SegmentModel || line[1] != SegmentDirectory {
		t.Fatalf("expected model first on the first line, got %#v", line)
	}
}

func findSegmentConfig(t *testing.T, cfg Config, id SegmentID) SegmentConfig {
	t.Helper()
	for _, segment := range cfg.Segments {
		if segment.ID == id {
			return segment
		}
	}
	t.Fatalf("segment %q not found", id)
	return SegmentConfig{}
}

func TestDefaultConfigUsesTwoLayoutLines(t *testing.T) {
	cfg := DefaultConfig()
	want := []LayoutLine{
		{Segments: []SegmentID{SegmentModel, SegmentDirectory, SegmentGit, SegmentContextWindow}},
		{Segments: []SegmentID{SegmentEnvironment, SegmentCost, SegmentSession}},
	}
	if !reflect.DeepEqual(cfg.Layout.Lines, want) {
		t.Fatalf("unexpected default layout: got %#v, want %#v", cfg.Layout.Lines, want)
	}
}

func TestConfigLoadsCustomEmoji(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := `
[[segments]]
id = "model"
enabled = true

[segments.icon]
emoji = "🤖"
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Segments) != 1 || cfg.Segments[0].Icon.Emoji != "🤖" {
		t.Fatalf("unexpected custom emoji config: %#v", cfg.Segments)
	}
}

func TestConfigLoadsFixedMultiLineLayout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := `
[style]
separator = " | "

[[layout.lines]]
segments = ["model", "git"]

[[layout.lines]]
segments = ["context_window", "cost"]
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Layout.Lines) != 2 || len(cfg.Layout.Lines[1].Segments) != 2 {
		t.Fatalf("unexpected layout: %#v", cfg.Layout)
	}
	if cfg.Layout.Lines[1].Segments[1] != SegmentCost {
		t.Fatalf("unexpected second line segments: %#v", cfg.Layout.Lines[1])
	}
}

func TestConfigLoadsThreeConfiguredLinesAndSubset(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := `
[[layout.lines]]
segments = ["model"]

[[layout.lines]]
segments = ["git", "context_window"]

[[layout.lines]]
segments = ["environment"]
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Layout.Lines) != 3 || len(cfg.Layout.Lines[0].Segments) != 1 || cfg.Layout.Lines[1].Segments[1] != SegmentContextWindow {
		t.Fatalf("unexpected custom layout: %#v", cfg.Layout.Lines)
	}
}

func TestExampleConfigLoadsAllMetricLines(t *testing.T) {
	cfg, err := LoadFromPath(filepath.Join("..", "..", "examples", "config.toml"))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(cfg.Layout.Lines, DefaultConfig().Layout.Lines) {
		t.Fatalf("unexpected example layout: %#v", cfg.Layout.Lines)
	}
}

func TestCacheConfigDefaultsAndOverrides(t *testing.T) {
	for _, tc := range []struct {
		name, contents string
		want           CacheConfig
	}{
		{"defaults", "theme = 'default'", CacheConfig{GitSeconds: 3, EnvironmentSeconds: 60}},
		{"partial", "[cache]\ngit_seconds = 7", CacheConfig{GitSeconds: 7, EnvironmentSeconds: 60}},
		{"disable", "[cache]\ngit_seconds = 0\nenvironment_seconds = 0", CacheConfig{}},
		{"custom", "[cache]\ngit_seconds = 5\nenvironment_seconds = 120", CacheConfig{GitSeconds: 5, EnvironmentSeconds: 120}},
	} {
		t.Run(tc.name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "config.toml")
			if err := os.WriteFile(path, []byte(tc.contents), 0o600); err != nil {
				t.Fatal(err)
			}
			cfg, err := LoadFromPath(path)
			if err != nil {
				t.Fatal(err)
			}
			if cfg.Cache != tc.want {
				t.Fatalf("cache=%+v, want %+v", cfg.Cache, tc.want)
			}
		})
	}
}
