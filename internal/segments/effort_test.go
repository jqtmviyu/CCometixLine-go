package segments

import (
	"testing"

	"ccometixline-go/internal/protocol"
)

func TestEffortSegmentCollectFromPayload(t *testing.T) {
	segment := EffortSegment{Input: protocol.InputData{Effort: &protocol.Effort{Level: "HIGH"}}}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "high" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestEffortSegmentCollectUnknownPayload(t *testing.T) {
	segment := EffortSegment{Input: protocol.InputData{Effort: &protocol.Effort{Level: "super-max"}}}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "super-max?" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestEffortSegmentCollectDefault(t *testing.T) {
	segment := EffortSegment{Input: protocol.InputData{Effort: &protocol.Effort{Level: "@@"}}}

	data := segment.Collect()
	if data == nil {
		t.Fatal("expected effort segment data")
	}
	if data.Primary != "auto" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}

func TestResolveEffortUsesPayloadOnly(t *testing.T) {
	effort := resolveEffort(protocol.InputData{Effort: &protocol.Effort{Level: "high"}})
	if effort != "high" {
		t.Fatalf("expected payload effort high, got %q", effort)
	}
}

func TestResolveEffortMissingPayloadFallsBackToDefault(t *testing.T) {
	effort := resolveEffort(protocol.InputData{})
	if effort != "" {
		t.Fatalf("expected default effort, got %q", effort)
	}
}
