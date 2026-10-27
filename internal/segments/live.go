package segments

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"ccometixline-go/internal/protocol"
)

type ContextBreakdownSegment struct {
	Bridge *protocol.Bridge
}

func (s ContextBreakdownSegment) Collect() *SegmentData {
	if s.Bridge == nil || s.Bridge.Context == nil {
		return nil
	}
	var categories []protocol.BridgeCategory
	for _, category := range s.Bridge.Context.Categories {
		if category.Kind == "used" && category.Tokens > 0 {
			categories = append(categories, category)
		}
	}
	if len(categories) == 0 {
		return nil
	}
	sort.Slice(categories, func(i, j int) bool { return categories[i].Tokens > categories[j].Tokens })
	if len(categories) > 3 {
		categories = categories[:3]
	}

	parts := make([]string, 0, len(categories))
	for _, category := range categories {
		parts = append(parts, shortenCategory(category.Name)+" "+formatMetricTokens(category.Tokens))
	}
	metadata := map[string]string{"top": strings.Join(parts, " · ")}
	if s.Bridge.Context.Used != nil {
		metadata["used"] = fmt.Sprintf("%d", *s.Bridge.Context.Used)
	}
	if s.Bridge.Context.Window != nil {
		metadata["window"] = fmt.Sprintf("%d", *s.Bridge.Context.Window)
	}
	return &SegmentData{Primary: strings.Join(parts, " · "), Metadata: metadata}
}

type TimingTTFTSegment struct {
	Bridge *protocol.Bridge
}

func (s TimingTTFTSegment) Collect() *SegmentData {
	if s.Bridge == nil || s.Bridge.Turn == nil || s.Bridge.Turn.TTFTMS == nil {
		return nil
	}
	ms := *s.Bridge.Turn.TTFTMS
	text := formatDuration(ms)
	if ms < 10_000 {
		text = fmt.Sprintf("%.1fs", float64(ms)/1000)
	}
	return &SegmentData{Primary: text, Metadata: map[string]string{"ttft_ms": fmt.Sprintf("%d", ms)}}
}

type TimingTPSSegment struct {
	Bridge *protocol.Bridge
}

func (s TimingTPSSegment) Collect() *SegmentData {
	if s.Bridge == nil || s.Bridge.Turn == nil || s.Bridge.Turn.TPS == nil || *s.Bridge.Turn.TPS <= 0 {
		return nil
	}
	text := fmt.Sprintf("%.0f tok/s", *s.Bridge.Turn.TPS)
	return &SegmentData{Primary: text, Metadata: map[string]string{"tps": fmt.Sprintf("%v", *s.Bridge.Turn.TPS)}}
}

type ActivityToolsSegment struct {
	Bridge *protocol.Bridge
	Now    time.Time
}

func (s ActivityToolsSegment) Collect() *SegmentData {
	if s.Bridge == nil || len(s.Bridge.Tools) == 0 {
		return nil
	}
	tools := append([]protocol.BridgeTool(nil), s.Bridge.Tools...)
	sort.Slice(tools, func(i, j int) bool { return tools[i].Since < tools[j].Since })
	shown := make([]string, 0, 3)
	for i := 0; i < len(tools) && i < 2; i++ {
		shown = append(shown, tools[i].Name)
	}
	if len(tools) > 2 {
		shown = append(shown, fmt.Sprintf("+%d", len(tools)-2))
	}
	metadata := map[string]string{"count": fmt.Sprintf("%d", len(tools))}
	if tools[0].Since > 0 {
		now := s.Now
		if now.IsZero() {
			now = time.Now()
		}
		elapsed := now.UnixMilli() - tools[0].Since
		if elapsed >= 0 {
			metadata["elapsed_ms"] = fmt.Sprintf("%d", elapsed)
			shown = append(shown, formatDuration(uint64(elapsed)))
		}
	}
	return &SegmentData{Primary: strings.Join(shown, " · "), Metadata: metadata}
}

type ActivityAgentsSegment struct {
	Bridge *protocol.Bridge
}

func (s ActivityAgentsSegment) Collect() *SegmentData {
	if s.Bridge == nil {
		return nil
	}
	count := 0
	types := make([]string, 0)
	seen := make(map[string]bool)
	for _, agent := range s.Bridge.Agents {
		if !agent.IsActive() {
			continue
		}
		count++
		if agent.Type != "" && !seen[agent.Type] {
			seen[agent.Type] = true
			types = append(types, agent.Type)
		}
	}
	if count == 0 {
		return nil
	}
	primary := fmt.Sprintf("%d running", count)
	if len(types) > 0 {
		primary += " · " + strings.Join(types, " · ")
	}
	return &SegmentData{Primary: primary, Metadata: map[string]string{"count": fmt.Sprintf("%d", count), "types": strings.Join(types, " · ")}}
}

func shortenCategory(name string) string {
	switch name {
	case "System prompt":
		return "sys"
	case "System tools":
		return "tools"
	case "MCP tools":
		return "mcp"
	case "Custom agents":
		return "agents"
	case "Memory files":
		return "memory"
	case "Slash commands":
		return "commands"
	case "Messages":
		return "msgs"
	default:
		fields := strings.Fields(name)
		if len(fields) == 0 {
			return "other"
		}
		return strings.ToLower(fields[0])
	}
}

func formatMetricTokens(tokens uint64) string {
	if tokens < 1000 {
		return fmt.Sprintf("%d", tokens)
	}
	value := float64(tokens) / 1000
	if tokens%1000 == 0 {
		return fmt.Sprintf("%dk", tokens/1000)
	}
	return fmt.Sprintf("%.1fk", value)
}
