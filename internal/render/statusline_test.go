package render

import (
	"strings"
	"testing"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
	"ccometixline-go/internal/segments"
)

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

	items := segments.CollectAll(cfg, config.LoadModelConfig(), input)
	result := StatusLineGenerator{Config: cfg}.Generate(items)

	checks := []string{"Sonnet 4.5", "high", "CCometixLine-go", "0.8% · 1.6k tokens", "$0.02", "1m31s", "mem", "skills", "mcp", "plugins"}
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

func TestGeneratePowerlineStatusLine(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Style.Mode = config.StyleModePowerline
	cfg.Style.Separator = ""
	cfg.Segments = cfg.Segments[:2]

	input := protocol.InputData{
		Model:     protocol.Model{ID: "claude-sonnet-4-5-20250929", DisplayName: "Claude Sonnet 4.5"},
		Workspace: protocol.Workspace{CurrentDir: "D:/Work-Hxx/Demo/statusLine/CCometixLine-go"},
	}

	items := segments.CollectAll(cfg, config.LoadModelConfig(), input)
	result := StatusLineGenerator{Config: cfg}.Generate(items)

	if !strings.Contains(result, "") {
		t.Fatalf("expected powerline arrow in result, got %q", result)
	}
}
