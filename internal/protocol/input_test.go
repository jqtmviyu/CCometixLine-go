package protocol

import (
	"encoding/json"
	"testing"
)

func TestInputDataParsesEffort(t *testing.T) {
	payload := []byte(`{"model":{"id":"claude-sonnet-4-5-20250929","display_name":"Claude Sonnet 4.5"},"workspace":{"current_dir":"D:/Work-Hxx/Demo/statusLine/CCometixLine-go"},"effort":"high"}`)

	var input InputData
	if err := json.Unmarshal(payload, &input); err != nil {
		t.Fatal(err)
	}
	if input.Effort != "high" {
		t.Fatalf("expected effort high, got %q", input.Effort)
	}
}

func TestInputDataParsesContextWindowObjectUsage(t *testing.T) {
	payload := []byte(`{"model":{"id":"gpt-5.4","display_name":"gpt-5.4"},"workspace":{"current_dir":"D:/Work-Hxx/Demo/statusLine"},"context_window":{"context_window_size":200000,"used_percentage":71,"current_usage":{"input_tokens":160,"output_tokens":238,"cache_read_input_tokens":142464}}}`)

	var input InputData
	if err := json.Unmarshal(payload, &input); err != nil {
		t.Fatal(err)
	}
	if input.ContextWindow == nil {
		t.Fatal("expected context_window parsed")
	}
	if input.ContextWindow.ContextWindowSize == nil || *input.ContextWindow.ContextWindowSize != 200000 {
		t.Fatalf("unexpected context window size: %#v", input.ContextWindow.ContextWindowSize)
	}
	if input.ContextWindow.UsedPercentage == nil || *input.ContextWindow.UsedPercentage != 71 {
		t.Fatalf("unexpected used percentage: %#v", input.ContextWindow.UsedPercentage)
	}
	if input.ContextWindow.CurrentUsage == nil || input.ContextWindow.CurrentUsage.Raw == nil || input.ContextWindow.CurrentUsage.Raw.CacheReadInputTokens == nil || *input.ContextWindow.CurrentUsage.Raw.CacheReadInputTokens != 142464 {
		t.Fatalf("unexpected current usage: %#v", input.ContextWindow.CurrentUsage)
	}
}

func TestInputDataParsesContextWindowNumericUsage(t *testing.T) {
	payload := []byte(`{"model":{"id":"gpt-5.4","display_name":"gpt-5.4"},"workspace":{"current_dir":"D:/Work-Hxx/Demo/statusLine"},"context_window":{"context_window_size":200000,"current_usage":12345}}`)

	var input InputData
	if err := json.Unmarshal(payload, &input); err != nil {
		t.Fatal(err)
	}
	if input.ContextWindow == nil || input.ContextWindow.CurrentUsage == nil || input.ContextWindow.CurrentUsage.Number == nil {
		t.Fatal("expected numeric current_usage parsed")
	}
	if *input.ContextWindow.CurrentUsage.Number != 12345 {
		t.Fatalf("unexpected numeric current usage: %#v", input.ContextWindow.CurrentUsage.Number)
	}
}

func TestNormalizePrefersAnthropicFields(t *testing.T) {
	inputTokens := uint32(1200)
	promptTokens := uint32(999)
	outputTokens := uint32(300)
	cacheCreation := uint32(100)
	cacheRead := uint32(50)

	raw := RawUsage{
		InputTokens:              &inputTokens,
		PromptTokens:             &promptTokens,
		OutputTokens:             &outputTokens,
		CacheCreationInputTokens: &cacheCreation,
		CacheReadInputTokens:     &cacheRead,
	}

	normalized := raw.Normalize()
	if normalized.InputTokens != 1200 {
		t.Fatalf("expected input tokens 1200, got %d", normalized.InputTokens)
	}
	if normalized.OutputTokens != 300 {
		t.Fatalf("expected output tokens 300, got %d", normalized.OutputTokens)
	}
	if normalized.UsedTokens() != 1650 {
		t.Fatalf("expected used tokens 1650, got %d", normalized.UsedTokens())
	}
	if normalized.CachedTokens() != 150 {
		t.Fatalf("expected cached tokens 150, got %d", normalized.CachedTokens())
	}
}
