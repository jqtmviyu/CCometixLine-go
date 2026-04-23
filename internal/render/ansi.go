package render

import (
	"fmt"
	"strings"

	"ccometixline-go/internal/config"
)

// colorCode 返回颜色的 SGR 参数部分（不含 \x1b[ 和 m）
// base: 前景 30, 背景 40
func colorCode(color *config.AnsiColor, base int) string {
	if color == nil {
		return ""
	}

	if color.C16 != nil {
		code := base + int(*color.C16)
		if *color.C16 >= 8 {
			code = base + 60 + int(*color.C16-8)
		}
		return fmt.Sprintf("%d", code)
	}

	if color.C256 != nil {
		return fmt.Sprintf("%d;5;%d", base+8, *color.C256)
	}

	if color.R != nil && color.G != nil && color.B != nil {
		return fmt.Sprintf("%d;2;%d;%d;%d", base+8, *color.R, *color.G, *color.B)
	}

	return ""
}

func ApplyColor(text string, color *config.AnsiColor) string {
	code := colorCode(color, 30)
	if code == "" {
		return text
	}
	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", code, text)
}

func ApplyStyle(text string, color *config.AnsiColor, bold bool) string {
	codes := []string{}
	if bold {
		codes = append(codes, "1")
	}

	if code := colorCode(color, 30); code != "" {
		codes = append(codes, code)
	}

	if len(codes) == 0 {
		return text
	}

	return fmt.Sprintf("\x1b[%sm%s\x1b[0m", strings.Join(codes, ";"), text)
}

func ApplyBackgroundColor(color *config.AnsiColor) string {
	code := colorCode(color, 40)
	if code == "" {
		return ""
	}
	return fmt.Sprintf("\x1b[%sm", code)
}

func ColorToForegroundCode(color *config.AnsiColor) string {
	code := colorCode(color, 30)
	if code == "" {
		return ""
	}
	return fmt.Sprintf("\x1b[%sm", code)
}
