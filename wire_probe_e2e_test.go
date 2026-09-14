//go:build e2e_codex

package codex_test

import (
	"context"
	"os/exec"
	"testing"
	"time"

	codex "github.com/godeps/codex-sdk-go"
)

// TestE2E_RealCLINotificationDecoding drives the REAL codex app-server and
// asserts that notifications are decoded into typed events — the regression
// guard for the camelCase wire fixes (item/started, error frames, usage
// casing) captured from codex-cli 0.153.4 on 2026-09-14.
//
// The turn may legitimately fail (e.g. revoked login state): this test
// validates frame DECODING, not model success.
func TestE2E_RealCLINotificationDecoding(t *testing.T) {
	if _, err := exec.LookPath("codex"); err != nil {
		t.Skip("codex CLI not on PATH")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 150*time.Second)
	defer cancel()

	client, err := codex.NewClient(ctx, codex.CodexOptions{AllowPATH: true})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer client.CloseContext(context.Background())

	th, err := client.StartThread(ctx, codex.ThreadOptions{
		WorkingDirectory: t.TempDir(),
		SandboxMode:      codex.SandboxReadOnly,
		ApprovalPolicy:   codex.ApprovalNever,
	})
	if err != nil {
		t.Fatalf("StartThread: %v", err)
	}

	handle, err := th.StartTurnContext(ctx,
		codex.Input{Text: "Compute 17*23 step by step, then reply DONE."},
		codex.TurnOptions{})
	if err != nil {
		t.Fatalf("StartTurn: %v", err)
	}
	stream, err := handle.StreamContext(ctx)
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}

	var (
		sawItemStarted bool
		sawTypedItem   bool
		sawErrorInfo   bool
		sawTerminal    bool
	)
	for {
		ev, err := stream.Next(ctx)
		if err != nil {
			break
		}
		switch ev.Type {
		case "item.started":
			sawItemStarted = true
			if ev.Item != nil && ev.Item.ItemType() != "" {
				sawTypedItem = true
			}
			if ev.ThreadID == "" || ev.TurnID == "" {
				t.Errorf("item.started missing thread/turn ids: %+v", ev)
			}
		case "error":
			if ev.Error != nil && ev.Error.CodexErrorInfo != "" {
				sawErrorInfo = true
				t.Logf("typed error frame: codexErrorInfo=%s", ev.Error.CodexErrorInfo)
			}
		case "turn.completed", "turn.failed":
			sawTerminal = true
		}
		if sawTerminal {
			break
		}
	}

	if !sawItemStarted {
		t.Error("never observed a typed item.started event (raw method leaked into Type?)")
	}
	if !sawTypedItem {
		t.Error("item.started carried no typed Item payload")
	}
	if !sawTerminal {
		t.Error("never observed a terminal turn event")
	}
	// sawErrorInfo is opportunistic: only present when the runtime errors.
	t.Logf("PASS: typed decoding verified (item.started=%v typedItem=%v errorInfo=%v)",
		sawItemStarted, sawTypedItem, sawErrorInfo)
}
