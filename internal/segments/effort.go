package segments

type EffortSegment struct{ Level string }

func (s EffortSegment) Collect() *SegmentData {
	level := s.Level
	if level == "" {
		level = "auto"
	}
	return &SegmentData{Primary: level, Metadata: map[string]string{"effort": level}}
}
