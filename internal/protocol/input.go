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
	TotalInputTokens  *uint64             `json:"total_input_tokens"`
	ContextWindowSize *uint32             `json:"context_window_size"`
	UsedPercentage    *float64            `json:"used_percentage"`
	CurrentUsage      *ContextWindowUsage `json:"current_usage"`
}

func (w ContextWindow) ContextTokens() (uint64, bool) {
	if w.TotalInputTokens != nil {
		return *w.TotalInputTokens, true
	}
	if w.CurrentUsage == nil {
		return 0, false
	}
	return w.CurrentUsage.ContextTokens()
}

type ContextWindowUsage struct {
	Number *uint32
	Raw    *RawUsage
}

type Effort struct {
	Level string `json:"level"`
}

type InputData struct {
	SessionID      string         `json:"session_id"`
	Model          Model          `json:"model"`
	Workspace      Workspace      `json:"workspace"`
	TranscriptPath string         `json:"transcript_path"`
	Effort         *Effort        `json:"effort"`
	Cost           *Cost          `json:"cost"`
	ContextWindow  *ContextWindow `json:"context_window"`
	PromptCache    *PromptCache   `json:"prompt_cache"`
}

type PromptCache struct {
	Warm                *bool      `json:"warm"`
	CachingObserved     *bool      `json:"caching_observed"`
	TTL                 *string    `json:"ttl"`
	ExpiresAt           *float64   `json:"expires_at"`
	Requests            *uint64    `json:"requests"`
	Misses              *uint64    `json:"misses"`
	HitRatio            *float64   `json:"hit_ratio"`
	LastMissCause       *MissCause `json:"last_miss_cause"`
	RecacheTokensIfCold *uint64    `json:"recache_tokens_if_cold"`
}

type MissCause struct {
	Causes []string `json:"causes"`
}

type RawUsage struct {
	InputTokens              *uint32 `json:"input_tokens,omitempty"`
	OutputTokens             *uint32 `json:"output_tokens,omitempty"`
	CacheCreationInputTokens *uint32 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     *uint32 `json:"cache_read_input_tokens,omitempty"`
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
		InputTokens:              valueOrZero(u.InputTokens),
		OutputTokens:             valueOrZero(u.OutputTokens),
		CacheCreationInputTokens: valueOrZero(u.CacheCreationInputTokens),
		CacheReadInputTokens:     valueOrZero(u.CacheReadInputTokens),
	}
}

func (u NormalizedUsage) ContextTokens() uint64 {
	return uint64(u.InputTokens) + uint64(u.CacheCreationInputTokens) + uint64(u.CacheReadInputTokens)
}

func (u NormalizedUsage) CachedTokens() uint64 {
	return uint64(u.CacheCreationInputTokens) + uint64(u.CacheReadInputTokens)
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

func (u ContextWindowUsage) ContextTokens() (uint64, bool) {
	if u.Number != nil {
		return uint64(*u.Number), true
	}
	if u.Raw == nil {
		return 0, false
	}
	if u.Raw.InputTokens == nil && u.Raw.CacheCreationInputTokens == nil && u.Raw.CacheReadInputTokens == nil {
		return 0, false
	}

	return u.Raw.Normalize().ContextTokens(), true
}

func valueOrZero(value *uint32) uint32 {
	if value == nil {
		return 0
	}

	return *value
}
