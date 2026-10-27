package segments

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func TestContextWindowSegmentUsesReportedPercentageAndInputTokens(t *testing.T) {
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
	if data.Primary != "71% · 142.6k tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
	}
	if data.Metadata["tokens"] != "142624" {
		t.Fatalf("expected context tokens 142624, got %q", data.Metadata["tokens"])
	}
	if data.Metadata["percentage"] != "71" {
		t.Fatalf("expected reported percentage 71, got %q", data.Metadata["percentage"])
	}
}

func TestContextWindowSegmentCalculatesPercentageFromInputTokens(t *testing.T) {
	inputTokens := uint32(160)
	outputTokens := uint32(238)
	cacheRead := uint32(142464)
	contextLimit := uint32(200000)
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "gpt-5.4"},
			ContextWindow: &protocol.ContextWindow{
				ContextWindowSize: &contextLimit,
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
	if data.Primary != "71.3% · 142.6k tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
	}
	if data.Metadata["tokens"] != "142624" {
		t.Fatalf("expected context tokens 142624, got %q", data.Metadata["tokens"])
	}
	if data.Metadata["percentage"] != "71.312" {
		t.Fatalf("expected calculated percentage 71.312, got %q", data.Metadata["percentage"])
	}
}

func TestContextWindowSegmentPrefersTotalInputTokens(t *testing.T) {
	totalInputTokens := uint64(100000)
	inputTokens := uint32(12000)
	contextLimit := uint32(200000)
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "gpt-5.4"},
			ContextWindow: &protocol.ContextWindow{
				TotalInputTokens:  &totalInputTokens,
				ContextWindowSize: &contextLimit,
				CurrentUsage: &protocol.ContextWindowUsage{
					Raw: &protocol.RawUsage{InputTokens: &inputTokens},
				},
			},
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data == nil || data.Primary != "50% · 100k tokens" {
		t.Fatalf("expected total_input_tokens to take precedence, got %#v", data)
	}
}

func TestContextWindowSegmentUsesTotalInputTokensForCorrectedWindow(t *testing.T) {
	totalInputTokens := uint64(45000)
	inputTokens := uint32(12000)
	contextLimit := uint32(200000)
	usedPercentage := 55.0
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "gpt-5.4"},
			ContextWindow: &protocol.ContextWindow{
				TotalInputTokens:  &totalInputTokens,
				ContextWindowSize: &contextLimit,
				UsedPercentage:    &usedPercentage,
				CurrentUsage: &protocol.ContextWindowUsage{
					Raw: &protocol.RawUsage{InputTokens: &inputTokens},
				},
			},
		},
		ModelConfig: modelConfigWith(t, "[[models]]\npattern = \"gpt-5.4\"\ncontext_limit = 100000\n"),
	}

	data := segment.Collect()
	if data == nil || data.Primary != "45% · 45k tokens" {
		t.Fatalf("expected corrected percentage to use total_input_tokens, got %#v", data)
	}
}

func TestContextWindowSegmentUsesReportedWindowSize(t *testing.T) {
	inputTokens := uint32(142624)
	outputTokens := uint32(238)
	contextLimit := uint32(1000000)
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "claude-sonnet-4-5-20250929"},
			ContextWindow: &protocol.ContextWindow{
				ContextWindowSize: &contextLimit,
				CurrentUsage: &protocol.ContextWindowUsage{
					Raw: &protocol.RawUsage{
						InputTokens:  &inputTokens,
						OutputTokens: &outputTokens,
					},
				},
			},
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data.Primary != "14.3% · 142.6k tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
	}
	if data.Metadata["limit"] != "1000000" {
		t.Fatalf("expected reported limit 1000000, got %q", data.Metadata["limit"])
	}
}

func TestContextWindowSegmentUsesExplicitModelWindowCorrection(t *testing.T) {
	inputTokens := uint32(160)
	outputTokens := uint32(238)
	cacheRead := uint32(45464)
	contextLimit := uint32(200000)
	usedPercentage := 23.0
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
		ModelConfig: modelConfigWith(t, "[[models]]\npattern = \"gpt-5.4\"\ncontext_limit = 100000\n"),
	}

	data := segment.Collect()
	if data.Primary != "45.6% · 45.6k tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
	}
	if data.Metadata["limit"] != "100000" {
		t.Fatalf("expected configured limit 100000, got %q", data.Metadata["limit"])
	}
}

func TestContextWindowSegmentFallsBackToModelWindow(t *testing.T) {
	inputTokens := uint32(128000)
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "qwen3-coder"},
			ContextWindow: &protocol.ContextWindow{
				CurrentUsage: &protocol.ContextWindowUsage{
					Raw: &protocol.RawUsage{InputTokens: &inputTokens},
				},
			},
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data.Primary != "50% · 128k tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
	}
	if data.Metadata["limit"] != "256000" {
		t.Fatalf("expected model limit 256000, got %q", data.Metadata["limit"])
	}
}

func TestContextWindowSegmentShowsPercentageWithoutUsage(t *testing.T) {
	contextLimit := uint32(200000)
	usedPercentage := 71.0
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "gpt-5.4"},
			ContextWindow: &protocol.ContextWindow{
				ContextWindowSize: &contextLimit,
				UsedPercentage:    &usedPercentage,
			},
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data.Primary != "71% · - tokens" {
		t.Fatalf("unexpected primary: %s", data.Primary)
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

func TestContextWindowSegmentAppendsCacheHit(t *testing.T) {
	totalInputTokens := uint64(124000)
	contextLimit := uint32(200000)
	usedPercentage := 62.0
	observed := true
	requests := uint64(10)
	hitRatio := 0.97
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model: protocol.Model{ID: "gpt-5.4"},
			ContextWindow: &protocol.ContextWindow{
				TotalInputTokens:  &totalInputTokens,
				ContextWindowSize: &contextLimit,
				UsedPercentage:    &usedPercentage,
			},
			PromptCache: &protocol.PromptCache{
				CachingObserved: &observed, Requests: &requests, HitRatio: &hitRatio,
			},
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data == nil || data.Primary != "62% · 124k tokens · hit 97%" {
		t.Fatalf("unexpected context window with cache hit: %#v", data)
	}
	if data.Metadata["hit_ratio"] != "0.97" {
		t.Fatalf("expected hit ratio metadata 0.97, got %q", data.Metadata["hit_ratio"])
	}
}

func TestContextWindowSegmentHidesUnobservedCacheHit(t *testing.T) {
	usedPercentage := 62.0
	observed := false
	requests := uint64(10)
	hitRatio := 0.97
	segment := ContextWindowSegment{
		Input: protocol.InputData{
			Model:         protocol.Model{ID: "gpt-5.4"},
			ContextWindow: &protocol.ContextWindow{UsedPercentage: &usedPercentage},
			PromptCache:   &protocol.PromptCache{CachingObserved: &observed, Requests: &requests, HitRatio: &hitRatio},
		},
		ModelConfig: config.LoadModelConfig(),
	}

	data := segment.Collect()
	if data == nil || strings.Contains(data.Primary, "hit ") {
		t.Fatalf("expected unobserved cache hit to be hidden, got %#v", data)
	}
}

func modelConfigWith(t *testing.T, models string) config.ModelConfig {
	t.Helper()

	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	configDir := filepath.Join(root, "ccline")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, "models.toml"), []byte(models), 0o600); err != nil {
		t.Fatal(err)
	}

	return config.LoadModelConfig()
}

func uint32Ptr(value uint32) *uint32 {
	return &value
}
