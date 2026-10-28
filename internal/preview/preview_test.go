package preview

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ccometixline-go/internal/config"
)

func TestRenderUsesConfiguredSegmentsAndLayout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Layout.Lines = []config.LayoutLine{
		{Segments: []config.SegmentID{config.SegmentModel}},
		{Segments: []config.SegmentID{config.SegmentContextWindow}},
		{Segments: []config.SegmentID{config.SegmentSession}},
	}
	setEnabled(&cfg, config.SegmentCost, true)
	setEnabled(&cfg, config.SegmentGit, false)
	for i := range cfg.Segments {
		if cfg.Segments[i].ID == config.SegmentModel {
			cfg.Segments[i].Icon.Emoji = "M"
		}
	}

	first := Render(cfg)
	second := Render(cfg)
	if first != second {
		t.Fatalf("preview output changed between runs:\n%s\n---\n%s", first, second)
	}
	if !strings.Contains(first, "配置预览（示例数据，非实时会话）") {
		t.Fatalf("missing preview notice: %q", first)
	}
	if !strings.Contains(first, "Sonnet 4.5") || !strings.Contains(first, "62% · 124k tokens") || !strings.Contains(first, "hit 92%") {
		t.Fatalf("missing sample metrics: %q", first)
	}
	if strings.Contains(first, "main") {
		t.Fatalf("disabled git segment was rendered: %q", first)
	}
	if !strings.Contains(first, "\n") {
		t.Fatalf("expected preview to preserve configured rows: %q", first)
	}
}

func TestRenderShowsNoEnabledSegments(t *testing.T) {
	cfg := config.DefaultConfig()
	for i := range cfg.Segments {
		cfg.Segments[i].Enabled = false
	}
	got := Render(cfg)
	if !strings.Contains(got, "没有启用且可预览的 segment") {
		t.Fatalf("unexpected empty preview: %q", got)
	}
}

func setEnabled(cfg *config.Config, id config.SegmentID, enabled bool) {
	for i := range cfg.Segments {
		if cfg.Segments[i].ID == id {
			cfg.Segments[i].Enabled = enabled
			return
		}
	}
}

func TestPreviewUsesFixedFactsAndRealSessionFormatting(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	t.Setenv("CCLINE_CACHE_DIR", filepath.Join(root, "cache"))
	t.Setenv("CLAUDE_CODE_EFFORT_LEVEL", "max")
	t.Setenv("PATH", t.TempDir())
	cfg := config.DefaultConfig()
	setEnabled(&cfg, config.SegmentSession, true)
	output := Render(cfg)
	for _, text := range []string{"Sonnet 4.5", "high", "main", "memory 3 · skill 5 · mcp 2 · plugin 1", "hit 92%", "18m42s", "\x1b[32m+24\x1b[0m \x1b[31m-6\x1b[0m"} {
		if !strings.Contains(output, text) {
			t.Fatalf("missing %q in fixed preview: %q", text, output)
		}
	}
	if _, err := os.Stat(filepath.Join(root, "cache")); !os.IsNotExist(err) {
		t.Fatalf("preview created cache files: %v", err)
	}
}
