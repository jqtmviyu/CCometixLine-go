package segments

import (
	"fmt"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

type ContextWindowSegment struct {
	Input       protocol.InputData
	ModelConfig config.ModelConfig
}

func (s ContextWindowSegment) Collect() *SegmentData {
	contextLimit := s.ModelConfig.GetContextLimit(s.Input.Model.ID)

	used, rate, ok := parsePayloadContextWindow(s.Input, contextLimit)
	if !ok {
		return &SegmentData{
			Primary: "- · - tokens",
			Metadata: map[string]string{
				"limit":      fmt.Sprintf("%d", contextLimit),
				"model":      s.Input.Model.ID,
				"tokens":     "-",
				"percentage": "-",
			},
		}
	}

	percentageDisplay := fmt.Sprintf("%.1f%%", rate)
	if rate == float64(uint64(rate)) {
		percentageDisplay = fmt.Sprintf("%.0f%%", rate)
	}

	tokensDisplay := fmt.Sprintf("%d", used)
	if used >= 1000 {
		k := float64(used) / 1000.0
		if k == float64(uint64(k)) {
			tokensDisplay = fmt.Sprintf("%dk", int(k))
		} else {
			tokensDisplay = fmt.Sprintf("%.1fk", k)
		}
	}

	return &SegmentData{
		Primary: percentageDisplay + " · " + tokensDisplay + " tokens",
		Metadata: map[string]string{
			"limit":      fmt.Sprintf("%d", contextLimit),
			"model":      s.Input.Model.ID,
			"tokens":     fmt.Sprintf("%d", used),
			"percentage": fmt.Sprintf("%v", rate),
		},
	}
}

func parsePayloadContextWindow(input protocol.InputData, contextLimit uint32) (uint32, float64, bool) {
	if input.ContextWindow == nil || input.ContextWindow.CurrentUsage == nil || contextLimit == 0 {
		return 0, 0, false
	}

	used := input.ContextWindow.CurrentUsage.UsedTokens()
	if used == 0 {
		return 0, 0, false
	}

	return used, (float64(used) / float64(contextLimit)) * 100.0, true
}
