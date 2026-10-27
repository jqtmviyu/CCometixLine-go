package segments

import (
	"fmt"

	"ccometixline-go/internal/observation"
)

type EnvironmentSegment struct{ Counts observation.EnvironmentCounts }

func (s EnvironmentSegment) Collect() *SegmentData {
	counts := s.Counts
	if counts.MemoryFiles == 0 && counts.MCPs == 0 && counts.Skills == 0 && counts.Plugins == 0 {
		return nil
	}

	skillsValue := fmt.Sprintf("%d", counts.Skills)
	pluginsValue := fmt.Sprintf("%d", counts.Plugins)

	return &SegmentData{
		Primary: fmt.Sprintf("memory %d · skill %s · mcp %d · plugin %s", counts.MemoryFiles, skillsValue, counts.MCPs, pluginsValue),
		Metadata: map[string]string{
			"memory_files": fmt.Sprintf("%d", counts.MemoryFiles),
			"skills":       skillsValue,
			"mcps":         fmt.Sprintf("%d", counts.MCPs),
			"plugins":      pluginsValue,
		},
	}
}
