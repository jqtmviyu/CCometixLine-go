package segments

import (
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
	if data.Primary != "71.4% · 142.9k tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
	}
	if data.Metadata["tokens"] != "142862" {
		t.Fatalf("expected tokens metadata 142862, got %q", data.Metadata["tokens"])
	}
	if data.Metadata["percentage"] != "71.431" {
		t.Fatalf("expected percentage metadata 71.431, got %q", data.Metadata["percentage"])
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

func TestContextWindowSegmentCalculatesPercentageWithoutUsedPercentage(t *testing.T) {
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

func uint32Ptr(value uint32) *uint32 {
	return &value
}
