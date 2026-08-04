package codex

import (
	"context"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestFlattenConfigOverridesReturnsDeterministicNestedAssignments(t *testing.T) {
	t.Parallel()

	flag := map[string]any{"flag": true}
	got, err := flattenConfigOverrides(map[string]any{
		"z": json.Number("7"),
		"a": map[string]any{
			"arr": []any{"x", uint(2)},
			"cfg": &flag,
		},
	})
	if err != nil {
		t.Fatalf("flattenConfigOverrides() error = %v", err)
	}

	want := []string{
		`a.arr=["x", 2]`,
		`a.cfg.flag=true`,
		`z=7`,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("flattenConfigOverrides() = %#v, want %#v", got, want)
	}
}

func TestFlattenConfigValueRejectsNonStringMapKeys(t *testing.T) {
	t.Parallel()

	_, err := flattenConfigValue("bad", reflect.ValueOf(map[int]string{1: "x"}))
	if err == nil || !strings.Contains(err.Error(), "map keys must be strings") {
		t.Fatalf("flattenConfigValue() error = %v", err)
	}
}

func TestBuildEnvUsesOverrideAndProcessEnvironmentFallback(t *testing.T) {
	t.Setenv("CODEX_TEST_ENV_ONLY", "from-process")
	t.Setenv(internalOriginatorEnv, "from-process-originator")

	overrideEnv := buildEnv(map[string]string{
		"PATH":                "/override/bin",
		internalOriginatorEnv: "from-override-originator",
	}, "https://override.test", "sk-override")
	overrideMap := make(map[string]string, len(overrideEnv))
	for _, entry := range overrideEnv {
		key, value, _ := strings.Cut(entry, "=")
		overrideMap[key] = value
	}
	if overrideMap["PATH"] != "/override/bin" || overrideMap["OPENAI_BASE_URL"] != "https://override.test" || overrideMap["CODEX_API_KEY"] != "sk-override" {
		t.Fatalf("buildEnv(override) = %#v", overrideMap)
	}
	if overrideMap[internalOriginatorEnv] != "from-override-originator" {
		t.Fatalf("buildEnv(override) originator = %q", overrideMap[internalOriginatorEnv])
	}

	processEnv := buildEnvMap(nil, "", "")
	if processEnv["CODEX_TEST_ENV_ONLY"] != "from-process" {
		t.Fatalf("buildEnvMap(nil) missing process env: %#v", processEnv)
	}
	if processEnv[internalOriginatorEnv] != "from-process-originator" {
		t.Fatalf("buildEnvMap(nil) originator = %q", processEnv[internalOriginatorEnv])
	}
}

func TestBuildTurnPayloadHandlesOutputSchemaValidation(t *testing.T) {
	t.Parallel()

	preset := ApprovalPresetAutoReview
	payload, err := buildTurnPayload("thread-1", TextInput("hello"), TurnOptions{
		ApprovalPreset: &preset,
		Model:          "gpt-test",
		OutputSchema:   map[string]any{"type": "object"},
		Personality:    Personality("reviewer"),
	})
	if err != nil {
		t.Fatalf("buildTurnPayload(valid) error = %v", err)
	}
	if payload["approvalPolicy"] != "on-request" || payload["approvalsReviewer"] != "auto_review" {
		t.Fatalf("buildTurnPayload(valid) approval = %#v", payload)
	}
	if _, ok := payload["outputSchema"].(map[string]any); !ok {
		t.Fatalf("buildTurnPayload(valid) outputSchema = %#v", payload["outputSchema"])
	}

	if _, err := buildTurnPayload("thread-1", TextInput("hello"), TurnOptions{
		OutputSchema: []any{"not", "an", "object"},
	}); err == nil || !strings.Contains(err.Error(), "outputSchema must be a JSON object") {
		t.Fatalf("buildTurnPayload(invalid schema) error = %v", err)
	}
}

func TestIsJSONObjectRecognizesObjectsButNotPointers(t *testing.T) {
	t.Parallel()

	if !isJSONObject(map[string]any{"type": "object"}) {
		t.Fatal("isJSONObject(map) = false")
	}
	if !isJSONObject(struct{ Name string }{Name: "x"}) {
		t.Fatal("isJSONObject(struct) = false")
	}
	if isJSONObject(&struct{ Name string }{Name: "x"}) {
		t.Fatal("isJSONObject(pointer) = true")
	}
}

func TestRealRetryWaitCompletesDeterministicallyWithSynctest(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		done := make(chan error, 1)
		go func() {
			done <- realRetryWait(context.Background(), 5*time.Second)
		}()

		time.Sleep(5 * time.Second)
		if err := <-done; err != nil {
			t.Fatalf("realRetryWait() error = %v", err)
		}
	})
}
