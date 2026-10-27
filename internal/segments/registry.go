package segments

import (
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/observation"
	"ccometixline-go/internal/protocol"
)

type Collector struct {
	Layout config.ResolvedLayout
	Cache  config.CacheConfig
	Models config.ModelConfig
}

// Facts 是一次显示所用的事实；预览提供固定事实，实时路径按布局读取本地资料。
type Facts struct {
	Input protocol.InputData
	Local observation.Snapshot
	Git   *GitInfo
	Now   time.Time
}

func (c Collector) Collect(input protocol.InputData) []CollectedSegment {
	request := observation.Request{CWD: input.Workspace.CurrentDir, CacheTTL: c.Cache.EnvironmentTTL()}
	if input.Effort != nil {
		request.InputEffort = input.Effort.Level
	}
	facts := Facts{Input: input, Now: time.Now()}
	needGit := false
	for _, field := range c.Layout.Segments {
		switch field.ID {
		case config.SegmentEnvironment:
			request.Environment = true
		case config.SegmentModel:
			request.Effort = true
		case config.SegmentGit:
			needGit = true
		}
	}
	if needGit {
		if info, ok := getCachedGitInfo(input.Workspace.CurrentDir, c.Cache.GitTTL(), facts.Now); ok {
			facts.Git = &info
		}
	}
	facts.Local = observation.Read(request)
	return c.Format(facts)
}

func (c Collector) Format(facts Facts) []CollectedSegment {
	results := make([]CollectedSegment, 0, len(c.Layout.Segments))
	for _, segmentConfig := range c.Layout.Segments {
		var segment Segment
		switch segmentConfig.ID {
		case config.SegmentModel:
			segment = ModelSegment{Input: facts.Input, ModelConfig: c.Models, Effort: facts.Local.Effort}
		case config.SegmentDirectory:
			segment = DirectorySegment{Input: facts.Input}
		case config.SegmentGit:
			segment = GitSegment{Info: facts.Git}
		case config.SegmentContextWindow:
			segment = ContextWindowSegment{Input: facts.Input, ModelConfig: c.Models}
		case config.SegmentCost:
			segment = CostSegment{Input: facts.Input}
		case config.SegmentSession:
			segment = SessionSegment{Input: facts.Input}
		case config.SegmentEnvironment:
			segment = EnvironmentSegment{Counts: facts.Local.Environment}
		default:
			continue
		}
		if data := segment.Collect(); data != nil {
			results = append(results, CollectedSegment{Config: segmentConfig, Data: *data})
		}
	}
	return results
}

type CollectedSegment struct {
	Config config.SegmentConfig
	Data   SegmentData
}
