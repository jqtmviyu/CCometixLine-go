package config

// ResolvedLayout 保存本次运行生效的字段。
type ResolvedLayout struct {
	Lines    []LayoutLine
	Segments []SegmentConfig
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
	return layout
}
