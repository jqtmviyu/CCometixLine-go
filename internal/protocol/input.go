package protocol

import "encoding/json"

type Model struct {
	ID          string `json:"id"`
	DisplayName string `json:"display_name"`
}

type Workspace struct {
	CurrentDir string `json:"current_dir"`
}

type Cost struct {
	TotalCostUSD       *float64 `json:"total_cost_usd"`
	TotalDurationMS    *uint64  `json:"total_duration_ms"`
	TotalAPIDurationMS *uint64  `json:"total_api_duration_ms"`
	TotalLinesAdded    *uint32  `json:"total_lines_added"`
	TotalLinesRemoved  *uint32  `json:"total_lines_removed"`
}

type ContextWindow struct {
	ContextWindowSize *uint32             `json:"context_window_size"`
	UsedPercentage    *float64            `json:"used_percentage"`
	CurrentUsage      *ContextWindowUsage `json:"current_usage"`
}

type ContextWindowUsage struct {
	Number *uint32
	Raw    *RawUsage
}

type InputData struct {
	Model          Model          `json:"model"`
	Workspace      Workspace      `json:"workspace"`
	TranscriptPath string         `json:"transcript_path"`
	Effort         string         `json:"effort"`
	Cost           *Cost          `json:"cost"`
	ContextWindow  *ContextWindow `json:"context_window"`
}

type PromptTokensDetails struct {
	CachedTokens *uint32 `json:"cached_tokens,omitempty"`
	AudioTokens  *uint32 `json:"audio_tokens,omitempty"`
}

type RawUsage struct {
	InputTokens               *uint32              `json:"input_tokens,omitempty"`
	PromptTokens              *uint32              `json:"prompt_tokens,omitempty"`
	OutputTokens              *uint32              `json:"output_tokens,omitempty"`
	CompletionTokens          *uint32              `json:"completion_tokens,omitempty"`
	TotalTokens               *uint32              `json:"total_tokens,omitempty"`
	CacheCreationInputTokens  *uint32              `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens      *uint32              `json:"cache_read_input_tokens,omitempty"`
	CacheCreationPromptTokens *uint32              `json:"cache_creation_prompt_tokens,omitempty"`
	CacheReadPromptTokens     *uint32              `json:"cache_read_prompt_tokens,omitempty"`
	CachedTokens              *uint32              `json:"cached_tokens,omitempty"`
	PromptTokensDetails       *PromptTokensDetails `json:"prompt_tokens_details,omitempty"`
	CompletionTokensDetails   map[string]uint32    `json:"completion_tokens_details,omitempty"`
}

type NormalizedUsage struct {
	InputTokens              uint32
	OutputTokens             uint32
	CacheCreationInputTokens uint32
	CacheReadInputTokens     uint32
}

type Message struct {
	Role          string          `json:"role"`
	Usage         *RawUsage       `json:"usage"`
	Content       string          `json:"content"`
	StopReason    json.RawMessage `json:"stop_reason"`
	HasStopReason bool            `json:"-"`
}

type TranscriptEntry struct {
	Type              *string  `json:"type"`
	Message           *Message `json:"message"`
	LeafUUID          *string  `json:"leafUuid"`
	UUID              *string  `json:"uuid"`
	ParentUUID        *string  `json:"parentUuid"`
	Summary           *string  `json:"summary"`
	Timestamp         *string  `json:"timestamp"`
	IsSidechain       *bool    `json:"isSidechain"`
	IsAPIErrorMessage *bool    `json:"isApiErrorMessage"`
}

func (u RawUsage) Normalize() NormalizedUsage {
	return NormalizedUsage{
		InputTokens:              firstUint32(u.InputTokens, u.PromptTokens),
		OutputTokens:             firstUint32(u.OutputTokens, u.CompletionTokens),
		CacheCreationInputTokens: firstUint32(u.CacheCreationInputTokens, u.CacheCreationPromptTokens),
		CacheReadInputTokens: firstUint32(
			u.CacheReadInputTokens,
			u.CacheReadPromptTokens,
			u.CachedTokens,
			cachedTokensFromDetails(u.PromptTokensDetails),
		),
	}
}

func (u NormalizedUsage) UsedTokens() uint32 {
	return u.InputTokens + u.CacheCreationInputTokens + u.CacheReadInputTokens + u.OutputTokens
}

func (u NormalizedUsage) CachedTokens() uint32 {
	return u.CacheCreationInputTokens + u.CacheReadInputTokens
}

func (m *Message) UnmarshalJSON(data []byte) error {
	type alias Message
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	var decoded alias
	if err := json.Unmarshal(data, &decoded); err != nil {
		return err
	}

	*m = Message(decoded)
	_, m.HasStopReason = raw["stop_reason"]
	return nil
}

func (u *ContextWindowUsage) UnmarshalJSON(data []byte) error {
	if string(data) == "null" {
		u.Number = nil
		u.Raw = nil
		return nil
	}

	var number uint32
	if err := json.Unmarshal(data, &number); err == nil {
		u.Number = &number
		u.Raw = nil
		return nil
	}

	var raw RawUsage
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	u.Number = nil
	u.Raw = &raw
	return nil
}

func (u ContextWindowUsage) UsedTokens() uint32 {
	if u.Number != nil {
		return *u.Number
	}
	if u.Raw == nil {
		return 0
	}

	return u.Raw.Normalize().UsedTokens()
}

func cachedTokensFromDetails(details *PromptTokensDetails) *uint32 {
	if details == nil {
		return nil
	}

	return details.CachedTokens
}

func firstUint32(values ...*uint32) uint32 {
	for _, value := range values {
		if value != nil {
			return *value
		}
	}

	return 0
}
