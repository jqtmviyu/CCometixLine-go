package render

import (
	"strings"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/segments"
)

type StatusLineGenerator struct {
	Config config.Config
}

type renderedItem struct {
	text   string
	config config.SegmentConfig
}

func (g StatusLineGenerator) Generate(items []segments.CollectedSegment) string {
	var parts []renderedItem
	for _, item := range items {
		text := g.renderSegment(item.Config, item.Data)
		if text != "" {
			parts = append(parts, renderedItem{text, item.Config})
		}
	}

	if len(parts) == 0 {
		return ""
	}

	if g.Config.Style.Separator == "\uE0B0" {
		return g.joinWithPowerlineArrows(parts)
	}

	texts := make([]string, len(parts))
	for i, p := range parts {
		texts[i] = p.text
	}
	return g.joinWithWhiteSeparators(texts)
}

func (g StatusLineGenerator) renderSegment(cfg config.SegmentConfig, data segments.SegmentData) string {
	icon := g.getIcon(cfg)
	if dynamicIcon, ok := data.Metadata["dynamic_icon"]; ok {
		icon = dynamicIcon
	}

	if cfg.Colors.Background != nil {
		bgCode := ApplyBackgroundColor(cfg.Colors.Background)
		iconColored := strings.ReplaceAll(ApplyColor(icon, cfg.Colors.Icon), "\x1b[0m", "")
		textStyled := strings.ReplaceAll(ApplyStyle(data.Primary, cfg.Colors.Text, cfg.Styles.TextBold), "\x1b[0m", "")
		content := " " + iconColored + " " + textStyled + " "
		if data.Secondary != "" {
			secondary := strings.ReplaceAll(ApplyStyle(data.Secondary, cfg.Colors.Text, cfg.Styles.TextBold), "\x1b[0m", "")
			content += secondary + " "
		}
		return bgCode + content + "\x1b[49m"
	}

	iconColored := ApplyColor(icon, cfg.Colors.Icon)
	textStyled := ApplyStyle(data.Primary, cfg.Colors.Text, cfg.Styles.TextBold)
	segment := iconColored + " " + textStyled
	if data.Secondary != "" {
		segment += " " + ApplyStyle(data.Secondary, cfg.Colors.Text, cfg.Styles.TextBold)
	}
	return segment
}

func (g StatusLineGenerator) getIcon(cfg config.SegmentConfig) string {
	switch g.Config.Style.Mode {
	case config.StyleModePlain:
		return cfg.Icon.Plain
	case config.StyleModeNerdFont, config.StyleModePowerline:
		return cfg.Icon.NerdFont
	default:
		return cfg.Icon.Plain
	}
}

func (g StatusLineGenerator) joinWithWhiteSeparators(rendered []string) string {
	separator := "\x1b[37m" + g.Config.Style.Separator + "\x1b[0m"
	return strings.Join(rendered, separator)
}

func (g StatusLineGenerator) joinWithPowerlineArrows(parts []renderedItem) string {
	if len(parts) == 1 {
		return parts[0].text
	}

	result := parts[0].text
	for i := 1; i < len(parts); i++ {
		prevBg := parts[i-1].config.Colors.Background
		currBg := parts[i].config.Colors.Background
		result += g.createPowerlineArrow(prevBg, currBg)
		result += parts[i].text
	}
	result += "\x1b[0m"
	return result
}

func (g StatusLineGenerator) createPowerlineArrow(prevBg *config.AnsiColor, currBg *config.AnsiColor) string {
	arrow := "\uE0B0"
	switch {
	case prevBg != nil && currBg != nil:
		return ApplyBackgroundColor(currBg) + ColorToForegroundCode(prevBg) + arrow + "\x1b[0m"
	case prevBg != nil:
		return ColorToForegroundCode(prevBg) + arrow + "\x1b[0m"
	case currBg != nil:
		return ApplyBackgroundColor(currBg) + arrow + "\x1b[0m"
	default:
		return arrow
	}
}
