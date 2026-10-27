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

func TestDefaultConfigUsesSelectedIconsAndKeepsModelEffortTogether(t *testing.T) {
	cfg := DefaultConfig()
	for id, want := range map[SegmentID]string{
		SegmentModel:            "🤖",
		SegmentEffort:           "",
		SegmentGit:              "🐙",
		SegmentContextWindow:    "💿",
		SegmentCache:            "🔥",
		SegmentCacheColdCost:    "💲",
		SegmentContextBreakdown: "🔈",
		SegmentTimingTPS:        "⚡",
		SegmentActivityTools:    "✏️",
		SegmentActivityAgents:   "👥",
	} {
		segment := findSegmentConfig(t, cfg, id)
		if segment.Icon.Emoji != want {
			t.Errorf("unexpected emoji for %q: got %q, want %q", id, segment.Icon.Emoji, want)
		}
	}

	line := cfg.Layout.Lines[0].Segments
	if len(line) < 2 || line[0] != SegmentModel || line[1] != SegmentEffort {
		t.Fatalf("expected model and effort to be adjacent on the first line, got %#v", line)
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

func TestDefaultConfigUsesThreeLayoutLines(t *testing.T) {
	cfg := DefaultConfig()
	want := []LayoutLine{
		{Segments: []SegmentID{SegmentModel, SegmentEffort, SegmentDirectory, SegmentGit, SegmentEnvironment}},
		{Segments: []SegmentID{SegmentContextWindow, SegmentContextBreakdown, SegmentCache, SegmentCacheColdCost, SegmentCost, SegmentSession, SegmentTimingTTFT, SegmentTimingTPS}},
		{Segments: []SegmentID{SegmentActivityTools, SegmentActivityAgents}},
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
wrap_width = 88

[[layout.lines]]
segments = ["model", "git"]

[[layout.lines]]
segments = ["context_window", "cache"]

[layout]
task = ["task.status", "task.label"]
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
	if cfg.Layout.Lines[1].Segments[1] != SegmentCache {
		t.Fatalf("unexpected second line segments: %#v", cfg.Layout.Lines[1])
	}
	if len(cfg.Layout.Task) != 2 || cfg.Layout.Task[0] != SegmentTaskStatus {
		t.Fatalf("unexpected task layout: %#v", cfg.Layout.Task)
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

func TestConfigLegacyGoSegmentOrderBecomesOneLine(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := `[[segments]]
id = "model"
enabled = true
[[segments]]
id = "git"
enabled = true
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Layout.Lines) != 1 || len(cfg.Layout.Lines[0].Segments) != 2 {
		t.Fatalf("expected legacy segment list preserved as one line, got %#v", cfg.Layout)
	}
}
