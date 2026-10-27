package render

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/segments"
)

type StatusLineGenerator struct {
	Style  config.StyleConfig
	Layout config.ResolvedLayout
}

func (g StatusLineGenerator) Generate(items []segments.CollectedSegment) string {
	byID := make(map[config.SegmentID]string, len(items))
	for _, item := range items {
		if text := g.renderSegment(item.Config, item.Data); text != "" {
			byID[item.Config.ID] = text
		}
	}

	var lines []string
	for _, row := range g.Layout.Lines {
		parts := make([]string, 0, len(row.Segments))
		for _, id := range row.Segments {
			if text, ok := byID[id]; ok {
				parts = append(parts, text)
			}
		}
		if len(parts) > 0 {
			lines = append(lines, g.joinParts(parts))
		}
	}

	return strings.Join(lines, "\n")
}

func (g StatusLineGenerator) GenerateTaskLine(items []segments.CollectedSegment, columns int) string {
	parts := make([]string, 0, len(items))
	for _, item := range items {
		if text := g.renderSegment(item.Config, item.Data); text != "" {
			parts = append(parts, text)
		}
	}
	if len(parts) == 0 {
		return ""
	}
	text := g.joinParts(parts)
	if columns > 0 && ansi.StringWidth(text) > columns {
		return ansi.Truncate(text, columns, "…")
	}
	return text
}

func (g StatusLineGenerator) joinParts(parts []string) string {
	return strings.Join(parts, "\x1b[37m"+g.Style.Separator+"\x1b[0m")
}

func (g StatusLineGenerator) renderSegment(cfg config.SegmentConfig, data segments.SegmentData) string {
	icon := cfg.Icon.Emoji
	if dynamicIcon, ok := data.Metadata["dynamic_icon"]; ok {
		icon = dynamicIcon
	}

	if cfg.Colors.Background != nil {
		bgCode := ApplyBackgroundColor(cfg.Colors.Background)
		iconColored := strings.ReplaceAll(ApplyColor(icon, cfg.Colors.Icon), "\x1b[0m", "")
		textStyled := strings.ReplaceAll(ApplyStyle(data.Primary, cfg.Colors.Text, cfg.Styles.TextBold), "\x1b[0m", "")
		content := " " + textStyled + " "
		if icon != "" {
			content = " " + iconColored + " " + textStyled + " "
		}
		if data.Secondary != "" {
			secondary := strings.ReplaceAll(ApplyStyle(data.Secondary, cfg.Colors.Text, cfg.Styles.TextBold), "\x1b[0m", "")
			content += secondary + " "
		}
		return bgCode + content + "\x1b[49m"
	}

	iconColored := ApplyColor(icon, cfg.Colors.Icon)
	textStyled := ApplyStyle(data.Primary, cfg.Colors.Text, cfg.Styles.TextBold)
	segment := textStyled
	if icon != "" {
		segment = iconColored + " " + textStyled
	}
	if data.Secondary != "" {
		segment += " " + ApplyStyle(data.Secondary, cfg.Colors.Text, cfg.Styles.TextBold)
	}
	return segment
}
