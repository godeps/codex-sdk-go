package codex

import (
	"encoding/json"
	"testing"
)

func TestThreadEventUnmarshalAgentMessage(t *testing.T) {
	raw := `{"type":"item.completed","item":{"id":"1","type":"agent_message","text":"hello"}}`
	var event ThreadEvent
	if err := json.Unmarshal([]byte(raw), &event); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if event.Type != "item.completed" {
		t.Fatalf("unexpected type: %s", event.Type)
	}
	msg, ok := event.Item.(*AgentMessageItem)
	if !ok {
		t.Fatalf("expected agent_message item, got %T", event.Item)
	}
	if msg.Text != "hello" {
		t.Fatalf("unexpected message: %s", msg.Text)
	}
}

func TestThreadEventUnmarshalUnknownItem(t *testing.T) {
	raw := `{"type":"item.completed","item":{"id":"1","type":"unknown"}}`
	var event ThreadEvent
	if err := json.Unmarshal([]byte(raw), &event); err == nil {
		t.Fatalf("expected error for unknown item type")
	}
}
