package segments

import (
	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

type ModelSegment struct {
	Input       protocol.InputData
	ModelConfig config.ModelConfig
}

func (s ModelSegment) Collect() *SegmentData {
	metadata := map[string]string{
		"model_id":     s.Input.Model.ID,
		"display_name": s.Input.Model.DisplayName,
	}

	return &SegmentData{
		Primary:   s.formatModelName(),
		Secondary: "",
		Metadata:  metadata,
	}
}

func (s ModelSegment) formatModelName() string {
	if name, ok := s.ModelConfig.GetDisplayName(s.Input.Model.ID); ok {
		return name
	}

	base := s.Input.Model.DisplayName
	if base == "" {
		base = s.Input.Model.ID
	}

	if suffix, ok := s.ModelConfig.GetDisplaySuffix(s.Input.Model.ID); ok {
		return base + suffix
	}

	return base
}
