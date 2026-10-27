package render

import (
	"strings"

	"github.com/charmbracelet/x/ansi"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/segments"
)

type StatusLineGenerator struct {
	Config config.Config
}

type renderedItem struct {
	id     config.SegmentID
	text   string
	config config.SegmentConfig
}

func (g StatusLineGenerator) Generate(items []segments.CollectedSegment) string {
	var modelConfig *config.SegmentConfig
	for i := range items {
		if items[i].Config.ID == config.SegmentModel {
			modelConfig = &items[i].Config
		}
	}
	byID := make(map[config.SegmentID]renderedItem, len(items))
	order := make([]config.SegmentID, 0, len(items))
	for _, item := range items {
		// 模型和思考力度使用同一套样式，修改模型配色时一起生效。
		if item.Config.ID == config.SegmentEffort && modelConfig != nil {
			item.Config.Colors = modelConfig.Colors
			item.Config.Styles = modelConfig.Styles
		}
		text := g.renderSegment(item.Config, item.Data)
		if text == "" {
			continue
		}
		if _, exists := byID[item.Config.ID]; !exists {
			order = append(order, item.Config.ID)
		}
		byID[item.Config.ID] = renderedItem{id: item.Config.ID, text: text, config: item.Config}
	}

	layout := g.Config.Layout.Lines
	if len(layout) == 0 {
		layout = []config.LayoutLine{{Segments: order}}
	}

	var lines []string
	for _, row := range layout {
		parts := make([]renderedItem, 0, len(row.Segments))
		for _, id := range row.Segments {
			if item, ok := byID[id]; ok {
				parts = append(parts, item)
			}
		}
		if len(parts) > 0 {
			lines = append(lines, g.joinParts(parts))
		}
	}

	return strings.Join(lines, "\n")
}

func (g StatusLineGenerator) GenerateTaskLine(items []segments.CollectedSegment, columns int) string {
	parts := make([]renderedItem, 0, len(items))
	for _, item := range items {
		text := g.renderSegment(item.Config, item.Data)
		if text != "" {
			parts = append(parts, renderedItem{id: item.Config.ID, text: text, config: item.Config})
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

func (g StatusLineGenerator) joinParts(parts []renderedItem) string {
	return g.joinWithWhiteSeparators(parts)
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

func isModelEffortPair(left config.SegmentID, right config.SegmentID) bool {
	return (left == config.SegmentModel && right == config.SegmentEffort) ||
		(left == config.SegmentEffort && right == config.SegmentModel)
}

func (g StatusLineGenerator) joinWithWhiteSeparators(parts []renderedItem) string {
	var result strings.Builder
	for i, part := range parts {
		if i > 0 {
			separator := g.Config.Style.Separator
			if isModelEffortPair(parts[i-1].id, part.id) {
				model := part.config
				if parts[i-1].id == config.SegmentModel {
					model = parts[i-1].config
				}
				result.WriteString(ApplyStyle(" · ", model.Colors.Text, model.Styles.TextBold))
			} else {
				result.WriteString("\x1b[37m" + separator + "\x1b[0m")
			}
		}
		result.WriteString(part.text)
	}
	return result.String()
}
