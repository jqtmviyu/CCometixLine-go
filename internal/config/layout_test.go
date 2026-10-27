package config

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestResolvedLayoutSharesSelectionAndTaskDefaults(t *testing.T) {
	cfg := DefaultConfig()
	cfg.Segments = []SegmentConfig{
		{ID: SegmentModel, Enabled: true},
		{ID: SegmentGit, Enabled: true},
		{ID: SegmentTaskName, Enabled: false},
	}
	cfg.Layout.Lines = []LayoutLine{{Segments: []SegmentID{SegmentModel, SegmentModel, SegmentEnvironment}}}
	cfg.Layout.Task = []SegmentID{SegmentTaskName, SegmentTaskStatus, SegmentTaskElapsed}
	layout := cfg.ResolveLayout()
	if got := segmentIDs(layout.Segments); !reflect.DeepEqual(got, []SegmentID{SegmentModel}) {
		t.Fatalf("selected=%v", got)
	}
	if !reflect.DeepEqual(layout.Lines, cfg.Layout.Lines) {
		t.Fatalf("line order or duplicates changed: %+v", layout.Lines)
	}
	if got := segmentIDs(layout.Task); !reflect.DeepEqual(got, []SegmentID{SegmentTaskStatus, SegmentTaskElapsed}) {
		t.Fatalf("task defaults or explicit disable changed: %v", got)
	}
	cfg.Layout.Task = []SegmentID{}
	if got := cfg.ResolveLayout(); len(got.Task) != 0 {
		t.Fatalf("empty task layout should stay empty: %+v", got.Task)
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
[layout]
task=[]
`
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadFromPath(path)
	if err != nil {
		t.Fatal(err)
	}
	layout := cfg.ResolveLayout()
	if !reflect.DeepEqual(layout.Lines, DefaultConfig().Layout.Lines) || len(layout.Task) != 0 {
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
