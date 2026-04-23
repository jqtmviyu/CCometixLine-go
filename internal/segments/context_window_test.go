package segments

import (
	"os"
	"path/filepath"
	"testing"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func TestContextWindowSegmentCollectFromPayload(t *testing.T) {
	inputTokens := uint32(160)
	outputTokens := uint32(238)
	cacheRead := uint32(142464)
	contextLimit := uint32(200000)
	usedPercentage := 71.0
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "gpt-5.4"},
			ContextWindow: &protocol.ContextWindow{
				ContextWindowSize: &contextLimit,
				UsedPercentage:    &usedPercentage,
				CurrentUsage: &protocol.ContextWindowUsage{
					Raw: &protocol.RawUsage{
						InputTokens:          &inputTokens,
						OutputTokens:         &outputTokens,
						CacheReadInputTokens: &cacheRead,
					},
				},
			},
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected context window segment data")
	}
	if data.Primary != "71% · 142.9k tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
	}
	if data.Metadata["tokens"] != "142862" {
		t.Fatalf("expected tokens metadata 142862, got %q", data.Metadata["tokens"])
	}
	if data.Metadata["percentage"] != "71" {
		t.Fatalf("expected percentage metadata 71, got %q", data.Metadata["percentage"])
	}
}

func TestContextWindowSegmentCollectFromNumericPayloadUsage(t *testing.T) {
	contextLimit := uint32(200000)
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "gpt-5.4"},
			ContextWindow: &protocol.ContextWindow{
				ContextWindowSize: &contextLimit,
				CurrentUsage: &protocol.ContextWindowUsage{
					Number: uint32Ptr(25000),
				},
			},
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected context window segment data")
	}
	if data.Primary != "12.5% · 25k tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
	}
}

func TestContextWindowSegmentFallbackToTranscript(t *testing.T) {
	// 写一个只有 payload 缺失 usage、但 transcript 有历史记录的场景
	transcriptPath := writeTranscriptFile(t, []string{
		`{"type":"assistant","message":{"role":"assistant","usage":{"input_tokens":1200,"output_tokens":300,"cache_read_input_tokens":50}}}`,
	})

	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model:          protocol.Model{ID: "claude-sonnet-4-5-20250929"},
			TranscriptPath: transcriptPath,
			// 故意不提供 context_window 字段
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected fallback to transcript, got nil")
	}
	// 1200+300+50=1550 tokens，sonnet 200000 limit → 0.775%
	if data.Metadata["tokens"] != "1550" {
		t.Fatalf("expected tokens 1550 from transcript, got %q", data.Metadata["tokens"])
	}
}

func TestContextWindowSegmentFallbackPicksLastAssistant(t *testing.T) {
	// transcript 有多行，应该取最后一个 assistant 行
	transcriptPath := writeTranscriptFile(t, []string{
		`{"type":"assistant","message":{"role":"assistant","usage":{"input_tokens":1000,"output_tokens":100}}}`,
		`{"type":"human","message":{"role":"human"}}`,
		`{"type":"assistant","message":{"role":"assistant","usage":{"input_tokens":5000,"output_tokens":500}}}`,
	})

	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model:          protocol.Model{ID: "claude-sonnet-4-5-20250929"},
			TranscriptPath: transcriptPath,
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected data, got nil")
	}
	// 应取最后一个 assistant: 5000+500=5500
	if data.Metadata["tokens"] != "5500" {
		t.Fatalf("expected 5500 from last assistant, got %q", data.Metadata["tokens"])
	}
}

func TestContextWindowSegmentNilWithNoDataNoTranscript(t *testing.T) {
	segment := ContextWindowSegment{
		Input:       protocol.InputData{Model: protocol.Model{ID: "gpt-5.4"}},
		ModelConfig: config.LoadModelConfig(),
	}
	data := segment.Collect()
	if data == nil {
		t.Fatal("expected fallback data, got nil")
	}
	if data.Primary != "- · - tokens" {
		t.Fatalf("expected '- · - tokens', got %q", data.Primary)
	}
}

func writeTranscriptFile(t *testing.T, lines []string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")
	content := ""
	for _, l := range lines {
		content += l + "\n"
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}
