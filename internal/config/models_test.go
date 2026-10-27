package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestCacheWriteCostRequiresConfiguredPriceAndTTL(t *testing.T) {
	model := ModelConfig{ModelEntries: []ModelEntry{{
		Pattern:                 "provider/model-x",
		InputPriceUSDPerMillion: 2.5,
		CacheWriteMultipliers:   map[string]float64{"5m": 1.25, "1h": 2},
	}}}

	cost, ok := model.CacheWriteCost("provider/model-x", "5m", 100_000)
	if !ok || cost != 0.3125 {
		t.Fatalf("expected 0.3125 cold cost, got %v, ok=%v", cost, ok)
	}
	if _, ok := model.CacheWriteCost("provider/model-x", "1m", 100_000); ok {
		t.Fatal("expected missing TTL multiplier to suppress cost")
	}
	if _, ok := (ModelConfig{}).CacheWriteCost("unknown", "5m", 100_000); ok {
		t.Fatal("expected unpriced model to suppress cost")
	}
}

func TestLoadModelConfigReadsCachePricing(t *testing.T) {
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	modelDir := filepath.Join(root, "ccline")
	if err := os.MkdirAll(modelDir, 0o700); err != nil {
		t.Fatal(err)
	}
	contents := `[[models]]
pattern = "provider/model-x"
input_price_usd_per_million = 2.5
cache_write_multipliers = { "5m" = 1.25, "1h" = 2.0 }
`
	if err := os.WriteFile(filepath.Join(modelDir, "models.toml"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}

	models := LoadModelConfig()
	cost, ok := models.CacheWriteCost("provider/model-x", "1h", 100_000)
	if !ok || cost != 0.5 {
		t.Fatalf("expected 0.5 cold cost, got %v, ok=%v", cost, ok)
	}
}

func TestModelEntryMatchKindsApplyToNameWindowAndPrice(t *testing.T) {
	tests := []struct {
		name    string
		match   string
		pattern string
		modelID string
	}{
		{name: "contains default", pattern: "vendor/model-x", modelID: "provider/vendor/model-x-rev"},
		{name: "prefix", match: "prefix", pattern: "vendor/model", modelID: "VENDOR/MODEL-x"},
		{name: "exact", match: "exact", pattern: "vendor/model-x", modelID: "VENDOR/MODEL-X"},
		{name: "regex", match: "regex", pattern: `^vendor/model-[a-z]+$`, modelID: "VENDOR/MODEL-x"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			models := loadModelConfigForTest(t, fmt.Sprintf(`[[models]]
pattern = %q
match = %q
display_name = "Matched Model"
context_limit = 64000
input_price_usd_per_million = 1.5
cache_write_multipliers = { "5m" = 1.0 }
`, tc.pattern, tc.match))

			if got, ok := models.GetDisplayName(tc.modelID); !ok || got != "Matched Model" {
				t.Fatalf("display name = %q, ok=%v", got, ok)
			}
			if limit, corrected := models.ResolveContextLimit(tc.modelID, 200000); limit != 64000 || !corrected {
				t.Fatalf("context limit = %d, corrected=%v", limit, corrected)
			}
			if cost, ok := models.CacheWriteCost(tc.modelID, "5m", 1_000_000); !ok || cost != 1.5 {
				t.Fatalf("cold cost = %v, ok=%v", cost, ok)
			}
			if _, ok := models.GetDisplayName("unmatched/model"); ok {
				t.Fatal("unexpected display name match")
			}
			if _, ok := models.CacheWriteCost("unmatched/model", "5m", 1_000_000); ok {
				t.Fatal("unexpected cache price match")
			}
		})
	}
}

func TestModelEntryFirstMatchWinsAcrossConsumers(t *testing.T) {
	models := loadModelConfigForTest(t, `[[models]]
pattern = "vendor/"
match = "prefix"
display_name = "First"
context_limit = 64000
input_price_usd_per_million = 1.0
cache_write_multipliers = { "5m" = 1.0 }

[[models]]
pattern = "vendor/model"
match = "contains"
display_name = "Second"
context_limit = 128000
input_price_usd_per_million = 2.0
cache_write_multipliers = { "5m" = 1.0 }
`)
	if got, _ := models.GetDisplayName("vendor/model-x"); got != "First" {
		t.Fatalf("first match name = %q", got)
	}
	if limit, _ := models.ResolveContextLimit("vendor/model-x", 200000); limit != 64000 {
		t.Fatalf("first match context limit = %d", limit)
	}
	if cost, ok := models.CacheWriteCost("vendor/model-x", "5m", 1_000_000); !ok || cost != 1.0 {
		t.Fatalf("first match price = %v, ok=%v", cost, ok)
	}
}

func TestModelEntryInvalidAndUnknownMatcherFailClosed(t *testing.T) {
	models := loadModelConfigForTest(t, `[[models]]
pattern = "(bad"
match = "regex"
display_name = "Invalid"
context_limit = 64000
input_price_usd_per_million = 1.0
cache_write_multipliers = { "5m" = 1.0 }

[[models]]
pattern = "vendor/model"
match = "glob"
display_name = "Unknown"
context_limit = 128000
input_price_usd_per_million = 2.0
cache_write_multipliers = { "5m" = 1.0 }
`)
	for _, modelID := range []string{"(bad", "vendor/model"} {
		if _, ok := models.GetDisplayName(modelID); ok {
			t.Fatalf("matcher unexpectedly accepted %q", modelID)
		}
		if _, ok := models.CacheWriteCost(modelID, "5m", 1_000_000); ok {
			t.Fatalf("invalid matcher unexpectedly supplied price for %q", modelID)
		}
	}
}

func TestModelMatcherDoesNotChangeContextModifierContainsSemantics(t *testing.T) {
	models := loadModelConfigForTest(t, `[[models]]
pattern = "vendor/model"
match = "prefix"
display_name = "Vendor Model"
context_limit = 64000

[[context_modifiers]]
pattern = "[1m]"
display_suffix = " 1M"
context_limit = 1000000
`)
	id := "vendor/model-x[1m]"
	if name, ok := models.GetDisplayName(id); !ok || name != "Vendor Model 1M" {
		t.Fatalf("unexpected modified display name %q, ok=%v", name, ok)
	}
	if limit, corrected := models.ResolveContextLimit(id, 200000); limit != 1000000 || !corrected {
		t.Fatalf("unexpected modified context limit %d, corrected=%v", limit, corrected)
	}
}

func loadModelConfigForTest(t *testing.T, contents string) ModelConfig {
	t.Helper()
	root := t.TempDir()
	t.Setenv("CLAUDE_CONFIG_DIR", root)
	modelDir := filepath.Join(root, "ccline")
	if err := os.MkdirAll(modelDir, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(modelDir, "models.toml"), []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return LoadModelConfig()
}
