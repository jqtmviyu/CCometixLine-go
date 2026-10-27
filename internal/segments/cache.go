package segments

import (
	"fmt"
	"strings"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

type CacheSegment struct {
	Input protocol.InputData
	Now   time.Time
}

func (s CacheSegment) Collect() *SegmentData {
	cache := s.Input.PromptCache
	if cache == nil || cache.CachingObserved != nil && !*cache.CachingObserved {
		return nil
	}

	now := s.Now
	if now.IsZero() {
		now = time.Now()
	}
	var parts []string
	metadata := make(map[string]string)
	if cache.TTL != nil {
		metadata["ttl"] = *cache.TTL
	}
	if cache.Warm != nil {
		if *cache.Warm {
			state := "warm"
			if cache.ExpiresAt != nil {
				left := int64(*cache.ExpiresAt - float64(now.Unix()))
				if left > 0 {
					state += " " + formatDuration(uint64(left)*1000)
				} else {
					state = "cold"
				}
			}
			parts = append(parts, state)
			metadata["state"] = state
		} else {
			parts = append(parts, "cold")
			metadata["state"] = "cold"
		}
	}
	if cache.HitRatio != nil && cache.Requests != nil && *cache.Requests > 1 && *cache.HitRatio >= 0 && *cache.HitRatio <= 1 {
		hit := fmt.Sprintf("%.0f%%", *cache.HitRatio*100)
		parts = append(parts, "hit "+hit)
		metadata["hit_ratio"] = fmt.Sprintf("%v", *cache.HitRatio)
	}
	if cache.Misses != nil && *cache.Misses > 0 {
		misses := fmt.Sprintf("miss %d", *cache.Misses)
		parts = append(parts, misses)
		metadata["misses"] = fmt.Sprintf("%d", *cache.Misses)
	}
	if cache.LastMissCause != nil && len(cache.LastMissCause.Causes) > 0 {
		cause := strings.ReplaceAll(cache.LastMissCause.Causes[0], "_", " ")
		parts = append(parts, cause)
		metadata["cause"] = cause
	}
	if len(parts) == 0 {
		return nil
	}

	return &SegmentData{Primary: strings.Join(parts, " · "), Metadata: metadata}
}

type CacheColdCostSegment struct {
	Input       protocol.InputData
	ModelConfig config.ModelConfig
}

func (s CacheColdCostSegment) Collect() *SegmentData {
	cache := s.Input.PromptCache
	if cache == nil || cache.CachingObserved != nil && !*cache.CachingObserved || cache.TTL == nil || cache.RecacheTokensIfCold == nil {
		return nil
	}
	cost, ok := s.ModelConfig.CacheWriteCost(s.Input.Model.ID, *cache.TTL, *cache.RecacheTokensIfCold)
	if !ok {
		return nil
	}

	return &SegmentData{
		Primary: fmt.Sprintf("~$%.2f", cost),
		Metadata: map[string]string{
			"usd":    fmt.Sprintf("%v", cost),
			"tokens": fmt.Sprintf("%d", *cache.RecacheTokensIfCold),
			"ttl":    *cache.TTL,
		},
	}
}
