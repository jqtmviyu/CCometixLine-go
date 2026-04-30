package segments

import (
	"testing"

	"ccometixline-go/internal/protocol"
)

func TestEnvironmentSegmentCollectUnknownState(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	segment := EnvironmentSegment{Input: protocol.InputData{Workspace: protocol.Workspace{CurrentDir: t.TempDir()}}}

	data := segment.Collect()
	if data != nil {
		t.Fatalf("expected nil data for empty unknown environment, got %#v", data)
	}
}

func TestEnvironmentSegmentFormattingWithoutPipes(t *testing.T) {
	t.Setenv("CLAUDE_CONFIG_DIR", t.TempDir())
	workspace := t.TempDir()
	segment := EnvironmentSegment{Input: protocol.InputData{Workspace: protocol.Workspace{CurrentDir: workspace}}}

	data := segment.Collect()
	if data != nil && data.Primary != "" && data.Primary != "0 mem · ? skills · 0 mcp · ? plugins" {
		t.Fatalf("unexpected primary: %q", data.Primary)
	}
}
