package protocol

import "testing"

func TestDecodeSubagentInput(t *testing.T) {
	payload := []byte(`{"session_id":"s1","columns":72,"tasks":[{"id":"a1","name":"search","type":"local_agent","status":"running","label":"Reading files","startTime":1700000000000,"tokenSamples":[100,250]},{"id":"b1","type":"local_bash","status":"completed"}]}`)
	input, ok, err := DecodeSubagentInput(payload)
	if err != nil {
		t.Fatal(err)
	}
	if !ok || input.SessionID != "s1" || input.Columns != 72 || len(input.Tasks) != 2 {
		t.Fatalf("unexpected task input: %#v, detected=%v", input, ok)
	}
	if input.Tasks[0].AgentType != "" || input.Tasks[0].Kind != "local_agent" || input.Tasks[0].Label != "Reading files" {
		t.Fatalf("unexpected task fields: %#v", input.Tasks[0])
	}
	if len(input.Tasks[0].TokenSamples) != 2 || input.Tasks[0].TokenSamples[1] != 250 {
		t.Fatalf("unexpected token samples: %#v", input.Tasks[0].TokenSamples)
	}
}

func TestDecodeSubagentInputLeavesStatuslinePayloadAlone(t *testing.T) {
	input, ok, err := DecodeSubagentInput([]byte(`{"model":{"id":"claude-sonnet"}}`))
	if err != nil || ok || input != nil {
		t.Fatalf("unexpected task detection: %#v, %v, %v", input, ok, err)
	}
}
