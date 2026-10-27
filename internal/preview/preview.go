package preview

import (
	"fmt"
	"strings"
	"time"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/observation"
	"ccometixline-go/internal/protocol"
	"ccometixline-go/internal/render"
	"ccometixline-go/internal/segments"
)

func Render(cfg config.Config) string {
	layout := cfg.ResolveLayout()
	facts, models := sampleFacts()
	collector := segments.Collector{Layout: layout, Models: models}
	generator := render.StatusLineGenerator{Style: cfg.Style, Layout: layout}
	var output strings.Builder
	fmt.Fprintln(&output, "CCometixLine Go · 配置预览（示例数据，非实时会话）")
	fmt.Fprintln(&output, "statusLine")
	if text := generator.Generate(collector.Format(facts)); text != "" {
		fmt.Fprintln(&output, text)
	} else {
		fmt.Fprintln(&output, "(没有启用且可预览的 segment)")
	}
	task := protocol.Task{
		ID:           "preview-task",
		Name:         "Inspect cache behavior",
		Kind:         "local_agent",
		AgentType:    "Explore",
		Status:       "running",
		Description:  "Example background task",
		Label:        "Reviewing model config",
		StartTime:    1_700_000_000_000,
		TokenSamples: []float64{88_000, 94_000},
	}
	items := segments.CollectTask(layout.Task, task, facts.Now)
	if text := generator.GenerateTaskLine(items, 100); text != "" {
		fmt.Fprintln(&output, "subagentStatusLine")
		fmt.Fprintln(&output, text)
	}
	return strings.TrimSuffix(output.String(), "\n")
}

// 示例只提供固定事实，格式、字段隐藏和配色都走真实显示逻辑。
func sampleFacts() (segments.Facts, config.ModelConfig) {
	now := time.UnixMilli(1_700_000_018_000)
	tokens, limit, percentage := uint64(124_000), uint32(200_000), 62.0
	cost, duration, added, removed := 0.42, uint64(1_122_000), uint32(24), uint32(6)
	warm, observed := true, true
	expiry := float64(now.Unix() + 240)
	requests, misses := uint64(10), uint64(1)
	ratio, ttl, recache := 0.92, "5m", uint64(25_600)
	input := protocol.InputData{
		Model:     protocol.Model{ID: "preview-model", DisplayName: "Sonnet 4.5"},
		Workspace: protocol.Workspace{CurrentDir: "sample-project"},
		ContextWindow: &protocol.ContextWindow{
			TotalInputTokens: &tokens, ContextWindowSize: &limit, UsedPercentage: &percentage,
		},
		Cost: &protocol.Cost{
			TotalCostUSD: &cost, TotalDurationMS: &duration, TotalLinesAdded: &added, TotalLinesRemoved: &removed,
		},
		PromptCache: &protocol.PromptCache{
			Warm: &warm, CachingObserved: &observed, ExpiresAt: &expiry,
			Requests: &requests, HitRatio: &ratio, Misses: &misses,
			TTL: &ttl, RecacheTokensIfCold: &recache,
		},
	}
	models := config.ModelConfig{ModelEntries: []config.ModelEntry{{
		Pattern: "preview-model", Match: "exact", DisplayName: "Sonnet 4.5", ContextLimit: 200_000,
		InputPriceUSDPerMillion: 2.5, CacheWriteMultipliers: map[string]float64{"5m": 1.25},
	}}}
	return segments.Facts{
		Input: input, Now: now,
		Local: observation.Snapshot{
			Effort: "high", Environment: observation.EnvironmentCounts{MemoryFiles: 3, Skills: 5, MCPs: 2, Plugins: 1},
		},
		Git: &segments.GitInfo{Branch: "main", Status: segments.GitStatusDirty},
	}, models
}
