package segments

import (
	"fmt"

	"ccometixline-go/internal/protocol"
)

type SessionSegment struct {
	Input protocol.InputData
}

func (s SessionSegment) Collect() *SegmentData {
	if s.Input.Cost == nil || s.Input.Cost.TotalDurationMS == nil {
		return nil
	}

	metadata := map[string]string{
		"duration_ms": fmt.Sprintf("%d", *s.Input.Cost.TotalDurationMS),
	}
	if s.Input.Cost.TotalAPIDurationMS != nil {
		metadata["api_duration_ms"] = fmt.Sprintf("%d", *s.Input.Cost.TotalAPIDurationMS)
	}
	if s.Input.Cost.TotalLinesAdded != nil {
		metadata["lines_added"] = fmt.Sprintf("%d", *s.Input.Cost.TotalLinesAdded)
	}
	if s.Input.Cost.TotalLinesRemoved != nil {
		metadata["lines_removed"] = fmt.Sprintf("%d", *s.Input.Cost.TotalLinesRemoved)
	}

	return &SegmentData{
		Primary:   formatDuration(*s.Input.Cost.TotalDurationMS),
		Secondary: formatLineChanges(s.Input.Cost),
		Metadata:  metadata,
	}
}

func formatDuration(ms uint64) string {
	if ms < 1000 {
		return fmt.Sprintf("%dms", ms)
	}
	if ms < 60000 {
		return fmt.Sprintf("%ds", ms/1000)
	}
	if ms < 3600000 {
		minutes := ms / 60000
		seconds := (ms % 60000) / 1000
		if seconds == 0 {
			return fmt.Sprintf("%dm", minutes)
		}
		return fmt.Sprintf("%dm%ds", minutes, seconds)
	}

	hours := ms / 3600000
	minutes := (ms % 3600000) / 60000
	if minutes == 0 {
		return fmt.Sprintf("%dh", hours)
	}
	return fmt.Sprintf("%dh%dm", hours, minutes)
}

func formatLineChanges(cost *protocol.Cost) string {
	if cost == nil {
		return ""
	}

	added := uint32(0)
	removed := uint32(0)
	if cost.TotalLinesAdded != nil {
		added = *cost.TotalLinesAdded
	}
	if cost.TotalLinesRemoved != nil {
		removed = *cost.TotalLinesRemoved
	}

	switch {
	case added > 0 && removed > 0:
		return fmt.Sprintf("\x1b[32m+%d\x1b[0m \x1b[31m-%d\x1b[0m", added, removed)
	case added > 0:
		return fmt.Sprintf("\x1b[32m+%d\x1b[0m", added)
	case removed > 0:
		return fmt.Sprintf("\x1b[31m-%d\x1b[0m", removed)
	default:
		return ""
	}
}
