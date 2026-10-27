package segments

import (
	"testing"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func TestCollectTaskMetrics(t *testing.T) {
	cfg := config.DefaultConfig()
	task := protocol.Task{
		ID:           "a1",
		Name:         "search",
		Kind:         "local_agent",
		AgentType:    "Explore",
		Status:       "running",
		Description:  "Look for config",
		Label:        "Reading files",
		StartTime:    float64(1_700_000_000_000),
		TokenSamples: []float64{100_000, 125_000},
	}
	items := CollectTask(cfg, task, time.UnixMilli(1_700_000_012_000))
	values := make(map[config.SegmentID]string)
	for _, item := range items {
		values[item.Config.ID] = item.Data.Primary
	}
	if values[config.SegmentTaskLabel] != "Reading files" || values[config.SegmentTaskStatus] != "🏃 running" {
		t.Fatalf("unexpected task fields: %#v", values)
	}
	if values[config.SegmentTaskName] != "search · agent · Explore" || values[config.SegmentTaskElapsed] != "12s" || values[config.SegmentTaskTrend] != "+25k" {
		t.Fatalf("unexpected task metrics: %#v", values)
	}
}
