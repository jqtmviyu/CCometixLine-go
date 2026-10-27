package preview

import (
	"fmt"
	"strings"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
	"ccometixline-go/internal/render"
	"ccometixline-go/internal/segments"
)

var samples = map[config.SegmentID]segments.SegmentData{
	config.SegmentModel:         {Primary: "Sonnet 4.5"},
	config.SegmentEffort:        {Primary: "high"},
	config.SegmentDirectory:     {Primary: "sample-project"},
	config.SegmentGit:           {Primary: "main", Secondary: "●"},
	config.SegmentContextWindow: {Primary: "62% · 124k tokens"},
	config.SegmentCost:          {Primary: "$0.42"},
	config.SegmentSession:       {Primary: "18m42s", Secondary: "+24 -6"},
	config.SegmentEnvironment:   {Primary: "memory 3 · skill 5 · mcp 2 · plugin 1"},
	config.SegmentCache:         {Primary: "warm 4m · hit 92% · miss 1"},
	config.SegmentCacheColdCost: {Primary: "~$0.08"},
}

func Render(cfg config.Config) string {
	items := make([]segments.CollectedSegment, 0, len(cfg.Segments))
	for _, segmentConfig := range cfg.Segments {
		if !segmentConfig.Enabled {
			continue
		}
		data, ok := samples[segmentConfig.ID]
		if !ok {
			continue
		}
		items = append(items, segments.CollectedSegment{Config: segmentConfig, Data: data})
	}

	generator := render.StatusLineGenerator{Config: cfg}
	var output strings.Builder
	fmt.Fprintln(&output, "CCometixLine Go · 配置预览（示例数据，非实时会话）")
	fmt.Fprintln(&output, "statusLine")
	statusLine := generator.Generate(items)
	if statusLine == "" {
		fmt.Fprintln(&output, "(没有启用且可预览的 segment)")
	} else {
		fmt.Fprintln(&output, statusLine)
	}

	task := protocol.Task{
		ID:           "preview-task",
		Name:         "Inspect cache behavior",
		Kind:         "local_agent",
		AgentType:    "Explore",
		Status:       "running",
		Description:  "Example background task",
		Label:        "Reviewing model config",
		StartTime:    1_700_000_000_000,
		TokenSamples: []float64{88_000, 94_000},
	}
	taskItems := segments.CollectTask(cfg, task, time.UnixMilli(1_700_000_018_000))
	if taskLine := generator.GenerateTaskLine(taskItems, 100); taskLine != "" {
		fmt.Fprintln(&output, "subagentStatusLine")
		fmt.Fprintln(&output, taskLine)
	}

	return strings.TrimSuffix(output.String(), "\n")
}
