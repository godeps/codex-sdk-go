package codex

import (
	"encoding/json"
	"testing"
)

// TS SDK parity pins (@openai/codex-sdk 0.154.0).

func TestConfigOverrides_AppendedAfterStructuredConfig(t *testing.T) {
	// flattenConfigOverrides handles the structured Config map; the client
	// appends raw ConfigOverrides after it (TS ordering: config →
	// configOverrides → SDK-managed).
	flags, err := flattenConfigOverrides(map[string]any{"model": "gpt-5"})
	if err != nil {
		t.Fatalf("flattenConfigOverrides: %v", err)
	}
	opts := CodexOptions{ConfigOverrides: []string{`model_reasoning_effort="high"`}}
	flags = append(flags, opts.ConfigOverrides...)
	if len(flags) != 2 {
		t.Fatalf("flags = %v, want 2 entries", flags)
	}
	if flags[1] != `model_reasoning_effort="high"` {
		t.Fatalf("raw override not passed unchanged: %q", flags[1])
	}
}

func TestReasoningEffortConstants_TSParity(t *testing.T) {
	want := map[string]ModelReasoningEffort{
		"minimal": ReasoningMinimal, "low": ReasoningLow, "medium": ReasoningMedium,
		"high": ReasoningHigh, "xhigh": ReasoningXHigh,
		"max": ReasoningMax, "ultra": ReasoningUltra, "persistent": ReasoningPersistent,
	}
	for wire, got := range want {
		if string(got) != wire {
			t.Fatalf("constant %q has wire value %q", wire, string(got))
		}
	}
}

func TestCommandExecutionItem_Unmarshal_RealAppServerCasing(t *testing.T) {
	// Real app-server casing: aggregatedOutput / exitCode / inProgress.
	var camel CommandExecutionItem
	if err := json.Unmarshal([]byte(`{"id":"c1","type":"commandExecution","command":"ls","aggregatedOutput":"file1\nfile2","exitCode":0,"status":"inProgress"}`), &camel); err != nil {
		t.Fatalf("camelCase unmarshal: %v", err)
	}
	if camel.AggregatedOutput != "file1\nfile2" {
		t.Fatalf("aggregatedOutput lost: %q", camel.AggregatedOutput)
	}
	if camel.ExitCode == nil || *camel.ExitCode != 0 {
		t.Fatalf("exitCode lost: %v", camel.ExitCode)
	}
	if !camel.Status.IsInProgress() {
		t.Fatalf("inProgress not recognized: %q", camel.Status)
	}

	var snake CommandExecutionItem
	if err := json.Unmarshal([]byte(`{"id":"c1","type":"command_execution","command":"ls","aggregated_output":"out","exit_code":2,"status":"in_progress"}`), &snake); err != nil {
		t.Fatalf("snake_case unmarshal: %v", err)
	}
	if snake.AggregatedOutput != "out" || snake.ExitCode == nil || *snake.ExitCode != 2 || !snake.Status.IsInProgress() {
		t.Fatalf("snake_case decode wrong: %+v", snake)
	}
}

func TestMcpToolCallResult_Unmarshal_RealAppServerCasing(t *testing.T) {
	var r McpToolCallResult
	if err := json.Unmarshal([]byte(`{"content":[{"type":"text","text":"ok"}],"structuredContent":{"a":1}}`), &r); err != nil {
		t.Fatalf("camelCase unmarshal: %v", err)
	}
	if r.StructuredContent == nil {
		t.Fatal("structuredContent lost (real app-server casing)")
	}
	if len(r.Content) != 1 {
		t.Fatalf("content lost: %+v", r.Content)
	}

	var r2 McpToolCallResult
	if err := json.Unmarshal([]byte(`{"structured_content":{"b":2}}`), &r2); err != nil {
		t.Fatalf("snake_case unmarshal: %v", err)
	}
	if r2.StructuredContent == nil {
		t.Fatal("structured_content lost (exec JSONL casing)")
	}
}

func TestUserMessageItem_RealFrame(t *testing.T) {
	// Captured from real codex-cli 0.153.4 item/started frame.
	item, err := parseThreadItem(json.RawMessage(`{"type":"userMessage","id":"01a09d61","clientId":null,"content":[{"type":"text","text":"hello","text_elements":[]}]}`))
	if err != nil {
		t.Fatalf("parseThreadItem: %v", err)
	}
	um, ok := item.(*UserMessageItem)
	if !ok {
		t.Fatalf("item type = %T, want *UserMessageItem", item)
	}
	if len(um.Content) != 1 || um.Content[0].Text != "hello" {
		t.Fatalf("content not decoded: %+v", um.Content)
	}
}

func TestThreadError_CodexErrorInfo(t *testing.T) {
	var e ThreadError
	if err := json.Unmarshal([]byte(`{"message":"boom","codexErrorInfo":"unauthorized","additionalDetails":null,"misalignment":null}`), &e); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if e.CodexErrorInfo != "unauthorized" {
		t.Fatalf("codexErrorInfo = %q", e.CodexErrorInfo)
	}
}
