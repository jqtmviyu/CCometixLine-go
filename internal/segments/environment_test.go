package segments

import (
	"ccometixline-go/internal/observation"
	"testing"
)

func TestEnvironmentSegmentFormatsCountsWithMiddleDots(t *testing.T) {
	data := (EnvironmentSegment{Counts: observation.EnvironmentCounts{Skills: 1}}).Collect()
	if data == nil || data.Primary != "memory 0 · skill 1 · mcp 0 · plugin 0" {
		t.Fatalf("unexpected environment display: %#v", data)
	}
}
