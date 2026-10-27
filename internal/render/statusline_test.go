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
			result := ansi.Strip((StatusLineGenerator{Config: cfg}).Generate(items))
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

	result := ansi.Strip((StatusLineGenerator{Config: cfg}).Generate(items))
	lines := strings.Split(result, "\n")
	if len(lines) != 3 || lines[0] != "model" || lines[1] != "directory" || lines[2] != "environment" {
		t.Fatalf("unexpected configured lines: %#v", lines)
	}
}

func TestGenerateTaskLineStillTruncatesToColumns(t *testing.T) {
	cfg := config.DefaultConfig()
	items := []segments.CollectedSegment{{
		Config: config.SegmentConfig{ID: config.SegmentTaskName, Icon: config.IconConfig{Emoji: "↳"}},
		Data:   segments.SegmentData{Primary: "a-very-long-task-name"},
	}}
	result := (StatusLineGenerator{Config: cfg}).GenerateTaskLine(items, 8)
	if ansi.StringWidth(result) > 8 || !strings.HasSuffix(ansi.Strip(result), "…") {
		t.Fatalf("task line was not truncated to columns: %q", result)
	}
}

func TestGenerateUsesMiddleDotForAdjacentModelAndEffort(t *testing.T) {
	tests := []struct {
		name      string
		separator string
		lines     [][]config.SegmentID
		want      string
	}{
		{
			name:      "default separator",
			separator: " | ",
			lines:     [][]config.SegmentID{{config.SegmentModel, config.SegmentEffort, config.SegmentDirectory}},
			want:      "sonnet · high | project",
		},
		{
			name:      "custom separator and reversed pair",
			separator: " :: ",
			lines:     [][]config.SegmentID{{config.SegmentEffort, config.SegmentModel, config.SegmentDirectory}},
			want:      "high · sonnet :: project",
		},
		{
			name:      "nonadjacent pair",
			separator: " | ",
			lines:     [][]config.SegmentID{{config.SegmentModel, config.SegmentDirectory, config.SegmentEffort}},
			want:      "sonnet | project | high",
		},
		{
			name:      "separate lines",
			separator: " | ",
			lines:     [][]config.SegmentID{{config.SegmentModel}, {config.SegmentEffort}},
			want:      "sonnet\nhigh",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			cfg.Style.Separator = test.separator
			cfg.Layout.Lines = make([]config.LayoutLine, len(test.lines))
			items := make([]segments.CollectedSegment, 0, 3)
			for i, line := range test.lines {
				cfg.Layout.Lines[i].Segments = line
				for _, id := range line {
					value := map[config.SegmentID]string{
						config.SegmentModel:     "sonnet",
						config.SegmentEffort:    "high",
						config.SegmentDirectory: "project",
					}[id]
					items = append(items, segments.CollectedSegment{
						Config: config.SegmentConfig{ID: id},
						Data:   segments.SegmentData{Primary: value},
					})
				}
			}

			got := ansi.Strip((StatusLineGenerator{Config: cfg}).Generate(items))
			if got != test.want {
				t.Fatalf("unexpected separators: got %q, want %q", got, test.want)
			}
		})
	}
}

func TestGenerateUsesDefaultEmojiIcons(t *testing.T) {
	cfg := config.DefaultConfig()
	ids := []config.SegmentID{
		config.SegmentModel,
		config.SegmentEffort,
		config.SegmentGit,
		config.SegmentContextWindow,
		config.SegmentCache,
	}
	cfg.Layout.Lines = []config.LayoutLine{{Segments: ids}}

	configs := make(map[config.SegmentID]config.SegmentConfig, len(cfg.Segments))
	for _, segment := range cfg.Segments {
		configs[segment.ID] = segment
	}
	values := map[config.SegmentID]string{
		config.SegmentModel:         "Sonnet",
		config.SegmentEffort:        "high",
		config.SegmentGit:           "main",
		config.SegmentContextWindow: "62% · 124k tokens",
		config.SegmentCache:         "warm 4m",
	}
	items := make([]segments.CollectedSegment, 0, len(ids))
	for _, id := range ids {
		items = append(items, segments.CollectedSegment{
			Config: configs[id],
			Data:   segments.SegmentData{Primary: values[id]},
		})
	}

	got := ansi.Strip((StatusLineGenerator{Config: cfg}).Generate(items))
	want := "🤖 Sonnet · high | 🐙 main | 💿 62% · 124k tokens | 🔥 warm 4m"
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

	items := segments.CollectAll(cfg, config.LoadModelConfig(), input)
	result := StatusLineGenerator{Config: cfg}.Generate(items)

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

func TestModelEffortPairSharesModelStyle(t *testing.T) {
	modelColor, effortColor := uint8(12), uint8(13)
	model := config.SegmentConfig{
		ID:     config.SegmentModel,
		Colors: config.ColorConfig{Text: &config.AnsiColor{C16: &modelColor}},
		Styles: config.TextStyleConfig{TextBold: true},
	}
	effort := config.SegmentConfig{
		ID:     config.SegmentEffort,
		Colors: config.ColorConfig{Text: &config.AnsiColor{C16: &effortColor}},
	}
	for _, ids := range [][]config.SegmentID{
		{config.SegmentModel, config.SegmentEffort},
		{config.SegmentEffort, config.SegmentModel},
	} {
		cfg := config.DefaultConfig()
		cfg.Layout.Lines = []config.LayoutLine{{Segments: ids}}
		items := []segments.CollectedSegment{
			{Config: model, Data: segments.SegmentData{Primary: "Sonnet"}},
			{Config: effort, Data: segments.SegmentData{Primary: "high"}},
		}
		texts := map[config.SegmentID]string{config.SegmentModel: "Sonnet", config.SegmentEffort: "high"}
		want := ApplyStyle(texts[ids[0]], model.Colors.Text, true) + ApplyStyle(" · ", model.Colors.Text, true) + ApplyStyle(texts[ids[1]], model.Colors.Text, true)
		if got := (StatusLineGenerator{Config: cfg}).Generate(items); got != want {
			t.Fatalf("model, separator and effort should share style: got %q, want %q", got, want)
		}
	}
}
