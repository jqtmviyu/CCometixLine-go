package protocol

import (
	"encoding/json"
	"testing"
)

func TestInputDataParsesPromptCache(t *testing.T) {
	payload := []byte(`{"session_id":"s1","prompt_cache":{"warm":true,"caching_observed":true,"ttl":"5m","expires_at":1700000300,"requests":4,"misses":1,"hit_ratio":0.75,"last_miss_cause":{"causes":["ttl_expired_5m"]},"recache_tokens_if_cold":12000}}`)
	var input InputData
	if err := json.Unmarshal(payload, &input); err != nil {
		t.Fatal(err)
	}
	if input.SessionID != "s1" || input.PromptCache == nil {
		t.Fatalf("unexpected input: %#v", input)
	}
	cache := input.PromptCache
	if cache.Warm == nil || !*cache.Warm || cache.TTL == nil || *cache.TTL != "5m" {
		t.Fatalf("unexpected cache state: %#v", cache)
	}
	if cache.HitRatio == nil || *cache.HitRatio != 0.75 || cache.RecacheTokensIfCold == nil || *cache.RecacheTokensIfCold != 12000 {
		t.Fatalf("unexpected cache metrics: %#v", cache)
	}
	if cache.LastMissCause == nil || len(cache.LastMissCause.Causes) != 1 || cache.LastMissCause.Causes[0] != "ttl_expired_5m" {
		t.Fatalf("unexpected miss cause: %#v", cache.LastMissCause)
	}
}
