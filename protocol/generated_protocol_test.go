package protocol

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestThreadStartParamsJSONAliasesAndOptionality(t *testing.T) {
	base := "system"
	source := ThreadSource("user")
	params := ThreadStartParams{
		BaseInstructions: &base,
		ThreadSource:     &source,
	}
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	got := string(data)
	if got != `{"baseInstructions":"system","threadSource":"user"}` {
		t.Fatalf("marshal = %s", got)
	}
}

func TestFutureEnumStringsRemainTyped(t *testing.T) {
	var plan PlanType
	if err := json.Unmarshal([]byte(`"future_plan"`), &plan); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if plan != PlanType("future_plan") {
		t.Fatalf("plan = %q", plan)
	}
}

func TestUnknownUnionStringFallsBackToRaw(t *testing.T) {
	var mode AuthMode
	if err := json.Unmarshal([]byte(`"future_auth_mode"`), &mode); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := mode.Value.(*UnknownUnionValue); !ok {
		t.Fatalf("mode.Value = %T, want *UnknownUnionValue", mode.Value)
	}
}

func TestMalformedKnownNotificationFallsBackToRaw(t *testing.T) {
	var notification ServerNotification
	raw := []byte(`{"method":"thread/started"}`)
	if err := json.Unmarshal(raw, &notification); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := notification.Value.(*UnknownUnionValue); !ok {
		t.Fatalf("notification.Value = %T, want *UnknownUnionValue", notification.Value)
	}
	encoded, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestUnknownDiscriminatorNotificationFallsBackToRawRoundTrip(t *testing.T) {
	var notification ServerNotification
	raw := []byte(`{"method":"future/notification","params":{"ok":true}}`)
	if err := json.Unmarshal(raw, &notification); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := notification.Value.(*UnknownUnionValue); !ok {
		t.Fatalf("notification.Value = %T, want *UnknownUnionValue", notification.Value)
	}
	encoded, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestUnknownUserInputVariantFallsBackToRaw(t *testing.T) {
	var input UserInput
	raw := []byte(`{"type":"futureInput","foo":"bar"}`)
	if err := json.Unmarshal(raw, &input); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := input.Value.(*UnknownUnionValue); !ok {
		t.Fatalf("input.Value = %T, want *UnknownUnionValue", input.Value)
	}
}

func TestUnknownDiscriminatorThreadItemFallsBackToRawRoundTrip(t *testing.T) {
	var item ThreadItem
	raw := []byte(`{"type":"futureItem","id":"1","foo":"bar"}`)
	if err := json.Unmarshal(raw, &item); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := item.Value.(*UnknownUnionValue); !ok {
		t.Fatalf("item.Value = %T, want *UnknownUnionValue", item.Value)
	}
	encoded, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestAdditionalPropertiesRoundTrip(t *testing.T) {
	raw := []byte(`{"enabled":true,"customFlag":"beta","nested":{"ok":1}}`)
	var analytics AnalyticsConfig
	if err := json.Unmarshal(raw, &analytics); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if analytics.Enabled == nil || !*analytics.Enabled {
		t.Fatalf("analytics.Enabled = %#v", analytics.Enabled)
	}
	if string(analytics.Extras["customFlag"]) != `"beta"` {
		t.Fatalf("analytics.Extras = %#v", analytics.Extras)
	}
	encoded, err := json.Marshal(analytics)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestConfigAdditionalPropertiesPreserveFixedFields(t *testing.T) {
	raw := []byte(`{"model":"gpt-5","unknownKey":{"value":1},"analytics":{"enabled":true,"x":"y"}}`)
	var cfg Config
	if err := json.Unmarshal(raw, &cfg); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if cfg.Model == nil || *cfg.Model != "gpt-5" {
		t.Fatalf("cfg.Model = %#v", cfg.Model)
	}
	if cfg.Extras == nil || cfg.Extras["unknownKey"] == nil {
		t.Fatalf("cfg.Extras = %#v", cfg.Extras)
	}
	if cfg.Analytics == nil || cfg.Analytics.Extras["x"] == nil {
		t.Fatalf("cfg.Analytics = %#v", cfg.Analytics)
	}
	encoded, err := json.Marshal(cfg)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestAdditionalPropertiesMarshalPrefersFixedFields(t *testing.T) {
	enabled := true
	analytics := AnalyticsConfig{
		Enabled: &enabled,
		Extras: map[string]json.RawMessage{
			"enabled": []byte(`false`),
			"custom":  []byte(`null`),
		},
	}
	encoded, err := json.Marshal(analytics)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if decoded["enabled"] != true {
		t.Fatalf("decoded enabled = %#v, want true", decoded["enabled"])
	}
	if _, ok := decoded["custom"]; !ok {
		t.Fatalf("decoded missing custom key: %#v", decoded)
	}
}

func TestPrimitiveUnionFallbackRoundTrip(t *testing.T) {
	for _, raw := range [][]byte{
		[]byte(`"request-id"`),
		[]byte(`12345`),
		[]byte(`"all"`),
		[]byte(`["/tmp","/srv"]`),
	} {
		switch raw[0] {
		case '"', '1':
			var id RequestId
			if err := json.Unmarshal(raw, &id); err != nil {
				t.Fatalf("RequestId json.Unmarshal(%s) error = %v", raw, err)
			}
			encoded, err := json.Marshal(id)
			if err != nil {
				t.Fatalf("RequestId json.Marshal(%s) error = %v", raw, err)
			}
			assertJSONEqual(t, raw, encoded)
		default:
			var filter ThreadListCwdFilter
			if err := json.Unmarshal(raw, &filter); err != nil {
				t.Fatalf("ThreadListCwdFilter json.Unmarshal(%s) error = %v", raw, err)
			}
			encoded, err := json.Marshal(filter)
			if err != nil {
				t.Fatalf("ThreadListCwdFilter json.Marshal(%s) error = %v", raw, err)
			}
			assertJSONEqual(t, raw, encoded)
		}
	}
}

func TestRequiredNullableFieldRoundTripAndInvalid(t *testing.T) {
	rawNull := []byte(`{"sectionId":null,"threadId":"thread-1"}`)
	var nullParams ThreadSectionMoveParams
	if err := json.Unmarshal(rawNull, &nullParams); err != nil {
		t.Fatalf("json.Unmarshal(null) error = %v", err)
	}
	if nullParams.SectionID != nil {
		t.Fatalf("SectionID = %#v, want nil", nullParams.SectionID)
	}
	encodedNull, err := json.Marshal(nullParams)
	if err != nil {
		t.Fatalf("json.Marshal(null) error = %v", err)
	}
	assertJSONEqual(t, rawNull, encodedNull)

	sectionID := "section-1"
	params := ThreadSectionMoveParams{
		SectionID: &sectionID,
		ThreadID:  "thread-1",
	}
	encoded, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(non-null) error = %v", err)
	}
	assertJSONEqual(t, []byte(`{"sectionId":"section-1","threadId":"thread-1"}`), encoded)

	var invalid ThreadSectionMoveParams
	if err := json.Unmarshal([]byte(`{"threadId":"thread-1"}`), &invalid); err == nil {
		t.Fatal("json.Unmarshal(missing required-nullable field) error = nil, want failure")
	}
}

func TestNestedUnionKnownVariantRoundTrip(t *testing.T) {
	raw := []byte(`{"completedAtMs":1,"threadId":"thread-1","turnId":"turn-1","item":{"type":"agentMessage","id":"item-1","text":"hello"}}`)
	var notification ItemCompletedNotification
	if err := json.Unmarshal(raw, &notification); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	msg, ok := notification.Item.Value.(*AgentMessageThreadItem)
	if !ok {
		t.Fatalf("notification.Item.Value = %T, want *AgentMessageThreadItem", notification.Item.Value)
	}
	if msg.Text != "hello" {
		t.Fatalf("msg.Text = %q, want hello", msg.Text)
	}
	encoded, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestNestedUnionUnknownVariantFallsBackToRaw(t *testing.T) {
	raw := []byte(`{"completedAtMs":1,"threadId":"thread-1","turnId":"turn-1","item":{"type":"futureItem","id":"1","foo":"bar"}}`)
	var notification ItemCompletedNotification
	if err := json.Unmarshal(raw, &notification); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := notification.Item.Value.(*UnknownUnionValue); !ok {
		t.Fatalf("notification.Item.Value = %T, want *UnknownUnionValue", notification.Item.Value)
	}
	encoded, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertJSONEqual(t, raw, encoded)
}

func TestNestedUnionMalformedKnownVariantFallsBackToRaw(t *testing.T) {
	raw := []byte(`{"completedAtMs":1,"threadId":"thread-1","turnId":"turn-1","item":{"type":"agentMessage","id":"item-1","foo":"bar"}}`)
	var notification ItemCompletedNotification
	if err := json.Unmarshal(raw, &notification); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	if _, ok := notification.Item.Value.(*UnknownUnionValue); !ok {
		t.Fatalf("notification.Item.Value = %T, want *UnknownUnionValue", notification.Item.Value)
	}
	encoded, err := json.Marshal(notification)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}
	assertJSONEqual(t, raw, encoded)
}

func FuzzThreadItemUnmarshal(f *testing.F) {
	f.Add([]byte(`{"type":"agentMessage","id":"1","text":"hello"}`))
	f.Add([]byte(`{"type":"future","id":"1"}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var item ThreadItem
		_ = json.Unmarshal(raw, &item)
	})
}

func FuzzServerNotificationUnmarshal(f *testing.F) {
	f.Add([]byte(`{"method":"thread/started","params":{"threadId":"t","turn":{"id":"x","items":[],"status":"completed"}}}`))
	f.Add([]byte(`{"method":"future/notification","params":{"ok":true}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var notification ServerNotification
		_ = json.Unmarshal(raw, &notification)
	})
}

func FuzzClientRequestUnmarshal(f *testing.F) {
	f.Add([]byte(`{"method":"thread/start","params":{"model":"gpt-5"}}`))
	f.Add([]byte(`{"method":"future/request","params":{"ok":true}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var request ClientRequest
		_ = json.Unmarshal(raw, &request)
	})
}

func FuzzServerRequestUnmarshal(f *testing.F) {
	f.Add([]byte(`{"method":"item/commandExecution/requestApproval","params":{"callId":"1","conversationId":"t","command":["pwd"]}}`))
	f.Add([]byte(`{"method":"future/serverRequest","params":{"ok":true}}`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var request ServerRequest
		_ = json.Unmarshal(raw, &request)
	})
}

func FuzzRequestIDUnmarshal(f *testing.F) {
	f.Add([]byte(`"abc"`))
	f.Add([]byte(`123`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var requestID RequestId
		_ = json.Unmarshal(raw, &requestID)
	})
}

func FuzzThreadListCwdFilterUnmarshal(f *testing.F) {
	f.Add([]byte(`"all"`))
	f.Add([]byte(`["/tmp"]`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var filter ThreadListCwdFilter
		_ = json.Unmarshal(raw, &filter)
	})
}

func FuzzAuthModeUnmarshal(f *testing.F) {
	f.Add([]byte(`"apikey"`))
	f.Add([]byte(`"chatgpt"`))
	f.Add([]byte(`"future_auth_mode"`))
	f.Fuzz(func(t *testing.T, raw []byte) {
		var mode AuthMode
		_ = json.Unmarshal(raw, &mode)
	})
}

func assertJSONEqual(t *testing.T, want, got []byte) {
	t.Helper()
	var left any
	if err := json.Unmarshal(want, &left); err != nil {
		t.Fatalf("json.Unmarshal(want) error = %v", err)
	}
	var right any
	if err := json.Unmarshal(got, &right); err != nil {
		t.Fatalf("json.Unmarshal(got) error = %v", err)
	}
	leftJSON, _ := json.Marshal(left)
	rightJSON, _ := json.Marshal(right)
	if !bytes.Equal(leftJSON, rightJSON) {
		t.Fatalf("json mismatch:\nwant=%s\ngot=%s", leftJSON, rightJSON)
	}
}
