package segments

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"

	"ccometixline-go/internal/config"
	"ccometixline-go/internal/protocol"
)

type ContextWindowSegment struct {
	Input       protocol.InputData
	ModelConfig config.ModelConfig
}

func (s ContextWindowSegment) Collect() *SegmentData {
	contextLimit := s.ModelConfig.GetContextLimit(s.Input.Model.ID)

	used, rate, ok := parsePayloadContextWindow(s.Input, contextLimit)
	if !ok {
		used, rate, ok = scanTranscriptForLastUsage(s.Input.TranscriptPath, contextLimit)
	}

	if !ok {
		return &SegmentData{
			Primary: "- · - tokens",
			Metadata: map[string]string{
				"limit":      fmt.Sprintf("%d", contextLimit),
				"model":      s.Input.Model.ID,
				"tokens":     "-",
				"percentage": "-",
			},
		}
	}

	percentageDisplay := fmt.Sprintf("%.1f%%", rate)
	if rate == float64(uint64(rate)) {
		percentageDisplay = fmt.Sprintf("%.0f%%", rate)
	}

	tokensDisplay := fmt.Sprintf("%d", used)
	if used >= 1000 {
		k := float64(used) / 1000.0
		if k == float64(uint64(k)) {
			tokensDisplay = fmt.Sprintf("%dk", int(k))
		} else {
			tokensDisplay = fmt.Sprintf("%.1fk", k)
		}
	}

	return &SegmentData{
		Primary: percentageDisplay + " · " + tokensDisplay + " tokens",
		Metadata: map[string]string{
			"limit":      fmt.Sprintf("%d", contextLimit),
			"model":      s.Input.Model.ID,
			"tokens":     fmt.Sprintf("%d", used),
			"percentage": fmt.Sprintf("%v", rate),
		},
	}
}

func parsePayloadContextWindow(input protocol.InputData, contextLimit uint32) (uint32, float64, bool) {
	if input.ContextWindow == nil {
		return 0, 0, false
	}

	used := uint32(0)
	if input.ContextWindow.CurrentUsage != nil {
		used = input.ContextWindow.CurrentUsage.UsedTokens()
	}
	if used == 0 {
		return 0, 0, false
	}

	if input.ContextWindow.UsedPercentage != nil {
		return used, *input.ContextWindow.UsedPercentage, true
	}
	if input.ContextWindow.ContextWindowSize != nil && *input.ContextWindow.ContextWindowSize > 0 {
		limit := *input.ContextWindow.ContextWindowSize
		return used, (float64(used) / float64(limit)) * 100.0, true
	}
	if contextLimit > 0 {
		return used, (float64(used) / float64(contextLimit)) * 100.0, true
	}

	return used, 0, true
}

// scanTranscriptForLastUsage 从 transcript jsonl 末尾往前找最近一个带 usage 的 assistant 行
func scanTranscriptForLastUsage(path string, contextLimit uint32) (uint32, float64, bool) {
	if path == "" {
		return 0, 0, false
	}

	lines, err := readLinesReversed(path)
	if err != nil {
		return 0, 0, false
	}

	for _, line := range lines {
		if len(line) == 0 {
			continue
		}
		var entry protocol.TranscriptEntry
		if err := json.Unmarshal(line, &entry); err != nil {
			continue
		}
		if !isAssistantEntry(entry) {
			continue
		}
		if entry.Message == nil || entry.Message.Usage == nil {
			continue
		}
		normalized := entry.Message.Usage.Normalize()
		used := normalized.UsedTokens()
		if used == 0 {
			continue
		}
		if contextLimit > 0 {
			return used, (float64(used) / float64(contextLimit)) * 100.0, true
		}
		return used, 0, true
	}

	return 0, 0, false
}

func isAssistantEntry(entry protocol.TranscriptEntry) bool {
	if entry.Type != nil && *entry.Type == "assistant" {
		return true
	}
	if entry.Message != nil && entry.Message.Role == "assistant" {
		return true
	}
	return false
}

// readLinesReversed 读取文件所有行，返回倒序切片（末行在前）
func readLinesReversed(path string) ([][]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	const maxLineBytes = 1024 * 1024 // 1MB，兼容超长行
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, maxLineBytes), maxLineBytes)

	var lines [][]byte
	for scanner.Scan() {
		b := scanner.Bytes()
		cp := make([]byte, len(b))
		copy(cp, b)
		lines = append(lines, cp)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	// 原地倒序
	for i, j := 0, len(lines)-1; i < j; i, j = i+1, j-1 {
		lines[i], lines[j] = lines[j], lines[i]
	}
	return lines, nil
}
