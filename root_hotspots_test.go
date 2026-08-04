package codex

import (
	"context"
	"encoding/json"
	"errors"
	"math"
	"reflect"
	"testing"

	"github.com/godeps/codex-sdk-go/internal/appserver"
)

func TestTranslateAppServerErrorAndMetadataSnapshot(t *testing.T) {
	t.Parallel()

	if got := translateAppServerError(nil); got != nil {
		t.Fatalf("translateAppServerError(nil) = %v", got)
	}
	if got := translateAppServerError(appserver.ErrTransportClosed); !errors.Is(got, ErrTransportClosed) {
		t.Fatalf("transport error = %v", got)
	}
	if got := translateAppServerError(appserver.ErrLimitExceeded); !errors.Is(got, ErrLimitExceeded) {
		t.Fatalf("limit error = %v", got)
	}
	rawData := json.RawMessage(`{"detail":"x"}`)
	got := translateAppServerError(&appserver.RPCError{Code: -32602, Message: "bad", Data: rawData})
	rpcErr, ok := got.(*RPCError)
	if !ok || rpcErr.Code != -32602 || rpcErr.Message != "bad" || string(rpcErr.Data) != string(rawData) {
		t.Fatalf("rpc translation = %#v", got)
	}

	client := &sdkClient{}
	if snap := client.metadataSnapshot(); snap != nil {
		t.Fatalf("metadataSnapshot() = %#v", snap)
	}
	client.metadata = &appserver.Metadata{ProtocolVersion: "2026-08-03", UserAgent: "ua", ServerInfo: &struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	}{Name: "srv", Version: "1"}}
	snap := client.metadataSnapshot()
	if snap == nil || snap.ProtocolVersion != "2026-08-03" || snap.ServerInfo == nil || snap.ServerInfo.Name != "srv" {
		t.Fatalf("metadataSnapshot() = %#v", snap)
	}
}

func TestTOMLHelpers(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		value float64
		want  string
	}{
		{math.NaN(), "nan"},
		{math.Inf(1), "inf"},
		{math.Inf(-1), "-inf"},
		{3.5, "3.5"},
	} {
		if got := formatTOMLFloat(tc.value); got != tc.want {
			t.Fatalf("formatTOMLFloat(%v) = %q, want %q", tc.value, got, tc.want)
		}
	}

	literal, err := encodeTOMLLiteral(reflect.ValueOf([]any{"x", true, 3.5}))
	if err != nil || literal != "[\"x\", true, 3.5]" {
		t.Fatalf("encodeTOMLLiteral(array) = %q err=%v", literal, err)
	}
	if literal, err := encodeTOMLLiteral(reflect.ValueOf(map[string]any{"x": 1})); err != nil || literal != "{x=1}" {
		t.Fatalf("encodeTOMLLiteral(map) = %q err=%v", literal, err)
	}
}

func TestConfigFlatteningRejectsNilValues(t *testing.T) {
	t.Parallel()
	if _, err := flattenConfigOverrides(map[string]any{"bad": nil}); err == nil {
		t.Fatal("flattenConfigOverrides(nil value) error = nil")
	}
}

func TestCancelGoalOperationDirect(t *testing.T) {
	server, captures := writeFailureServerWithCapture(t, "goal_cancel_rollover")
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	state := &goalOperation{client: client, threadID: "thread-goal", currentTurnID: "goal-turn-1", lastTurnID: "goal-turn-1"}
	client.cancelGoalOperation(state)
	if got := captures.mustRead(t, "interrupt_calls.txt"); got != "goal-turn-1\ngoal-turn-2\n" {
		t.Fatalf("interrupt calls = %q", got)
	}
}

func TestSDKClientRouteHelperErrorBranches(t *testing.T) {
	t.Parallel()

	client := newManagedSDKClient(CodexOptions{CodexPathOverride: "/missing/codex"})
	if err := client.registerTurn("turn-1"); err == nil {
		t.Fatal("registerTurn() error = nil")
	}
	if err := client.registerLogin("login-1"); err == nil {
		t.Fatal("registerLogin() error = nil")
	}
	if err := client.registerGoal("thread-1"); err == nil {
		t.Fatal("registerGoal() error = nil")
	}
	if _, err := client.nextLogin(context.Background(), "login-1"); err == nil {
		t.Fatal("nextLogin() error = nil")
	}
	if _, err := client.nextGoal(context.Background(), "thread-1"); err == nil {
		t.Fatal("nextGoal() error = nil")
	}
}
