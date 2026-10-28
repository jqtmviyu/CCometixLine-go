package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
	"ccometixline-go/internal/segments"
)

func TestGenerateDoesNotWrapConfiguredLine(t *testing.T) {
	for _, separator := range []string{" | ", " / "} {
		t.Run(separator, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Style.Separator = separator
			cfg.Layout.Lines = []config.LayoutLine{{Segments: []config.SegmentID{
				config.SegmentDirectory,
				config.SegmentContextWindow,
				config.SegmentEnvironment,
			}}}
			color := uint8(2)
			items := []segments.CollectedSegment{
				{Config: config.SegmentConfig{ID: config.SegmentDirectory, Colors: config.ColorConfig{Text: &config.AnsiColor{C16: &color}}}, Data: segments.SegmentData{Primary: "中文"}},
				{Config: config.SegmentConfig{ID: config.SegmentContextWindow, Icon: config.IconConfig{Emoji: "⚡️"}}, Data: segments.SegmentData{Primary: "very-long-context-value"}},
				{Config: config.SegmentConfig{ID: config.SegmentEnvironment, Icon: config.IconConfig{Emoji: "🧰"}}, Data: segments.SegmentData{Primary: "memory 1 · skill 0 · mcp 0 · plugin 0"}},
			}
			result := ansi.Strip((StatusLineGenerator{Style: cfg.Style, Layout: cfg.ResolveLayout()}).Generate(items))
			if strings.Contains(result, "\n") {
				t.Fatalf("configured single line was wrapped: %q", result)
			}
			if !strings.HasPrefix(result, "中文") || !strings.Contains(result, "very-long-context-value") || !strings.Contains(result, "🧰 memory 1") {
				t.Fatalf("segments moved out of configured order: %q", result)
			}
		})
	}
}

func TestGenerateRespectsExplicitThreeLineLayout(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Layout.Lines = []config.LayoutLine{
		{Segments: []config.SegmentID{config.SegmentModel}},
		{Segments: []config.SegmentID{config.SegmentDirectory}},
		{Segments: []config.SegmentID{config.SegmentEnvironment}},
	}
	items := []segments.CollectedSegment{
		{Config: config.SegmentConfig{ID: config.SegmentModel}, Data: segments.SegmentData{Primary: "model"}},
		{Config: config.SegmentConfig{ID: config.SegmentDirectory}, Data: segments.SegmentData{Primary: "directory"}},
		{Config: config.SegmentConfig{ID: config.SegmentEnvironment}, Data: segments.SegmentData{Primary: "environment"}},
		{Config: config.SegmentConfig{ID: config.SegmentCost}, Data: segments.SegmentData{Primary: "hidden"}},
	}

	result := ansi.Strip((StatusLineGenerator{Style: cfg.Style, Layout: cfg.ResolveLayout()}).Generate(items))
	lines := strings.Split(result, "\n")
	if len(lines) != 3 || lines[0] != "model" || lines[1] != "directory" || lines[2] != "environment" {
		t.Fatalf("unexpected configured lines: %#v", lines)
	}
}

func TestGenerateUsesDefaultEmojiIcons(t *testing.T) {
	cfg := config.DefaultConfig()
	ids := []config.SegmentID{
		config.SegmentModel,
		config.SegmentGit,
		config.SegmentContextWindow,
	}
	cfg.Layout.Lines = []config.LayoutLine{{Segments: ids}}

	configs := make(map[config.SegmentID]config.SegmentConfig, len(cfg.Segments))
	for _, segment := range cfg.Segments {
		configs[segment.ID] = segment
	}
	values := map[config.SegmentID]string{
		config.SegmentModel:         "Sonnet · high",
		config.SegmentGit:           "main",
		config.SegmentContextWindow: "62% · 124k tokens · hit 97%",
	}
	items := make([]segments.CollectedSegment, 0, len(ids))
	for _, id := range ids {
		items = append(items, segments.CollectedSegment{
			Config: configs[id],
			Data:   segments.SegmentData{Primary: values[id]},
		})
	}

	got := ansi.Strip((StatusLineGenerator{Style: cfg.Style, Layout: cfg.ResolveLayout()}).Generate(items))
	want := "🤖 Sonnet · high | 🐙 main | 💿 62% · 124k tokens · hit 97%"
	if got != want {
		t.Fatalf("unexpected emoji status line: got %q, want %q", got, want)
	}
}

func TestGenerateStatusLine(t *testing.T) {
	cfg := config.DefaultConfig()
	setSegmentEnabled(cfg.Segments, config.SegmentCost, true)
	setSegmentEnabled(cfg.Segments, config.SegmentSession, true)
	setSegmentEnabled(cfg.Segments, config.SegmentEnvironment, true)

	cost := 0.023
	duration := uint64(91234)
	apiDuration := uint64(45000)
	added := uint32(12)
	removed := uint32(3)

	usedPercentage := 0.8
	inputTokens := uint32(1600)

	input := protocol.InputData{
		Model:     protocol.Model{ID: "claude-sonnet-4-5-20250929", DisplayName: "Claude Sonnet 4.5"},
		Workspace: protocol.Workspace{CurrentDir: "D:/Work-Hxx/Demo/statusLine/CCometixLine-go"},
		Effort:    &protocol.Effort{Level: "high"},
		Cost: &protocol.Cost{
			TotalCostUSD:       &cost,
			TotalDurationMS:    &duration,
			TotalAPIDurationMS: &apiDuration,
			TotalLinesAdded:    &added,
			TotalLinesRemoved:  &removed,
		},
		ContextWindow: &protocol.ContextWindow{
			UsedPercentage: &usedPercentage,
			CurrentUsage: &protocol.ContextWindowUsage{
				Raw: &protocol.RawUsage{
					InputTokens: &inputTokens,
				},
			},
		},
	}

	items := (segments.Collector{Layout: cfg.ResolveLayout(), Cache: cfg.Cache, Models: config.LoadModelConfig()}).Collect(input)
	result := StatusLineGenerator{Style: cfg.Style, Layout: cfg.ResolveLayout()}.Generate(items)

	checks := []string{"Sonnet 4.5", "high", "CCometixLine-go", "0.8% · 1.6k tokens", "$0.02", "1m31s", "memory ", "skill ", "mcp ", "plugin "}
	for _, check := range checks {
		if !strings.Contains(result, check) {
			t.Fatalf("expected result contain %q, got %q", check, result)
		}
	}
	if strings.Index(result, "Sonnet 4.5") > strings.Index(result, "high") {
		t.Fatalf("expected effort after model, got %q", result)
	}
}

func setSegmentEnabled(segments []config.SegmentConfig, id config.SegmentID, enabled bool) {
	for i := range segments {
		if segments[i].ID == id {
			segments[i].Enabled = enabled
			return
		}
	}
}
