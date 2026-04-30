package segments

import (
	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func CollectAll(cfg config.Config, modelConfig config.ModelConfig, input protocol.InputData) []CollectedSegment {
	results := []CollectedSegment{}

	for _, segmentConfig := range cfg.Segments {
		if !segmentConfig.Enabled {
			continue
		}

		var segment Segment
		switch segmentConfig.ID {
		case config.SegmentModel:
			segment = ModelSegment{Input: input, ModelConfig: modelConfig}
		case config.SegmentEffort:
			segment = EffortSegment{Input: input}
		case config.SegmentDirectory:
			segment = DirectorySegment{Input: input}
		case config.SegmentGit:
			segment = GitSegment{Input: input}
		case config.SegmentContextWindow:
			segment = ContextWindowSegment{Input: input, ModelConfig: modelConfig}
		case config.SegmentCost:
			segment = CostSegment{Input: input}
		case config.SegmentSession:
			segment = SessionSegment{Input: input}
		case config.SegmentEnvironment:
			segment = EnvironmentSegment{Input: input}
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

type CollectedSegment struct {
	Config config.SegmentConfig
	Data   SegmentData
}
