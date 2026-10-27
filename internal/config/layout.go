package config

// ResolvedLayout 保存本次运行生效的字段；任务配置在遍历任务之前补齐。
type ResolvedLayout struct {
	Lines    []LayoutLine
	Segments []SegmentConfig
	Task     []SegmentConfig
}

func (cfg Config) ResolveLayout() ResolvedLayout {
	layout := ResolvedLayout{Lines: cfg.Layout.Lines}
	if len(layout.Lines) == 0 {
		layout.Lines = defaultLayout().Lines
	}
	selected := map[SegmentID]bool{}
	for _, line := range layout.Lines {
		for _, id := range line.Segments {
			selected[id] = true
		}
	}
	for _, segment := range cfg.Segments {
		if segment.Enabled && selected[segment.ID] {
			layout.Segments = append(layout.Segments, segment)
		}
	}
	if len(cfg.Layout.Task) == 0 {
		return layout
	}
	taskSettings := make(map[SegmentID]SegmentConfig, len(cfg.Segments))
	for _, segment := range cfg.Segments {
		taskSettings[segment.ID] = segment
	}
	for _, d := range defaultSegmentDefs {
		switch d.id {
		case SegmentTaskLabel, SegmentTaskName, SegmentTaskStatus, SegmentTaskElapsed, SegmentTaskTrend:
			if _, exists := taskSettings[d.id]; !exists {
				taskSettings[d.id] = defaultSegment(d)
			}
		}
	}
	for _, id := range cfg.Layout.Task {
		if segment, ok := taskSettings[id]; ok && segment.Enabled {
			layout.Task = append(layout.Task, segment)
		}
	}
	return layout
}
