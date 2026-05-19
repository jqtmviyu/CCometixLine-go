package segments

import (
	"fmt"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

type EnvironmentSegment struct {
	Input    protocol.InputData
	Settings config.SettingsSnapshot
}

func (s EnvironmentSegment) Collect() *SegmentData {
	counts := s.Settings.EnvironmentCounts()
	if counts.MemoryFiles == 0 && counts.MCPs == 0 && counts.Skills == 0 && counts.Plugins == 0 {
		return nil
	}

	skillsValue := fmt.Sprintf("%d", counts.Skills)
	pluginsValue := fmt.Sprintf("%d", counts.Plugins)

	return &SegmentData{
		Primary: fmt.Sprintf("%d mem · %s skills · %d mcp · %s plugins", counts.MemoryFiles, skillsValue, counts.MCPs, pluginsValue),
		Metadata: map[string]string{
			"memory_files": fmt.Sprintf("%d", counts.MemoryFiles),
			"skills":       skillsValue,
			"mcps":         fmt.Sprintf("%d", counts.MCPs),
			"plugins":      pluginsValue,
		},
	}
}
