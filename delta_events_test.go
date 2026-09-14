package codex

import (
	"encoding/json"
	"testing"
)

// Wire shapes mirror the official app-server protocol schema
// (protocol/generated_types.go, generated from codex-cli's schema JSON):
// camelCase field names throughout. These decode tests pin the streaming
// delta family — the app-server equivalent of the TS SDK's item.updated
// events, at token/output granularity.

func TestDecodeTurnNotification_AgentMessageDelta(t *testing.T) {
	raw := json.RawMessage(`{"method":"item/agentMessage/delta","params":{"threadId":"t1","turnId":"u1","itemId":"i1","delta":"Hel"}}`)
	ev, terminal, err := decodeTurnNotification("t1", raw, nil)
	if err != nil || terminal {
		t.Fatalf("decode: err=%v terminal=%v", err, terminal)
	}
	if ev.Type != "item.agent_message.delta" || ev.ItemID != "i1" || ev.Delta != "Hel" || ev.TurnID != "u1" {
		t.Fatalf("event mismatch: %+v", ev)
	}
}

func TestDecodeTurnNotification_ReasoningTextDelta(t *testing.T) {
	raw := json.RawMessage(`{"method":"item/reasoning/textDelta","params":{"threadId":"t1","turnId":"u1","itemId":"i2","delta":"think","contentIndex":0}}`)
	ev, terminal, err := decodeTurnNotification("t1", raw, nil)
	if err != nil || terminal {
		t.Fatalf("decode: err=%v terminal=%v", err, terminal)
	}
	if ev.Type != "item.reasoning_text.delta" || ev.Delta != "think" {
		t.Fatalf("event mismatch: %+v", ev)
	}
}

func TestDecodeTurnNotification_ReasoningSummaryDeltaAndPartAdded(t *testing.T) {
	raw := json.RawMessage(`{"method":"item/reasoning/summaryTextDelta","params":{"threadId":"t1","turnId":"u1","itemId":"i3","delta":"sum"}}`)
	ev, _, err := decodeTurnNotification("t1", raw, nil)
	if err != nil || ev.Type != "item.reasoning_summary_text.delta" || ev.Delta != "sum" {
		t.Fatalf("summaryTextDelta mismatch: %+v err=%v", ev, err)
	}
	raw2 := json.RawMessage(`{"method":"item/reasoning/summaryPartAdded","params":{"threadId":"t1","turnId":"u1","itemId":"i3","summaryIndex":2}}`)
	ev2, _, err := decodeTurnNotification("t1", raw2, nil)
	if err != nil || ev2.Type != "item.reasoning_summary_part.added" || ev2.SummaryIndex != 2 {
		t.Fatalf("summaryPartAdded mismatch: %+v err=%v", ev2, err)
	}
}

func TestDecodeTurnNotification_CommandExecutionOutputDelta(t *testing.T) {
	raw := json.RawMessage(`{"method":"item/commandExecution/outputDelta","params":{"threadId":"t1","turnId":"u1","itemId":"i4","delta":"line1\n"}}`)
	ev, terminal, err := decodeTurnNotification("t1", raw, nil)
	if err != nil || terminal {
		t.Fatalf("decode: err=%v terminal=%v", err, terminal)
	}
	if ev.Type != "item.command_execution.output_delta" || ev.Delta != "line1\n" {
		t.Fatalf("event mismatch: %+v", ev)
	}
}

func TestDecodeTurnNotification_FileChangeAndPlanDelta(t *testing.T) {
	raw := json.RawMessage(`{"method":"item/fileChange/outputDelta","params":{"threadId":"t1","turnId":"u1","itemId":"i5","delta":"+x"}}`)
	ev, _, err := decodeTurnNotification("t1", raw, nil)
	if err != nil || ev.Type != "item.file_change.output_delta" || ev.Delta != "+x" {
		t.Fatalf("fileChange mismatch: %+v err=%v", ev, err)
	}
	raw2 := json.RawMessage(`{"method":"item/plan/delta","params":{"threadId":"t1","turnId":"u1","itemId":"i6","delta":"step1"}}`)
	ev2, _, err := decodeTurnNotification("t1", raw2, nil)
	if err != nil || ev2.Type != "item.plan.delta" || ev2.Delta != "step1" {
		t.Fatalf("plan mismatch: %+v err=%v", ev2, err)
	}
}

func TestDecodeTurnNotification_McpToolCallProgress(t *testing.T) {
	raw := json.RawMessage(`{"method":"item/mcpToolCall/progress","params":{"threadId":"t1","turnId":"u1","itemId":"i7","message":"50% done"}}`)
	ev, terminal, err := decodeTurnNotification("t1", raw, nil)
	if err != nil || terminal {
		t.Fatalf("decode: err=%v terminal=%v", err, terminal)
	}
	if ev.Type != "item.mcp_tool_call.progress" || ev.Message != "50% done" || ev.ItemID != "i7" {
		t.Fatalf("event mismatch: %+v", ev)
	}
}

// Delta events must never be terminal (only turn/completed is) and must not
// disturb the latest-usage carry-over semantics.
func TestDecodeTurnNotification_DeltasNonTerminalAndUsageUnaffected(t *testing.T) {
	latest := &Usage{InputTokens: 5}
	for _, m := range []string{
		`item/agentMessage/delta`, `item/reasoning/textDelta`,
		`item/commandExecution/outputDelta`, `item/plan/delta`,
		`item/mcpToolCall/progress`,
	} {
		raw := json.RawMessage(`{"method":"` + m + `","params":{"threadId":"t1","turnId":"u1","itemId":"i","delta":"d","message":"m"}}`)
		ev, terminal, err := decodeTurnNotification("t1", raw, latest)
		if err != nil || terminal {
			t.Fatalf("%s: err=%v terminal=%v", m, err, terminal)
		}
		if ev.Usage != nil {
			t.Fatalf("%s: delta must not carry usage, got %+v", m, ev.Usage)
		}
	}
}
