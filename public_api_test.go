package codex

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestPublicAPIsUseAppServer(t *testing.T) {
	ctx := context.Background()
	script := writePublicAPIServer(t)
	client, err := NewClient(ctx, WithCodexPath(script))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	models, err := client.ListModels(ctx, true)
	if err != nil {
		t.Fatalf("Models() error = %v", err)
	}
	if len(models.Data) != 1 || models.Data[0].ID != "gpt-test" {
		t.Fatalf("Models() = %#v", models)
	}

	if err := client.LoginAPIKey(ctx, "sk-test"); err != nil {
		t.Fatalf("LoginAPIKey() error = %v", err)
	}

	login, err := client.LoginChatGPT(ctx)
	if err != nil {
		t.Fatalf("StartChatGPTLogin() error = %v", err)
	}
	if err := login.CancelContext(ctx); err != nil {
		t.Fatalf("login.Cancel() error = %v", err)
	}
	completed, err := login.WaitContext(ctx)
	if err != nil {
		t.Fatalf("login.Wait() error = %v", err)
	}
	if completed.LoginID != "login-browser" {
		t.Fatalf("login.Wait() loginID = %q", completed.LoginID)
	}

	account, err := client.Account(ctx, false)
	if err != nil {
		t.Fatalf("Account() error = %v", err)
	}
	if account.RequiresOpenAIAuth {
		t.Fatalf("Account() RequiresOpenAIAuth = true, want false")
	}

	thread, err := client.StartThread(ctx, ThreadOptions{})
	if err != nil {
		t.Fatalf("StartThread() error = %v", err)
	}
	if err := thread.SetNameContext(ctx, "named thread"); err != nil {
		t.Fatalf("SetName() error = %v", err)
	}
	read, err := thread.ReadContext(ctx, false)
	if err != nil {
		t.Fatalf("Read() error = %v", err)
	}
	if read.ID != "thread-1" || read.Name != "named thread" {
		t.Fatalf("ReadContext() = %#v", read)
	}
	if err := thread.CompactContext(ctx); err != nil {
		t.Fatalf("Compact() error = %v", err)
	}
	if metadata := client.Metadata(); metadata == nil || metadata.ProtocolVersion != "2026-08-03" {
		t.Fatalf("Metadata() = %#v", metadata)
	}
	if err := client.Logout(ctx); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
}

func TestStartChatGPTDeviceCodeLogin(t *testing.T) {
	ctx := context.Background()
	script := writePublicAPIServer(t)
	client, err := NewClient(ctx, WithCodexPath(script))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	login, err := client.LoginDeviceCode(ctx)
	if err != nil {
		t.Fatalf("StartChatGPTDeviceCodeLogin() error = %v", err)
	}
	if login.UserCode != "ABCD-EFGH" {
		t.Fatalf("UserCode = %q", login.UserCode)
	}
	if err := login.CancelContext(ctx); err != nil {
		t.Fatalf("login.Cancel() error = %v", err)
	}
	if _, err := login.WaitContext(ctx); err != nil {
		t.Fatalf("login.Wait() error = %v", err)
	}
}

func TestTurnHandleSteerAndInterrupt(t *testing.T) {
	ctx := context.Background()
	script := writePublicAPIServer(t)
	client, err := NewClient(ctx, WithCodexPath(script))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	thread, err := client.StartThread(ctx, ThreadOptions{})
	if err != nil {
		t.Fatalf("StartThread() error = %v", err)
	}
	handle, err := thread.StartTurnContext(ctx, TextInput("start"), TurnOptions{})
	if err != nil {
		t.Fatalf("StartTurn() error = %v", err)
	}
	if handle.ID() != "turn-1" {
		t.Fatalf("handle.ID() = %q", handle.ID())
	}
	if err := handle.SteerContext(ctx, TextInput("follow up")); err != nil {
		t.Fatalf("Steer() error = %v", err)
	}
	if err := handle.InterruptContext(ctx); err != nil {
		t.Fatalf("Interrupt() error = %v", err)
	}

	resumed, err := client.ResumeThread(ctx, "thread-123", ThreadOptions{})
	if err != nil {
		t.Fatalf("ResumeThread() error = %v", err)
	}
	if resumed.ID() != "thread-123" {
		t.Fatalf("ResumeThread().ID() = %q", resumed.ID())
	}
}

func TestThreadRunContextCollectsResult(t *testing.T) {
	ctx := context.Background()
	script := writePublicAPIServer(t)
	client, err := NewClient(ctx, WithCodexPath(script))
	if err != nil {
		t.Fatalf("NewClient() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	thread, err := client.StartThread(ctx, ThreadOptions{})
	if err != nil {
		t.Fatalf("StartThread() error = %v", err)
	}
	turn, err := thread.RunContext(ctx, TextInput("hello"), TurnOptions{})
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if turn.FinalResponse != "final answer" || len(turn.Items) != 1 || turn.Usage == nil {
		t.Fatalf("Run() = %#v", turn)
	}
}

func writePublicAPIServer(t *testing.T) string {
	t.Helper()

	script := `#!/usr/bin/env python3
import json
import sys

current_thread_id = "thread-1"
thread_name = ""
turn_counter = 0

def write(payload):
    sys.stdout.write(json.dumps(payload, separators=(",", ":")) + "\n")
    sys.stdout.flush()

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    message = json.loads(line)
    req_id = message.get("id")
    method = message.get("method")

    if method == "initialize":
        write({"id": req_id, "result": {"protocolVersion": "2026-08-03", "userAgent": "public-api-test", "serverInfo": {"name": "helper-app-server", "version": "1.0.0"}}})
    elif method == "initialized":
        continue
    elif method == "model/list":
        write({"id": req_id, "result": {"data": [{"id": "gpt-test"}]}})
    elif method == "account/login/start":
        params = message.get("params", {})
        login_type = params.get("type")
        if login_type == "apiKey":
            write({"id": req_id, "result": {"type": "apiKey"}})
        elif login_type == "chatgpt":
            write({"method": "account/login/completed", "params": {"loginId": "login-browser", "account": {"status": "ok"}}})
            write({"id": req_id, "result": {"type": "chatgpt", "loginId": "login-browser", "authUrl": "https://example.test/login"}})
        elif login_type == "chatgptDeviceCode":
            write({"method": "account/login/completed", "params": {"loginId": "login-device", "account": {"status": "ok"}}})
            write({"id": req_id, "result": {"type": "chatgptDeviceCode", "loginId": "login-device", "verificationUrl": "https://example.test/device", "userCode": "ABCD-EFGH"}})
    elif method == "account/read":
        write({"id": req_id, "result": {"account": {"status": "active"}, "requiresOpenaiAuth": False}})
    elif method == "account/logout":
        write({"id": req_id, "result": {}})
    elif method == "account/login/cancel":
        write({"id": req_id, "result": {}})
    elif method == "thread/start":
        write({"id": req_id, "result": {"thread": {"id": current_thread_id}}})
    elif method == "thread/resume":
        current_thread_id = message.get("params", {}).get("threadId", current_thread_id)
        write({"id": req_id, "result": {"thread": {"id": current_thread_id}}})
    elif method == "thread/name/set":
        thread_name = message.get("params", {}).get("name", "")
        write({"id": req_id, "result": {}})
    elif method == "thread/read":
        write({"id": req_id, "result": {"thread": {"id": current_thread_id, "name": thread_name}}})
    elif method == "thread/compact/start":
        write({"id": req_id, "result": {}})
    elif method == "turn/start":
        turn_counter += 1
        turn_id = f"turn-{turn_counter}"
        write({"method": "turn/started", "params": {"threadId": current_thread_id, "turn": {"id": turn_id, "status": "in_progress", "startedAt": 10}}})
        write({"id": req_id, "result": {"turn": {"id": turn_id, "status": "in_progress", "startedAt": 10}}})
        write({"method": "item/completed", "params": {"threadId": current_thread_id, "turnId": turn_id, "item": {"id": "item-1", "type": "agent_message", "text": "final answer", "phase": "final_answer"}}})
        write({"method": "thread/tokenUsage/updated", "params": {"threadId": current_thread_id, "turnId": turn_id, "tokenUsage": {"last": {"input_tokens": 1, "cached_input_tokens": 0, "output_tokens": 2}}}})
        write({"method": "turn/completed", "params": {"threadId": current_thread_id, "turn": {"id": turn_id, "status": "completed", "startedAt": 10, "completedAt": 12, "durationMs": 2000}}})
    elif method == "turn/steer":
        write({"id": req_id, "result": {}})
    elif method == "turn/interrupt":
        write({"id": req_id, "result": {}})
`

	path := filepath.Join(t.TempDir(), "fake-codex")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile(fake codex) error = %v", err)
	}
	return path
}
