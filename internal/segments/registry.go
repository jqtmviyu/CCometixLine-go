package segments

import (
	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func CollectAll(cfg config.Config, modelConfig config.ModelConfig, input protocol.InputData) []CollectedSegment {
	results := []CollectedSegment{}
	selectedSegments := collectableSegments(cfg)
	settings := loadSettingsSnapshot(selectedSegments, input, cfg.Cache)

	for _, segmentConfig := range selectedSegments {
		var segment Segment
		switch segmentConfig.ID {
		case config.SegmentModel:
			segment = ModelSegment{Input: input, ModelConfig: modelConfig}
		case config.SegmentEffort:
			segment = EffortSegment{Input: input, Settings: settings}
		case config.SegmentDirectory:
			segment = DirectorySegment{Input: input}
		case config.SegmentGit:
			segment = GitSegment{Input: input, CacheTTL: cfg.Cache.GitTTL()}
		case config.SegmentContextWindow:
			segment = ContextWindowSegment{Input: input, ModelConfig: modelConfig}
		case config.SegmentCost:
			segment = CostSegment{Input: input}
		case config.SegmentSession:
			segment = SessionSegment{Input: input}
		case config.SegmentEnvironment:
			segment = EnvironmentSegment{Input: input, Settings: settings}
		case config.SegmentCache:
			segment = CacheSegment{Input: input}
		case config.SegmentCacheColdCost:
			segment = CacheColdCostSegment{Input: input, ModelConfig: modelConfig}
		default:
			continue
		}

		data := segment.Collect()
		if data == nil {
			continue
		}
		results = append(results, CollectedSegment{Config: segmentConfig, Data: *data})
	}

	return results
}

// An absent layout keeps the legacy behavior of collecting every enabled segment.
func collectableSegments(cfg config.Config) []config.SegmentConfig {
	var selectedIDs map[config.SegmentID]struct{}
	if len(cfg.Layout.Lines) > 0 {
		selectedIDs = make(map[config.SegmentID]struct{})
		for _, line := range cfg.Layout.Lines {
			for _, id := range line.Segments {
				selectedIDs[id] = struct{}{}
			}
		}
	}

	selected := make([]config.SegmentConfig, 0, len(cfg.Segments))
	for _, segmentConfig := range cfg.Segments {
		if !segmentConfig.Enabled {
			continue
		}
		if selectedIDs != nil {
			if _, ok := selectedIDs[segmentConfig.ID]; !ok {
				continue
			}
		}
		selected = append(selected, segmentConfig)
	}
	return selected
}

func loadSettingsSnapshot(selected []config.SegmentConfig, input protocol.InputData, cache config.CacheConfig) config.SettingsSnapshot {
	segmentIDs := make([]config.SegmentID, 0, len(selected))
	_, hasDirectEffort := directEffort(input)
	for _, segment := range selected {
		if segment.ID == config.SegmentEffort && hasDirectEffort {
			continue
		}
		segmentIDs = append(segmentIDs, segment.ID)
	}
	return config.LoadSettingsSnapshotWithCache(input.Workspace.CurrentDir, segmentIDs, cache.EnvironmentTTL())
}

type CollectedSegment struct {
	Config config.SegmentConfig
	Data   SegmentData
}
