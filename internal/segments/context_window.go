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
	contextWindow := s.Input.ContextWindow
	var reportedLimit uint32
	var reportedPercentage *float64
	var used uint64
	var hasUsage bool
	if contextWindow != nil {
		if contextWindow.ContextWindowSize != nil {
			reportedLimit = *contextWindow.ContextWindowSize
		}
		reportedPercentage = contextWindow.UsedPercentage
		used, hasUsage = contextWindow.ContextTokens()
	}

	contextLimit, corrected := s.ModelConfig.ResolveContextLimit(s.Input.Model.ID, reportedLimit)
	percentage, hasPercentage := contextPercentage(reportedPercentage, used, hasUsage, contextLimit, corrected)
	if !hasPercentage {
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

	percentageDisplay := fmt.Sprintf("%.1f%%", percentage)
	if percentage == float64(uint64(percentage)) {
		percentageDisplay = fmt.Sprintf("%.0f%%", percentage)
	}

	tokensDisplay := "-"
	tokensMetadata := "-"
	if hasUsage {
		tokensMetadata = fmt.Sprintf("%d", used)
		tokensDisplay = tokensMetadata
		if used >= 1000 {
			k := float64(used) / 1000.0
			if used%1000 == 0 {
				tokensDisplay = fmt.Sprintf("%dk", uint64(k))
			} else {
				tokensDisplay = fmt.Sprintf("%.1fk", k)
			}
		}
	}

	primary := percentageDisplay + " · " + tokensDisplay + " tokens"
	metadata := map[string]string{
		"limit":      fmt.Sprintf("%d", contextLimit),
		"model":      s.Input.Model.ID,
		"tokens":     tokensMetadata,
		"percentage": fmt.Sprintf("%v", percentage),
	}
	if hit := cacheHitDisplay(s.Input.PromptCache); hit != "" {
		primary += " · " + hit
		metadata["hit_ratio"] = fmt.Sprintf("%v", *s.Input.PromptCache.HitRatio)
	}

	return &SegmentData{Primary: primary, Metadata: metadata}
}

// 命中率并进 tokens 段，共用同一个图标和配色；缓存未观测或请求数不足 2 次时不显示。
func cacheHitDisplay(cache *protocol.PromptCache) string {
	if cache == nil || cache.CachingObserved != nil && !*cache.CachingObserved {
		return ""
	}
	if cache.HitRatio == nil || cache.Requests == nil || *cache.Requests <= 1 || *cache.HitRatio < 0 || *cache.HitRatio > 1 {
		return ""
	}
	return fmt.Sprintf("hit %.0f%%", *cache.HitRatio*100)
}

func contextPercentage(reported *float64, used uint64, hasUsage bool, limit uint32, corrected bool) (float64, bool) {
	if reported != nil && validPercentage(*reported) && !corrected {
		return *reported, true
	}
	if hasUsage && limit > 0 {
		percentage := float64(used) / float64(limit) * 100.0
		if percentage > 100 {
			percentage = 100
		}
		return percentage, true
	}
	if reported != nil && validPercentage(*reported) {
		return *reported, true
	}

	return 0, false
}

func validPercentage(value float64) bool {
	return value >= 0 && value <= 100
}
