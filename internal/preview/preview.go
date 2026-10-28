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
	return strings.TrimSuffix(output.String(), "\n")
}

// 示例只提供固定事实，格式、字段隐藏和配色都走真实显示逻辑。
func sampleFacts() (segments.Facts, config.ModelConfig) {
	now := time.UnixMilli(1_700_000_018_000)
	tokens, limit, percentage := uint64(124_000), uint32(200_000), 62.0
	cost, duration, added, removed := 0.42, uint64(1_122_000), uint32(24), uint32(6)
	observed := true
	requests := uint64(10)
	ratio := 0.92
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
			CachingObserved: &observed, Requests: &requests, HitRatio: &ratio,
		},
	}
	models := config.ModelConfig{ModelEntries: []config.ModelEntry{{
		Pattern: "preview-model", Match: "exact", DisplayName: "Sonnet 4.5", ContextLimit: 200_000,
	}}}
	return segments.Facts{
		Input: input, Now: now,
		Local: observation.Snapshot{
			Effort: "high", Environment: observation.EnvironmentCounts{MemoryFiles: 3, Skills: 5, MCPs: 2, Plugins: 1},
		},
		Git: &segments.GitInfo{Branch: "main", Status: segments.GitStatusDirty},
	}, models
}
