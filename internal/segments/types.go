package segments

type SegmentData struct {
	Primary   string
	Secondary string
	Metadata  map[string]string
}

type Segment interface {
	Collect() *SegmentData
}
