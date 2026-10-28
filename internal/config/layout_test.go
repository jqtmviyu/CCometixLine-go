package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolvedLayoutSelectsEnabledSegments(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Segments = []SegmentConfig{
		{ID: SegmentModel, Enabled: true},
		{ID: SegmentGit, Enabled: true},
	}
	cfg.Layout.Lines = []LayoutLine{{Segments: []SegmentID{SegmentModel, SegmentModel, SegmentEnvironment}}}
	layout := cfg.ResolveLayout()
	if got := segmentIDs(layout.Segments); !reflect.DeepEqual(got, []SegmentID{SegmentModel}) {
		t.Fatalf("selected=%v", got)
	}
	if !reflect.DeepEqual(layout.Lines, cfg.Layout.Lines) {
		t.Fatalf("line order or duplicates changed: %+v", layout.Lines)
	}
}

func TestCustomSegmentsUseDefaultLayoutWhenLinesAreOmitted(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.toml")
	contents := `[[segments]]
id="environment"
enabled=true
[[segments]]
id="model"
enabled=true
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	layout := cfg.ResolveLayout()
	if !reflect.DeepEqual(layout.Lines, DefaultConfig().Layout.Lines) {
		t.Fatalf("omitted lines should use the default two-line layout: %+v", layout)
	}
	if got := segmentIDs(layout.Segments); !reflect.DeepEqual(got, []SegmentID{SegmentEnvironment, SegmentModel}) {
		t.Fatalf("unexpected selected fields: %v", got)
	}
}

func segmentIDs(fields []SegmentConfig) []SegmentID {
	ids := make([]SegmentID, 0, len(fields))
	for _, field := range fields {
		ids = append(ids, field.ID)
	}
	return ids
}
