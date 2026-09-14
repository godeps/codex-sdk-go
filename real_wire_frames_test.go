package codex

import (
	"encoding/json"
	"testing"
)

// The wire frames below were captured from the REAL codex-cli 0.153.4
// app-server (2026-09-14, wire_probe E2E). They pin the decoding contract to
// observed reality instead of mocks: all notification fields are camelCase.

func TestDecodeTurnNotification_RealItemStartedFrame(t *testing.T) {
	raw := json.RawMessage(`{"method":"item/started","params":{"item":{"type":"userMessage","id":"01a09d61-f9a6-7ee3-981a-4ed6491ca0af","clientId":null,"content":[{"type":"text","text":"hello","text_elements":[]}]},"threadId":"01a09d61-eb63-7d23-a98f-78350368b86e","turnId":"01a09d61-ed77-7301-a095-4be4aba280b0","startedAtMs":1789346838950}}`)

	ev, terminal, err := decodeTurnNotification("thread-fallback", raw, nil)
	if err != nil {
		t.Fatalf("decodeTurnNotification: %v", err)
	}
	if terminal {
		t.Fatal("item/started must not be terminal")
	}
	if ev.Type != "item.started" {
		t.Fatalf("type = %q, want item.started", ev.Type)
	}
	if ev.ThreadID != "01a09d61-eb63-7d23-a98f-78350368b86e" || ev.TurnID != "01a09d61-ed77-7301-a095-4be4aba280b0" {
		t.Fatalf("thread/turn ids not decoded: %+v", ev)
	}
	if ev.Item == nil {
		t.Fatal("item payload missing (previously dropped into the default branch untyped)")
	}
	if ev.Item.ItemType() == "" {
		t.Fatal("item type empty")
	}
}

func TestDecodeTurnNotification_RealErrorFrame(t *testing.T) {
	raw := json.RawMessage(`{"method":"error","params":{"error":{"message":"Your access token could not be refreshed because your refresh token was revoked. Please log out and sign in again.","codexErrorInfo":"unauthorized","additionalDetails":null,"misalignment":null},"willRetry":false,"threadId":"01a09d61-eb63-7d23-a98f-78350368b86e","turnId":"01a09d61-ed77-7301-a095-4be4aba280b0"}}`)

	ev, terminal, err := decodeTurnNotification("thread-fallback", raw, nil)
	if err != nil {
		t.Fatalf("decodeTurnNotification: %v", err)
	}
	if terminal {
		t.Fatal("error notification alone must not be terminal (turn/completed carries the failure)")
	}
	if ev.Type != "error" {
		t.Fatalf("type = %q, want error", ev.Type)
	}
	if ev.Error == nil {
		t.Fatal("error payload missing")
	}
	if ev.Error.CodexErrorInfo != "unauthorized" {
		t.Fatalf("codexErrorInfo = %q, want unauthorized", ev.Error.CodexErrorInfo)
	}
}

func TestUsage_Unmarshal_RealCamelCaseWire(t *testing.T) {
	// The real app-server emits camelCase token usage (protocol schema:
	// TokenUsageBreakdown). Before the fix, Usage only carried snake_case
	// tags and silently decoded to zero.
	var camel Usage
	if err := json.Unmarshal([]byte(`{"inputTokens":11,"cachedInputTokens":2,"cacheWriteInputTokens":3,"outputTokens":7,"reasoningOutputTokens":5,"totalTokens":18}`), &camel); err != nil {
		t.Fatalf("camelCase unmarshal: %v", err)
	}
	if camel.InputTokens != 11 || camel.CachedInputTokens != 2 || camel.CacheWriteInputTokens != 3 ||
		camel.OutputTokens != 7 || camel.ReasoningOutputTokens != 5 || camel.TotalTokens != 18 {
		t.Fatalf("camelCase decode wrong: %+v", camel)
	}

	var snake Usage
	if err := json.Unmarshal([]byte(`{"input_tokens":11,"cached_input_tokens":2,"cache_write_input_tokens":3,"output_tokens":7,"reasoning_output_tokens":5,"total_tokens":18}`), &snake); err != nil {
		t.Fatalf("snake_case unmarshal: %v", err)
	}
	if snake != camel {
		t.Fatalf("snake/camel parity broken: %+v vs %+v", snake, camel)
	}
}

func TestDecodeTurnNotification_RealTokenUsageCamelCase(t *testing.T) {
	// Same notification shape as the real app-server, usage in camelCase.
	raw := json.RawMessage(`{"method":"thread/tokenUsage/updated","params":{"threadId":"t1","turnId":"u1","tokenUsage":{"last":{"inputTokens":42,"cachedInputTokens":4,"outputTokens":9,"reasoningOutputTokens":2,"totalTokens":51}}}}`)

	ev, _, err := decodeTurnNotification("t1", raw, nil)
	if err != nil {
		t.Fatalf("decodeTurnNotification: %v", err)
	}
	if ev.Usage == nil {
		t.Fatal("usage missing")
	}
	if ev.Usage.InputTokens != 42 || ev.Usage.OutputTokens != 9 || ev.Usage.TotalTokens != 51 {
		t.Fatalf("usage decoded wrong (camelCase bug?): %+v", ev.Usage)
	}
}
