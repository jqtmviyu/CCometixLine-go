package preview

import (
	"strings"
	"testing"

	"ccometixline-go/internal/config"
)

func TestRenderUsesConfiguredSegmentsAndLayout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Layout.Lines = []config.LayoutLine{
		{Segments: []config.SegmentID{config.SegmentModel, config.SegmentEffort}},
		{Segments: []config.SegmentID{config.SegmentContextWindow, config.SegmentCache}},
		{Segments: []config.SegmentID{config.SegmentActivityTools}},
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
	if !strings.Contains(first, "Sonnet 4.5") || !strings.Contains(first, "62% · 124k tokens") || !strings.Contains(first, "warm 4m") {
		t.Fatalf("missing sample metrics: %q", first)
	}
	if strings.Contains(first, "main") {
		t.Fatalf("disabled git segment was rendered: %q", first)
	}
	if !strings.Contains(first, "subagentStatusLine") || !strings.Contains(first, "Reviewing model config") {
		t.Fatalf("missing task layout preview: %q", first)
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
	cfg.Layout.Task = nil
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
