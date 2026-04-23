package segments

import (
	"fmt"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

type EnvironmentSegment struct {
	Input protocol.InputData
}

func (s EnvironmentSegment) Collect() *SegmentData {
	counts := config.CountEnvironment(s.Input.Workspace.CurrentDir)
	if counts.MemoryFiles == 0 && counts.MCPs == 0 && !counts.SkillsKnown && !counts.PluginsKnown {
		return nil
	}

	skillsValue := "?"
	if counts.SkillsKnown {
		skillsValue = fmt.Sprintf("%d", counts.Skills)
	}

	pluginsValue := "?"
	if counts.PluginsKnown {
		pluginsValue = fmt.Sprintf("%d", counts.Plugins)
	}

	return &SegmentData{
		Primary: fmt.Sprintf("mem:%d mcp:%d skills:%s plugins:%s", counts.MemoryFiles, counts.MCPs, skillsValue, pluginsValue),
		Metadata: map[string]string{
			"memory_files":  fmt.Sprintf("%d", counts.MemoryFiles),
			"mcps":          fmt.Sprintf("%d", counts.MCPs),
			"skills":        skillsValue,
			"plugins":       pluginsValue,
			"skills_known":  fmt.Sprintf("%t", counts.SkillsKnown),
			"plugins_known": fmt.Sprintf("%t", counts.PluginsKnown),
		},
	}
}

