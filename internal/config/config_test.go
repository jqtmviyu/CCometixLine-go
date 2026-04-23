package config

import (
	"path/filepath"
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
		if segment.Icon.Plain != "🛠️" {
			t.Fatalf("expected plain icon %q, got %q", "🛠️", segment.Icon.Plain)
		}
		if segment.Icon.NerdFont != "\uF0AD" {
			t.Fatalf("expected nerd font icon %q, got %q", "\uF0AD", segment.Icon.NerdFont)
		}
		return
	}

	t.Fatalf("environment segment not found")
}
