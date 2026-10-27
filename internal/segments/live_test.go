package segments

import (
	"testing"
	"time"

	"ccometixline-go/internal/protocol"
)

func TestContextBreakdownSortsUsedCategories(t *testing.T) {
	used := uint64(100_000)
	window := uint64(200_000)
	bridge := &protocol.Bridge{Context: &protocol.BridgeContext{
		Used: &used, Window: &window,
		Categories: []protocol.BridgeCategory{
			{Name: "System prompt", Kind: "used", Tokens: 3_000},
			{Name: "Messages", Kind: "used", Tokens: 90_000},
			{Name: "System tools", Kind: "used", Tokens: 7_000},
			{Name: "Free space", Kind: "free", Tokens: 100_000},
		},
	}}
	data := (ContextBreakdownSegment{Bridge: bridge}).Collect()
	if data == nil || data.Primary != "msgs 90k · tools 7k · sys 3k" {
		t.Fatalf("unexpected context breakdown: %#v", data)
	}
}

func TestTimingSegmentsReadBridgeTurn(t *testing.T) {
	ttft := uint64(820)
	tps := 74.2
	bridge := &protocol.Bridge{Turn: &protocol.BridgeTurn{TTFTMS: &ttft, TPS: &tps}}
	if got := (TimingTTFTSegment{Bridge: bridge}).Collect().Primary; got != "0.8s" {
		t.Fatalf("unexpected TTFT: %q", got)
	}
	if got := (TimingTPSSegment{Bridge: bridge}).Collect().Primary; got != "74 tok/s" {
		t.Fatalf("unexpected TPS: %q", got)
	}
}

func TestActivitySegmentsShowRunningToolsAndAgents(t *testing.T) {
	bridge := &protocol.Bridge{
		Tools: []protocol.BridgeTool{
			{Name: "Bash", Since: 1_700_000_000_000},
			{Name: "Edit", Since: 1_700_000_010_000},
			{Name: "Read", Since: 1_700_000_020_000},
		},
		Agents: []protocol.BridgeAgent{
			{ID: "a1", Type: "Explore", Status: "running"},
			{ID: "a2", Type: "Explore", Status: "waiting"},
			{ID: "a3", Type: "Worker", Status: "completed"},
		},
	}
	toolData := (ActivityToolsSegment{Bridge: bridge, Now: time.UnixMilli(1_700_000_030_000)}).Collect()
	if toolData == nil || toolData.Primary != "Bash · Edit · +1 · 30s" {
		t.Fatalf("unexpected active tools: %#v", toolData)
	}
	agentData := (ActivityAgentsSegment{Bridge: bridge}).Collect()
	if agentData == nil || agentData.Primary != "2 running · Explore" {
		t.Fatalf("unexpected active agents: %#v", agentData)
	}
}
