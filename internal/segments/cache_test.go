package segments

import (
	"testing"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

func TestCacheSegmentShowsWarmHitAndMissCause(t *testing.T) {
	warm := true
	observed := true
	ttl := "5m"
	expires := float64(1_700_000_300)
	requests := uint64(4)
	misses := uint64(1)
	hitRatio := 0.75
	segment := CacheSegment{
		Input: protocol.InputData{PromptCache: &protocol.PromptCache{
			Warm: &warm, CachingObserved: &observed, TTL: &ttl, ExpiresAt: &expires,
			Requests: &requests, Misses: &misses, HitRatio: &hitRatio,
			LastMissCause: &protocol.MissCause{Causes: []string{"ttl_expired_5m"}},
		}},
		Now: time.Unix(1_700_000_000, 0),
	}

	data := segment.Collect()
	if data == nil || data.Primary != "warm 5m · hit 75% · miss 1 · ttl expired 5m" {
		t.Fatalf("unexpected cache display: %#v", data)
	}
}

func TestCacheSegmentHidesUnobservedCache(t *testing.T) {
	observed := false
	segment := CacheSegment{Input: protocol.InputData{PromptCache: &protocol.PromptCache{CachingObserved: &observed}}}
	if data := segment.Collect(); data != nil {
		t.Fatalf("expected no cache metric when cache is unobserved, got %#v", data)
	}
}

func TestCacheColdCostRequiresModelPricing(t *testing.T) {
	ttl := "1h"
	tokens := uint64(250_000)
	input := protocol.InputData{
		Model:       protocol.Model{ID: "provider/model-x"},
		PromptCache: &protocol.PromptCache{TTL: &ttl, RecacheTokensIfCold: &tokens},
	}
	priced := config.ModelConfig{ModelEntries: []config.ModelEntry{{
		Pattern:                 "model-x",
		InputPriceUSDPerMillion: 2,
		CacheWriteMultipliers:   map[string]float64{"1h": 2},
	}}}
	data := (CacheColdCostSegment{Input: input, ModelConfig: priced}).Collect()
	if data == nil || data.Primary != "~$1.00" {
		t.Fatalf("unexpected cold cache cost: %#v", data)
	}
	if data := (CacheColdCostSegment{Input: input}).Collect(); data != nil {
		t.Fatalf("expected missing price to hide cost, got %#v", data)
	}
}
