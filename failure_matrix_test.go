package codex

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestTurnTransportCloseAndStableTerminalError(t *testing.T) {
	server := writeFailureServer(t, "turn_transport_close")
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	thread, err := client.StartThread(context.Background(), ThreadOptions{})
	if err != nil {
		t.Fatalf("StartThread: %v", err)
	}
	handle, err := thread.StartTurnContext(context.Background(), TextInput("hello"), TurnOptions{})
	if err != nil {
		t.Fatalf("StartTurnContext: %v", err)
	}
	stream, err := handle.StreamContext(context.Background())
	if err != nil {
		t.Fatalf("StreamContext: %v", err)
	}
	if _, err := stream.Next(context.Background()); err != nil {
		t.Fatalf("first Next() error = %v", err)
	}
	_, firstErr := stream.Next(context.Background())
	if firstErr == nil {
		t.Fatal("second Next() error = nil")
	}
	_, secondErr := stream.Next(context.Background())
	if !errors.Is(secondErr, firstErr) && secondErr.Error() != firstErr.Error() {
		t.Fatalf("terminal error changed: first=%v second=%v", firstErr, secondErr)
	}
}

func TestGoalStartTimeoutAndMalformedRoute(t *testing.T) {
	t.Run("timeout", func(t *testing.T) {
		server := writeFailureServer(t, "goal_timeout")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()

		ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
		defer cancel()
		if _, err := client.StartGoal(ctx, "thread-goal", "ship"); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("StartGoal timeout error = %v", err)
		}
	})

	t.Run("malformed route", func(t *testing.T) {
		server := writeFailureServer(t, "goal_malformed")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()

		if _, err := client.StartGoal(context.Background(), "thread-goal", "ship"); err == nil {
			t.Fatal("StartGoal malformed error = nil")
		}
	})
}

func TestGoalCancelRolloverRetriesInterrupt(t *testing.T) {
	server, captures := writeFailureServerWithCapture(t, "goal_cancel_rollover")
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
	if err != nil {
		t.Fatalf("StartGoal: %v", err)
	}
	if err := handle.CancelContext(context.Background()); err != nil {
		t.Fatalf("CancelContext: %v", err)
	}
	if got := captures.mustRead(t, "interrupt_calls.txt"); got != "goal-turn-1\ngoal-turn-2\n" {
		t.Fatalf("interrupt calls = %q", got)
	}
}

func TestGoalRunFailedAndTransportClose(t *testing.T) {
	t.Run("failed turn", func(t *testing.T) {
		server := writeFailureServer(t, "goal_failed")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()

		handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
		if err != nil {
			t.Fatalf("StartGoal: %v", err)
		}
		if _, err := handle.RunContext(context.Background()); err == nil {
			t.Fatal("RunContext failed-turn error = nil")
		}
	})

	t.Run("transport close", func(t *testing.T) {
		server := writeFailureServer(t, "goal_transport_close")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()

		handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
		if err != nil {
			t.Fatalf("StartGoal: %v", err)
		}
		if _, err := handle.RunContext(context.Background()); err == nil {
			t.Fatal("RunContext transport-close error = nil")
		}
	})
}

func TestGoalStartPreconditions(t *testing.T) {
	t.Run("not idle", func(t *testing.T) {
		server := writeFailureServer(t, "goal_not_idle")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()
		if _, err := client.StartGoal(context.Background(), "thread-goal", "ship"); err == nil {
			t.Fatal("StartGoal not-idle error = nil")
		}
	})

	t.Run("ephemeral", func(t *testing.T) {
		server := writeFailureServer(t, "goal_ephemeral")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()
		if _, err := client.StartGoal(context.Background(), "thread-goal", "ship"); err == nil {
			t.Fatal("StartGoal ephemeral error = nil")
		}
	})
}

func TestLoginTransportCloseAndContextCancel(t *testing.T) {
	t.Run("transport close", func(t *testing.T) {
		server := writeFailureServer(t, "login_transport_close")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()

		handle, err := client.LoginChatGPT(context.Background())
		if err != nil {
			t.Fatalf("LoginChatGPT: %v", err)
		}
		if _, err := handle.WaitContext(context.Background()); err == nil {
			t.Fatal("WaitContext transport-close error = nil")
		}
	})

	t.Run("context cancel", func(t *testing.T) {
		server := writeFailureServer(t, "login_wait")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()

		handle, err := client.LoginChatGPT(context.Background())
		if err != nil {
			t.Fatalf("LoginChatGPT: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		if _, err := handle.WaitContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("WaitContext cancel error = %v", err)
		}
	})

	t.Run("missing login id", func(t *testing.T) {
		server := writeFailureServer(t, "login_missing_id")
		client, err := NewClient(context.Background(), WithCodexPath(server))
		if err != nil {
			t.Fatalf("NewClient: %v", err)
		}
		defer func() { _ = client.Close() }()

		handle, err := client.LoginChatGPT(context.Background())
		if err != nil {
			t.Fatalf("LoginChatGPT: %v", err)
		}
		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()
		if _, err := handle.WaitContext(ctx); !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("WaitContext missing login id error = %v", err)
		}
	})
}

func TestLoginWaitReturnsFailureNotification(t *testing.T) {
	server := writeFailureServer(t, "login_failure")
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	t.Run("browser", func(t *testing.T) {
		handle, err := client.LoginChatGPT(context.Background())
		if err != nil {
			t.Fatalf("LoginChatGPT: %v", err)
		}
		result, err := handle.WaitContext(context.Background())
		if err != nil {
			t.Fatalf("WaitContext: %v", err)
		}
		if result.LoginID != "login-browser" {
			t.Fatalf("LoginID = %q", result.LoginID)
		}
		if result.Account == nil || result.Account.Raw["status"] != "failed" || result.Account.Raw["reason"] != "browser_denied" {
			t.Fatalf("browser failure result = %#v", result)
		}
	})

	t.Run("device code", func(t *testing.T) {
		handle, err := client.LoginDeviceCode(context.Background())
		if err != nil {
			t.Fatalf("LoginDeviceCode: %v", err)
		}
		result, err := handle.WaitContext(context.Background())
		if err != nil {
			t.Fatalf("WaitContext: %v", err)
		}
		if result.LoginID != "login-device" {
			t.Fatalf("LoginID = %q", result.LoginID)
		}
		if result.Account == nil || result.Account.Raw["status"] != "failed" || result.Account.Raw["reason"] != "device_denied" {
			t.Fatalf("device-code failure result = %#v", result)
		}
	})
}

func TestGoalMutualExclusionAndRouteReuse(t *testing.T) {
	server := writeFailureServer(t, "goal_hold")
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	first, err := client.StartGoal(context.Background(), "thread-goal", "ship")
	if err != nil {
		t.Fatalf("StartGoal(first): %v", err)
	}
	if _, err := client.StartGoal(context.Background(), "thread-goal", "ship again"); err == nil {
		t.Fatal("StartGoal(second) error = nil")
	}
	if got := len(client.goals); got != 1 {
		t.Fatalf("active goals during ownership = %d, want 1", got)
	}

	first.Close()
	if got := len(client.goals); got != 0 {
		t.Fatalf("active goals after Close = %d, want 0", got)
	}
	select {
	case <-first.state.done:
	default:
		t.Fatal("first goal route did not stop after Close")
	}

	second, err := client.StartGoal(context.Background(), "thread-goal", "ship once more")
	if err != nil {
		t.Fatalf("StartGoal(after cleanup): %v", err)
	}
	second.Close()
	if got := len(client.goals); got != 0 {
		t.Fatalf("active goals after second Close = %d, want 0", got)
	}
}

func TestGoalCancelPausesBeforeInterruptAndReleasesState(t *testing.T) {
	server, captures := writeFailureServerWithCapture(t, "goal_cancel_rollover")
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
	if err != nil {
		t.Fatalf("StartGoal: %v", err)
	}
	if err := handle.CancelContext(context.Background()); err != nil {
		t.Fatalf("CancelContext: %v", err)
	}
	if got := captures.mustRead(t, "interrupt_calls.txt"); got != "goal-turn-1\ngoal-turn-2\n" {
		t.Fatalf("interrupt calls = %q", got)
	}
	if got := captures.mustRead(t, "request_log.txt"); got == "" {
		t.Fatal("request log empty")
	} else {
		pausePos := strings.Index(got, "thread/goal/set\n")
		interruptPos := strings.Index(got, "turn/interrupt\n")
		if pausePos == -1 || interruptPos == -1 || pausePos > interruptPos {
			t.Fatalf("request order = %q", got)
		}
	}
	if got := len(client.goals); got != 0 {
		t.Fatalf("active goals after cancel = %d, want 0", got)
	}

	reused, err := client.StartGoal(context.Background(), "thread-goal", "retry")
	if err != nil {
		t.Fatalf("StartGoal(reuse): %v", err)
	}
	reused.Close()
}

func TestGoalTerminalPathsReleaseOwnershipExactlyOnce(t *testing.T) {
	cases := []struct {
		name string
		mode string
		run  func(t *testing.T, client *Client) *GoalHandle
	}{
		{
			name: "success",
			mode: "goal_success",
			run: func(t *testing.T, client *Client) *GoalHandle {
				t.Helper()
				handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
				if err != nil {
					t.Fatalf("StartGoal: %v", err)
				}
				if _, err := handle.RunContext(context.Background()); err != nil {
					t.Fatalf("RunContext(success): %v", err)
				}
				return handle
			},
		},
		{
			name: "failed turn",
			mode: "goal_failed",
			run: func(t *testing.T, client *Client) *GoalHandle {
				t.Helper()
				handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
				if err != nil {
					t.Fatalf("StartGoal: %v", err)
				}
				if _, err := handle.RunContext(context.Background()); err == nil {
					t.Fatal("RunContext(failed) error = nil")
				}
				return handle
			},
		},
		{
			name: "timeout",
			mode: "goal_timeout",
			run: func(t *testing.T, client *Client) *GoalHandle {
				t.Helper()
				ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
				defer cancel()
				handle, err := client.StartGoal(ctx, "thread-goal", "ship")
				if !errors.Is(err, context.DeadlineExceeded) {
					t.Fatalf("StartGoal(timeout) error = %v", err)
				}
				return handle
			},
		},
		{
			name: "cancel",
			mode: "goal_cancel_rollover",
			run: func(t *testing.T, client *Client) *GoalHandle {
				t.Helper()
				handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
				if err != nil {
					t.Fatalf("StartGoal: %v", err)
				}
				if err := handle.CancelContext(context.Background()); err != nil {
					t.Fatalf("CancelContext: %v", err)
				}
				return handle
			},
		},
		{
			name: "malformed event",
			mode: "goal_malformed",
			run: func(t *testing.T, client *Client) *GoalHandle {
				t.Helper()
				handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
				if err == nil {
					t.Fatal("StartGoal(malformed) error = nil")
				}
				return handle
			},
		},
		{
			name: "transport close",
			mode: "goal_transport_close",
			run: func(t *testing.T, client *Client) *GoalHandle {
				t.Helper()
				handle, err := client.StartGoal(context.Background(), "thread-goal", "ship")
				if err != nil {
					t.Fatalf("StartGoal: %v", err)
				}
				if _, err := handle.RunContext(context.Background()); err == nil {
					t.Fatal("RunContext(transport close) error = nil")
				}
				return handle
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			server := writeFailureServer(t, tc.mode)
			client, err := NewClient(context.Background(), WithCodexPath(server))
			if err != nil {
				t.Fatalf("NewClient: %v", err)
			}
			defer func() { _ = client.Close() }()

			handle := tc.run(t, client)
			if got := len(client.goals); got != 0 {
				t.Fatalf("active goals after terminal path = %d, want 0", got)
			}
			if handle != nil && handle.state != nil {
				select {
				case <-handle.state.done:
				default:
					t.Fatal("goal route loop still running after terminal path")
				}
			}
		})
	}
}

type failureCaptures struct{ root string }

func (c failureCaptures) mustRead(t *testing.T, name string) string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(c.root, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	return string(data)
}

func writeFailureServer(t *testing.T, mode string) string {
	server, _ := writeFailureServerWithCapture(t, mode)
	return server
}

func writeFailureServerWithCapture(t *testing.T, mode string) (string, failureCaptures) {
	t.Helper()

	captures := failureCaptures{root: t.TempDir()}
	script := `#!/usr/bin/env python3
import json, os, pathlib, sys

mode = os.environ["FAIL_MODE"]
capture = pathlib.Path(os.environ["FAIL_CAPTURE_DIR"])
capture.mkdir(parents=True, exist_ok=True)
interrupt_calls = []

def write(payload):
    sys.stdout.write(json.dumps(payload, separators=(",", ":")) + "\n")
    sys.stdout.flush()

def flush_interrupts():
    capture.joinpath("interrupt_calls.txt").write_text("".join(call + "\n" for call in interrupt_calls))

def append_request(method):
    with capture.joinpath("request_log.txt").open("a", encoding="utf-8") as handle:
        handle.write(method + "\n")

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    message = json.loads(line)
    req_id = message.get("id")
    method = message.get("method")
    params = message.get("params", {})
    append_request(method)
    if method == "initialize":
        write({"id": req_id, "result": {"protocolVersion": "2026-08-03", "userAgent": "failure-test", "serverInfo": {"name": "helper-app-server", "version": "1.0.0"}}})
    elif method == "initialized":
        continue
    elif method == "thread/start":
        write({"id": req_id, "result": {"thread": {"id": "thread-1"}}})
    elif method == "account/login/start":
        if mode == "login_transport_close":
            write({"id": req_id, "result": {"loginId": "login-1", "authUrl": "https://example.test/login"}})
            break
        elif mode == "login_wait":
            write({"id": req_id, "result": {"loginId": "login-1", "authUrl": "https://example.test/login"}})
        elif mode == "login_failure":
            login_type = params.get("type")
            if login_type == "chatgptDeviceCode":
                write({"id": req_id, "result": {"loginId": "login-device", "verificationUrl": "https://example.test/device", "userCode": "ABCD-EFGH"}})
                write({"method": "account/login/completed", "params": {"loginId": "login-device", "account": {"type": "chatgpt", "status": "failed", "reason": "device_denied"}}})
            else:
                write({"id": req_id, "result": {"loginId": "login-browser", "authUrl": "https://example.test/login"}})
                write({"method": "account/login/completed", "params": {"loginId": "login-browser", "account": {"type": "chatgpt", "status": "failed", "reason": "browser_denied"}}})
        elif mode == "login_missing_id":
            write({"id": req_id, "result": {"loginId": "login-1", "authUrl": "https://example.test/login"}})
            write({"method": "account/login/completed", "params": {"account": {"type": "chatgpt", "status": "ok"}}})
        else:
            write({"id": req_id, "result": {"loginId": "login-1", "authUrl": "https://example.test/login"}})
            write({"method": "account/login/completed", "params": {"loginId": "login-1", "account": {"type": "chatgpt", "status": "ok"}}})
    elif method == "thread/read":
        if mode == "goal_not_idle":
            write({"id": req_id, "result": {"thread": {"id": "thread-goal", "name": "goal", "path": "/tmp/thread-goal.json", "cwd": "/tmp/project", "archived": False, "ephemeral": False, "status": {"type": "active"}}}})
        elif mode == "goal_ephemeral":
            write({"id": req_id, "result": {"thread": {"id": "thread-goal", "name": "goal", "path": "", "cwd": "/tmp/project", "archived": False, "ephemeral": True, "status": {"type": "idle"}}}})
        else:
            write({"id": req_id, "result": {"thread": {"id": "thread-goal", "name": "goal", "path": "/tmp/thread-goal.json", "cwd": "/tmp/project", "archived": False, "ephemeral": False, "status": {"type": "idle"}}}})
    elif method == "thread/goal/get":
        write({"id": req_id, "result": {"goal": {"objective": "ship", "status": "active"}}})
    elif method == "thread/goal/clear":
        write({"id": req_id, "result": {"cleared": True}})
    elif method == "thread/goal/set":
        goal_status = params.get("status", "active")
        write({"id": req_id, "result": {"goal": {"objective": "ship", "status": goal_status}}})
        if mode == "goal_cancel_rollover":
            write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "turnId": "goal-turn-1", "goal": {"objective": "ship", "status": "active"}}})
            write({"method": "turn/started", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "in_progress", "startedAt": 20}}})
        elif mode == "goal_hold":
            if goal_status == "active":
                write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "turnId": "goal-turn-1", "goal": {"objective": "ship", "status": "active"}}})
                write({"method": "turn/started", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "in_progress", "startedAt": 20}}})
        elif mode == "goal_failed":
            write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "turnId": "goal-turn-1", "goal": {"objective": "ship", "status": "active"}}})
            write({"method": "turn/started", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "in_progress", "startedAt": 20}}})
            write({"method": "turn/completed", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "failed", "error": {"message": "goal failed"}}}})
            write({"method": "thread/goal/cleared", "params": {"threadId": "thread-goal"}})
        elif mode == "goal_success":
            if goal_status == "active":
                write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "turnId": "goal-turn-1", "goal": {"objective": "ship", "status": "active"}}})
                write({"method": "turn/started", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "in_progress", "startedAt": 20}}})
                write({"method": "item/completed", "params": {"threadId": "thread-goal", "turnId": "goal-turn-1", "item": {"id": "goal-item-1", "type": "agent_message", "text": "goal ok", "phase": "final_answer"}}})
                write({"method": "turn/completed", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "completed", "startedAt": 20, "completedAt": 21, "durationMs": 1000}}})
                write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "goal": {"objective": "ship", "status": "complete"}}})
                write({"method": "thread/goal/cleared", "params": {"threadId": "thread-goal"}})
        elif mode == "goal_transport_close":
            write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "turnId": "goal-turn-1", "goal": {"objective": "ship", "status": "active"}}})
            write({"method": "turn/started", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "in_progress", "startedAt": 20}}})
            break
        elif mode == "goal_malformed":
            sys.stdout.write("{not json}\n")
            sys.stdout.flush()
            break
    elif method == "turn/start":
        if mode == "turn_transport_close":
            write({"method": "turn/started", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "in_progress", "startedAt": 10}}})
            write({"id": req_id, "result": {"turn": {"id": "turn-1", "status": "in_progress", "startedAt": 10}}})
            break
        elif mode == "turn_spam":
            write({"method": "turn/started", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "in_progress", "startedAt": 10}}})
            write({"id": req_id, "result": {"turn": {"id": "turn-1", "status": "in_progress", "startedAt": 10}}})
            for i in range(40):
                write({"method": "item/completed", "params": {"threadId": "thread-1", "turnId": "turn-1", "item": {"id": f"item-{i}", "type": "agent_message", "text": f"chunk-{i}"}}})
            write({"method": "turn/completed", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "completed", "startedAt": 10, "completedAt": 12, "durationMs": 2000}}})
        else:
            write({"id": req_id, "result": {"turn": {"id": "turn-1"}}})
    elif method == "turn/interrupt":
        interrupt_calls.append(params.get("turnId", ""))
        flush_interrupts()
        if mode == "goal_cancel_rollover" and params.get("turnId") == "goal-turn-1":
            write({"id": req_id, "error": {"code": -32600, "message": "expected active turn id goal-turn-1 but found goal-turn-2"}})
        else:
            write({"id": req_id, "result": {}})
    elif method == "account/login/cancel" or method == "thread/goal/set" or method == "thread/goal/clear":
        write({"id": req_id, "result": {}})
    else:
        write({"id": req_id, "result": {}})
`
	path := filepath.Join(t.TempDir(), "fake-codex")
	if runtime.GOOS == "windows" {
		path += ".py"
	}
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write fake server: %v", err)
	}
	if runtime.GOOS != "windows" {
		wrapper := filepath.Join(t.TempDir(), "codex")
		content := "#!/bin/sh\nFAIL_MODE=" + shellQuote(mode) + " FAIL_CAPTURE_DIR=" + shellQuote(captures.root) + " exec " + shellQuote(path) + " \"$@\"\n"
		if err := os.WriteFile(wrapper, []byte(content), 0o755); err != nil {
			t.Fatalf("write wrapper: %v", err)
		}
		return wrapper, captures
	}
	t.Setenv("FAIL_MODE", mode)
	t.Setenv("FAIL_CAPTURE_DIR", captures.root)
	return path, captures
}
