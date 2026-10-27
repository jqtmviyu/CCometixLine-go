package protocol

import (
	"bytes"
	"encoding/json"
)

type SubagentInput struct {
	SessionID string
	CWD       string
	Columns   int
	Tasks     []Task
}

type Task struct {
	ID                string    `json:"id"`
	Name              string    `json:"name"`
	Kind              string    `json:"type"`
	AgentType         string    `json:"agentType"`
	Status            string    `json:"status"`
	Description       string    `json:"description"`
	Label             string    `json:"label"`
	StartTime         float64   `json:"startTime"`
	Model             string    `json:"model"`
	ContextWindowSize uint64    `json:"contextWindowSize"`
	TokenCount        uint64    `json:"tokenCount"`
	TokenSamples      []float64 `json:"tokenSamples"`
	CWD               string    `json:"cwd"`
}

func DecodeSubagentInput(data []byte) (*SubagentInput, bool, error) {
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(data, &probe); err != nil {
		return nil, false, err
	}
	tasksData, ok := probe["tasks"]
	if !ok {
		return nil, false, nil
	}
	tasksData = bytes.TrimSpace(tasksData)
	if len(tasksData) == 0 || tasksData[0] != '[' {
		return nil, false, nil
	}

	var raw struct {
		SessionID string            `json:"session_id"`
		CWD       string            `json:"cwd"`
		Columns   int               `json:"columns"`
		Tasks     []json.RawMessage `json:"tasks"`
	}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, false, err
	}

	input := &SubagentInput{
		SessionID: raw.SessionID,
		CWD:       raw.CWD,
		Columns:   raw.Columns,
		Tasks:     make([]Task, 0, len(raw.Tasks)),
	}
	for _, rawTask := range raw.Tasks {
		var task Task
		if err := json.Unmarshal(rawTask, &task); err != nil || task.ID == "" {
			continue
		}
		input.Tasks = append(input.Tasks, task)
	}
	return input, true, nil
}
