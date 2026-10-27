package segments

import (
	"fmt"
	"strings"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func CollectTask(cfg config.Config, task protocol.Task, now time.Time) []CollectedSegment {
	settings := make(map[config.SegmentID]config.SegmentConfig, len(cfg.Segments))
	for _, segment := range cfg.Segments {
		settings[segment.ID] = segment
	}
	for _, segment := range config.DefaultConfig().Segments {
		switch segment.ID {
		case config.SegmentTaskLabel, config.SegmentTaskName, config.SegmentTaskStatus, config.SegmentTaskElapsed, config.SegmentTaskTrend:
			if _, exists := settings[segment.ID]; !exists {
				settings[segment.ID] = segment
			}
		}
	}

	var items []CollectedSegment
	for _, id := range cfg.Layout.Task {
		segmentConfig, ok := settings[id]
		if !ok || !segmentConfig.Enabled {
			continue
		}
		data := collectTaskMetric(id, task, now)
		if data != nil {
			items = append(items, CollectedSegment{Config: segmentConfig, Data: *data})
		}
	}
	return items
}

func collectTaskMetric(id config.SegmentID, task protocol.Task, now time.Time) *SegmentData {
	switch id {
	case config.SegmentTaskLabel:
		value := task.Label
		if value == "" {
			value = task.Description
		}
		if value == "" {
			return nil
		}
		return &SegmentData{Primary: value, Metadata: map[string]string{"label": value}}
	case config.SegmentTaskName:
		value := task.Name
		if value == "" {
			value = task.Description
		}
		if value == "" {
			return nil
		}
		kind := taskKind(task.Kind)
		if kind != "" {
			value += " · " + kind
		}
		if task.AgentType != "" {
			value += " · " + task.AgentType
		}
		return &SegmentData{Primary: value, Metadata: map[string]string{"name": task.Name, "kind": kind, "agent": task.AgentType}}
	case config.SegmentTaskStatus:
		status := task.Status
		if status == "" {
			status = "running"
		}
		mark := "●"
		switch status {
		case "running":
			mark = "🏃"
		case "completed":
			mark = "✓"
		case "failed", "killed", "error":
			mark = "✗"
		}
		return &SegmentData{Primary: mark + " " + status, Metadata: map[string]string{"status": status}}
	case config.SegmentTaskElapsed:
		if !taskIsRunning(task.Status) || task.StartTime <= 0 {
			return nil
		}
		elapsed := now.UnixMilli() - int64(task.StartTime)
		if elapsed < 0 {
			return nil
		}
		return &SegmentData{Primary: formatDuration(uint64(elapsed)), Metadata: map[string]string{"elapsed_ms": fmt.Sprintf("%d", elapsed)}}
	case config.SegmentTaskTrend:
		if len(task.TokenSamples) < 2 {
			return nil
		}
		delta := task.TokenSamples[len(task.TokenSamples)-1] - task.TokenSamples[0]
		value := formatSignedMetricTokens(delta)
		return &SegmentData{Primary: value, Metadata: map[string]string{"delta": value}}
	default:
		return nil
	}
}

func taskIsRunning(status string) bool {
	return status == "" || status == "running" || status == "pending"
}

func taskKind(kind string) string {
	switch kind {
	case "local_agent", "remote_agent":
		return "agent"
	case "local_bash":
		return "shell"
	case "local_workflow":
		return "workflow"
	case "in_process_teammate":
		return "teammate"
	default:
		return kind
	}
}

func formatSignedMetricTokens(value float64) string {
	sign := "+"
	if value < 0 {
		sign = "-"
		value = -value
	}
	if value >= 1000 {
		if value == float64(uint64(value/1000))*1000 {
			return fmt.Sprintf("%s%dk", sign, uint64(value/1000))
		}
		return fmt.Sprintf("%s%.1fk", sign, value/1000)
	}
	return sign + strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.1f", value), "0"), ".")
}
