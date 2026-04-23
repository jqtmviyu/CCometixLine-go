package segments

import (
	"fmt"

	"ccometixline-go/internal/protocol"
)

type CostSegment struct {
	Input protocol.InputData
}

func (s CostSegment) Collect() *SegmentData {
	if s.Input.Cost == nil || s.Input.Cost.TotalCostUSD == nil {
		return nil
	}

	cost := *s.Input.Cost.TotalCostUSD
	primary := "$0"
	if cost >= 0.01 {
		primary = fmt.Sprintf("$%.2f", cost)
	}

	return &SegmentData{
		Primary:   primary,
		Secondary: "",
		Metadata: map[string]string{
			"cost": fmt.Sprintf("%v", cost),
		},
	}
}
