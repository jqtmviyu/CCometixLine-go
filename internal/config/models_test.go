package config

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestModelEntryMatchKindsApplyToNameAndWindow(t *testing.T) {
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
`, tc.pattern, tc.match))

			if got, ok := models.GetDisplayName(tc.modelID); !ok || got != "Matched Model" {
				t.Fatalf("display name = %q, ok=%v", got, ok)
			}
			if limit, corrected := models.ResolveContextLimit(tc.modelID, 200000); limit != 64000 || !corrected {
				t.Fatalf("context limit = %d, corrected=%v", limit, corrected)
			}
			if _, ok := models.GetDisplayName("unmatched/model"); ok {
				t.Fatal("unexpected display name match")
			}
		})
	}
}

func TestModelEntryFirstMatchWins(t *testing.T) {
	models := loadModelConfigForTest(t, `[[models]]
pattern = "vendor/"
match = "prefix"
display_name = "First"
context_limit = 64000

[[models]]
pattern = "vendor/model"
match = "contains"
display_name = "Second"
context_limit = 128000
`)
	if got, _ := models.GetDisplayName("vendor/model-x"); got != "First" {
		t.Fatalf("first match name = %q", got)
	}
	if limit, _ := models.ResolveContextLimit("vendor/model-x", 200000); limit != 64000 {
		t.Fatalf("first match context limit = %d", limit)
	}
}

func TestModelEntryInvalidAndUnknownMatcherFailClosed(t *testing.T) {
	models := loadModelConfigForTest(t, `[[models]]
pattern = "(bad"
match = "regex"
display_name = "Invalid"
context_limit = 64000

[[models]]
pattern = "vendor/model"
match = "glob"
display_name = "Unknown"
context_limit = 128000
`)
	for _, modelID := range []string{"(bad", "vendor/model"} {
		if _, ok := models.GetDisplayName(modelID); ok {
			t.Fatalf("matcher unexpectedly accepted %q", modelID)
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
