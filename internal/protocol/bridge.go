package protocol

type Bridge struct {
	Version   int            `json:"v"`
	Session   string         `json:"session"`
	UpdatedAt int64          `json:"updated_at"`
	Context   *BridgeContext `json:"context"`
	Turn      *BridgeTurn    `json:"turn"`
	Tools     []BridgeTool   `json:"tools"`
	Agents    []BridgeAgent  `json:"agents"`
}

type BridgeContext struct {
	Threshold  *uint64          `json:"threshold"`
	Auto       bool             `json:"auto"`
	Window     *uint64          `json:"window"`
	Used       *uint64          `json:"used"`
	Categories []BridgeCategory `json:"categories"`
}

type BridgeCategory struct {
	Name   string `json:"name"`
	Kind   string `json:"kind"`
	Tokens uint64 `json:"tokens"`
}

type BridgeTurn struct {
	Model        string   `json:"model"`
	TTFTMS       *uint64  `json:"ttft_ms"`
	TPS          *float64 `json:"tps"`
	DurationMS   *uint64  `json:"duration_ms"`
	OutputTokens *uint64  `json:"output_tokens"`
	EndedAt      *int64   `json:"ended_at"`
}

type BridgeTool struct {
	Name  string `json:"name"`
	Since int64  `json:"since"`
	Agent string `json:"agent"`
}

type BridgeAgent struct {
	ID          string `json:"id"`
	Type        string `json:"type"`
	Status      string `json:"status"`
	Description string `json:"description"`
}

func (a BridgeAgent) IsActive() bool {
	switch a.Status {
	case "pending", "running", "waiting":
		return true
	default:
		return false
	}
}
