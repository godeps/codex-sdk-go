package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestClientTurnLifecycleAndPayloadMapping(t *testing.T) {
	server, captures := writeContextAPIServer(t)
	client, err := NewClient(context.Background(), CodexOptions{CodexPathOverride: server})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	page, err := client.ListModels(context.Background(), true)
	if err != nil {
		t.Fatalf("ListModels: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "gpt-test" {
		t.Fatalf("ListModels() = %#v", page)
	}

	thread, err := client.StartThread(context.Background(), ThreadOptions{})
	if err != nil {
		t.Fatalf("StartThread: %v", err)
	}

	preset := ApprovalPresetAutoReview
	handle, err := thread.StartTurnContext(context.Background(), ItemsInput(
		UserInput{Type: UserInputText, Text: "hello"},
		DataURLImageInput("data:image/png;base64,AA=="),
		LocalImageInput("/tmp/one.png"),
		SkillInput("review", "/skills/review"),
		MentionInput("spec", "/docs/spec.md"),
	), TurnOptions{
		ApprovalPreset:   &preset,
		Model:            "gpt-test",
		ReasoningEffort:  ReasoningHigh,
		WorkingDirectory: "/tmp/project",
		Personality:      "reviewer",
		SandboxMode:      SandboxWorkspaceWrite,
		ServiceTier:      "premium",
		ReasoningSummary: ReasoningSummaryDetailed,
		OutputSchema: map[string]any{
			"type": "object",
		},
	})
	if err != nil {
		t.Fatalf("StartTurnContext: %v", err)
	}

	stream, err := handle.StreamContext(context.Background())
	if err != nil {
		t.Fatalf("StreamContext: %v", err)
	}
	var got []string
	for {
		event, err := stream.Next(context.Background())
		if err != nil {
			if err.Error() == "EOF" {
				break
			}
			if err == context.Canceled {
				t.Fatalf("Next: %v", err)
			}
			if err == nil {
				continue
			}
			if err == io.EOF {
				break
			}
			t.Fatalf("Next: %v", err)
		}
		got = append(got, event.Type)
	}
	if want := []string{"turn.started", "item.completed", "item.completed", "thread.token_usage.updated", "turn.completed"}; !reflect.DeepEqual(got, want) {
		t.Fatalf("stream event types = %#v, want %#v", got, want)
	}
	if _, err := stream.Next(context.Background()); err != io.EOF {
		t.Fatalf("stream Next after completion = %v, want EOF", err)
	}

	second, err := thread.RunContext(context.Background(), TextInput("hello again"), TurnOptions{})
	if err != nil {
		t.Fatalf("RunContext: %v", err)
	}
	if second.FinalResponse != "final answer" {
		t.Fatalf("FinalResponse = %q", second.FinalResponse)
	}
	if second.Status != TurnStatusCompleted || second.ID == "" || second.StartedAt == nil || second.CompletedAt == nil || second.Duration == 0 {
		t.Fatalf("unexpected TurnResult: %#v", second)
	}

	captured := captures.readJSON(t, "turn_start_1.json")
	if captured["approvalPolicy"] != "on-request" || captured["approvalsReviewer"] != "auto_review" {
		t.Fatalf("approval payload = %#v", captured)
	}
	if captured["model"] != "gpt-test" || captured["cwd"] != "/tmp/project" || captured["reasoningEffort"] != "high" {
		t.Fatalf("turn payload = %#v", captured)
	}
	input := captured["input"].([]any)
	if len(input) != 5 {
		t.Fatalf("input payload = %#v", input)
	}
}

func TestClientListThreadsAndLoginContext(t *testing.T) {
	server, captures := writeContextAPIServer(t)
	client, err := NewClient(context.Background(), CodexOptions{CodexPathOverride: server})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	archived := false
	useStateDBOnly := true
	page, err := client.ListThreads(context.Background(), ThreadListOptions{
		Archived:       &archived,
		Cursor:         "next",
		CWD:            []string{"/tmp/project"},
		Limit:          10,
		ModelProviders: []string{"openai"},
		SearchTerm:     "release",
		SortDirection:  "desc",
		SortKey:        "updatedAt",
		SourceKinds:    []string{"user"},
		UseStateDBOnly: &useStateDBOnly,
	})
	if err != nil {
		t.Fatalf("ListThreads: %v", err)
	}
	if len(page.Data) != 1 || page.Data[0].ID != "thread-list-1" {
		t.Fatalf("ListThreads() = %#v", page)
	}
	listPayload := captures.readJSON(t, "thread_list_1.json")
	if listPayload["cursor"] != "next" || listPayload["searchTerm"] != "release" || listPayload["sortDirection"] != "desc" {
		t.Fatalf("thread/list payload = %#v", listPayload)
	}

	login, err := client.LoginChatGPT(context.Background())
	if err != nil {
		t.Fatalf("LoginChatGPT: %v", err)
	}
	result, err := login.WaitContext(context.Background())
	if err != nil {
		t.Fatalf("WaitContext: %v", err)
	}
	if result.LoginID != "login-browser" || result.Account == nil {
		t.Fatalf("login result = %#v", result)
	}
	device, err := client.LoginDeviceCode(context.Background())
	if err != nil {
		t.Fatalf("LoginDeviceCode: %v", err)
	}
	if _, err := device.WaitContext(context.Background()); err != nil {
		t.Fatalf("device WaitContext: %v", err)
	}
	if err := device.CancelContext(context.Background()); err != nil {
		t.Fatalf("CancelContext: %v", err)
	}
	if err := login.CancelContext(context.Background()); err != nil {
		t.Fatalf("login CancelContext: %v", err)
	}
	if err := client.LoginAPIKey(context.Background(), "sk-secret-value"); err != nil {
		t.Fatalf("LoginAPIKey: %v", err)
	}
	account, err := client.Account(context.Background(), false)
	if err != nil {
		t.Fatalf("Account: %v", err)
	}
	if account.Account == nil || account.RequiresOpenAIAuth {
		t.Fatalf("Account() = %#v", account)
	}
	if err := client.Logout(context.Background()); err != nil {
		t.Fatalf("Logout: %v", err)
	}
	if metadata := client.Metadata(); metadata == nil || metadata.ProtocolVersion != "2026-08-03" {
		t.Fatalf("Metadata() = %#v", metadata)
	}
}

func TestGoalHandleCoalescesTurnsAndBlocksRegularTurns(t *testing.T) {
	server, _ := writeContextAPIServer(t)
	client, err := NewClient(context.Background(), CodexOptions{CodexPathOverride: server})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	handle, err := client.StartGoal(context.Background(), "thread-goal", "ship it")
	if err != nil {
		t.Fatalf("StartGoal: %v", err)
	}
	thread := newClientThread(client, ThreadOptions{}, "thread-goal", true)
	if _, err := thread.StartTurnContext(context.Background(), TextInput("should fail"), TurnOptions{}); err == nil {
		t.Fatal("StartTurnContext while goal active = nil, want error")
	}
	result, err := handle.RunContext(context.Background())
	if err != nil {
		t.Fatalf("RunContext: %v", err)
	}
	t.Logf("goal result items=%#v", result.Items)
	if result.FinalResponse != "goal final" {
		t.Fatalf("FinalResponse = %q", result.FinalResponse)
	}
	if len(result.Items) != 2 {
		t.Fatalf("goal items = %#v", result.Items)
	}
	if goal, err := thread.GetGoalContext(context.Background()); err != nil || goal == nil {
		t.Fatalf("GetGoalContext() goal=%#v err=%v", goal, err)
	}
	status := GoalStatusPaused
	if goal, err := thread.SetGoalContext(context.Background(), GoalUpdate{Status: &status}); err != nil || goal == nil {
		t.Fatalf("SetGoalContext() goal=%#v err=%v", goal, err)
	}
	if goal, err := thread.PauseGoalContext(context.Background()); err != nil || goal == nil {
		t.Fatalf("PauseGoalContext() goal=%#v err=%v", goal, err)
	}
	if err := thread.ClearGoalContext(context.Background()); err != nil {
		t.Fatalf("ClearGoalContext() error = %v", err)
	}
}

func TestCodexLazyStartInitializesOnceUnderConcurrency(t *testing.T) {
	server, captures := writeContextAPIServer(t)
	codex := NewCodex(CodexOptions{CodexPathOverride: server})
	defer func() { _ = codex.Close() }()

	thread := codex.StartThread(ThreadOptions{})
	var wg sync.WaitGroup
	errCh := make(chan error, 3)
	for _, fn := range []func() error{
		func() error { _, err := codex.Models(true); return err },
		func() error { _, err := thread.Read(false); return err },
		func() error { return thread.SetName("x") },
	} {
		wg.Add(1)
		go func(fn func() error) {
			defer wg.Done()
			errCh <- fn()
		}(fn)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent lazy start error = %v", err)
		}
	}
	matches, err := filepath.Glob(filepath.Join(captures.root, "initialize_*.json"))
	if err != nil {
		t.Fatalf("Glob: %v", err)
	}
	if len(matches) != 1 {
		t.Fatalf("initialize count = %d, want 1", len(matches))
	}
}

func TestClientListModelsWhileTurnStreamOpen(t *testing.T) {
	server := writeSemanticServer(t, "delayed_turn")
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
	if event, err := stream.Next(context.Background()); err != nil || event.Type != "turn.started" {
		t.Fatalf("first event = %#v err=%v", event, err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	if _, err := client.ListModels(ctx, true); err != nil {
		t.Fatalf("ListModels while stream open: %v", err)
	}
	for {
		_, err := stream.Next(context.Background())
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("stream.Next() error = %v", err)
		}
	}
}

func TestTurnResultFinalResponseFallbackUsesLatestUnknownPhase(t *testing.T) {
	server := writeSemanticServer(t, "phase_fallback")
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	thread, err := client.StartThread(context.Background(), ThreadOptions{})
	if err != nil {
		t.Fatalf("StartThread: %v", err)
	}
	result, err := thread.RunContext(context.Background(), TextInput("hello"), TurnOptions{})
	if err != nil {
		t.Fatalf("RunContext: %v", err)
	}
	if result.FinalResponse != "second fallback" {
		t.Fatalf("FinalResponse = %q", result.FinalResponse)
	}
}

func TestTurnResultCommentaryOnlyDoesNotProduceFinalResponse(t *testing.T) {
	server := writeSemanticServer(t, "commentary_only")
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	thread, err := client.StartThread(context.Background(), ThreadOptions{})
	if err != nil {
		t.Fatalf("StartThread: %v", err)
	}
	result, err := thread.RunContext(context.Background(), TextInput("hello"), TurnOptions{})
	if err != nil {
		t.Fatalf("RunContext: %v", err)
	}
	if result.FinalResponse != "" {
		t.Fatalf("FinalResponse = %q, want empty", result.FinalResponse)
	}
}

func TestClientThreadLifecycleWrappersAndWait(t *testing.T) {
	server, _ := writeContextAPIServer(t)
	client, err := NewClient(context.Background(), CodexOptions{CodexPathOverride: server})
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}

	thread, err := client.ResumeThread(context.Background(), "thread-ctx-1", ThreadOptions{})
	if err != nil {
		t.Fatalf("ResumeThread: %v", err)
	}
	if thread.ID() != "thread-ctx-1" {
		t.Fatalf("ResumeThread ID = %q", thread.ID())
	}
	forked, err := client.ForkThread(context.Background(), thread.ID(), ThreadOptions{})
	if err != nil {
		t.Fatalf("ForkThread: %v", err)
	}
	if forked.ID() != "thread-fork-1" {
		t.Fatalf("ForkThread ID = %q", forked.ID())
	}
	if err := client.ArchiveThread(context.Background(), thread.ID()); err != nil {
		t.Fatalf("ArchiveThread: %v", err)
	}
	if err := client.UnarchiveThread(context.Background(), thread.ID()); err != nil {
		t.Fatalf("UnarchiveThread: %v", err)
	}
	if err := client.SetThreadName(context.Background(), thread.ID(), "renamed"); err != nil {
		t.Fatalf("SetThreadName: %v", err)
	}
	if err := client.CompactThread(context.Background(), thread.ID()); err != nil {
		t.Fatalf("CompactThread: %v", err)
	}
	if err := thread.ArchiveContext(context.Background()); err != nil {
		t.Fatalf("ArchiveContext: %v", err)
	}
	if err := thread.UnarchiveContext(context.Background()); err != nil {
		t.Fatalf("UnarchiveContext: %v", err)
	}
	if _, err := thread.ForkContext(context.Background(), ThreadOptions{}); err != nil {
		t.Fatalf("ForkContext: %v", err)
	}
	if err := thread.SetNameContext(context.Background(), "other"); err != nil {
		t.Fatalf("SetNameContext: %v", err)
	}
	if err := thread.CompactContext(context.Background()); err != nil {
		t.Fatalf("CompactContext: %v", err)
	}
	if _, err := thread.ReadContext(context.Background(), true); err != nil {
		t.Fatalf("ReadContext: %v", err)
	}
	if err := client.CloseContext(context.Background()); err != nil {
		t.Fatalf("CloseContext: %v", err)
	}
	if err := client.WaitContext(context.Background()); err != nil && !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("WaitContext() error = %v", err)
	}
	if err := client.Wait(); err != nil && !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Wait() error = %v", err)
	}
}

func TestTurnHandleLegacyWrappersAndRepeatedNext(t *testing.T) {
	server, _ := writeContextAPIServer(t)
	client, err := NewClient(context.Background(), CodexOptions{CodexPathOverride: server})
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
	if _, err := handle.Stream(); err != nil {
		t.Fatalf("Stream() = %v", err)
	}
	if _, err := handle.StreamContext(context.Background()); err == nil {
		t.Fatal("second StreamContext() error = nil")
	}

	handle2, err := thread.StartTurnContext(context.Background(), TextInput("hello"), TurnOptions{})
	if err != nil {
		t.Fatalf("StartTurnContext#2: %v", err)
	}
	stream, err := handle2.Stream()
	if err != nil {
		t.Fatalf("Stream: %v", err)
	}
	if _, err := handle2.StreamContext(context.Background()); err == nil {
		t.Fatal("second StreamContext() error = nil")
	}
	for {
		_, err := stream.Next(context.Background())
		if err != nil {
			if err == io.EOF {
				break
			}
			t.Fatalf("Next: %v", err)
		}
	}
	if _, err := stream.Next(context.Background()); err != io.EOF {
		t.Fatalf("Next after EOF = %v", err)
	}
	if err := stream.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if _, err := stream.Next(context.Background()); err != io.EOF {
		t.Fatalf("Next after Close and EOF = %v", err)
	}

	handle3, err := thread.StartTurnContext(context.Background(), TextInput("hello"), TurnOptions{})
	if err != nil {
		t.Fatalf("StartTurnContext#3: %v", err)
	}
	if _, err := handle3.Run(); err != nil {
		t.Fatalf("Run: %v", err)
	}
	streamed, err := thread.RunStreamed(TextInput("hello"), TurnOptions{})
	if err != nil {
		t.Fatalf("RunStreamed: %v", err)
	}
	for range streamed.Events {
	}
	if err := <-streamed.Done; err != nil {
		t.Fatalf("RunStreamed done: %v", err)
	}
	if _, err := thread.Run(TextInput("hello"), TurnOptions{}); err != nil {
		t.Fatalf("Thread.Run: %v", err)
	}
}

type apiCaptures struct {
	root string
}

func (c apiCaptures) readJSON(t *testing.T, name string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(c.root, name))
	if err != nil {
		t.Fatalf("read %s: %v", name, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode %s: %v", name, err)
	}
	return out
}

func writeContextAPIServer(t *testing.T) (string, apiCaptures) {
	t.Helper()

	captures := apiCaptures{root: t.TempDir()}
	script := `#!/usr/bin/env python3
import json, os, pathlib, sys

capture = pathlib.Path(os.environ["API_CAPTURE_DIR"])
capture.mkdir(parents=True, exist_ok=True)
counts = {}
thread_name = "thread-goal"
goal_active = False
goal_phase = 0
turn_counter = 0
logged_out = False

def write(payload):
    sys.stdout.write(json.dumps(payload, separators=(",", ":")) + "\n")
    sys.stdout.flush()

def capture_params(method, params):
    key = method.replace("/", "_")
    counts[key] = counts.get(key, 0) + 1
    capture.joinpath(f"{key}_{counts[key]}.json").write_text(json.dumps(params or {}, separators=(",", ":")))

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    message = json.loads(line)
    req_id = message.get("id")
    method = message.get("method")
    params = message.get("params", {})
    capture_params(method, params)

    if method == "initialize":
        write({"id": req_id, "result": {"protocolVersion": "2026-08-03", "userAgent": "client-api-test", "serverInfo": {"name": "helper-app-server", "version": "1.0.0"}}})
    elif method == "initialized":
        continue
    elif method == "model/list":
        write({"id": req_id, "result": {"data": [{"id": "gpt-test", "model": "gpt-test", "displayName": "GPT Test", "description": "test", "hidden": False, "isDefault": True, "supportedReasoningEfforts": ["low", "high"]}]}})
    elif method == "thread/start":
        thread_id = params.get("threadId") or "thread-ctx-1"
        if params.get("threadId") == "thread-goal":
            thread_id = "thread-goal"
        if goal_active and thread_id == "thread-goal":
            write({"id": req_id, "error": {"code": -32600, "message": "thread has an active goal operation: thread-goal"}})
            continue
        write({"id": req_id, "result": {"thread": {"id": thread_id}}})
    elif method == "thread/resume":
        write({"id": req_id, "result": {"thread": {"id": params["threadId"]}}})
    elif method == "thread/fork":
        write({"id": req_id, "result": {"thread": {"id": "thread-fork-1"}}})
    elif method == "thread/list":
        write({"id": req_id, "result": {"data": [{"id": "thread-list-1", "name": "listed", "path": "/tmp/thread.json", "cwd": "/tmp/project", "archived": False, "ephemeral": False, "status": {"type": "idle"}}], "nextCursor": "", "backwardsCursor": ""}})
    elif method == "thread/read":
        thread_id = params["threadId"]
        if thread_id == "thread-goal":
            current_turn_id = ""
            goal = None
            if goal_active:
                if goal_phase == 0:
                    current_turn_id = "goal-turn-1"
                    goal = {"objective": "ship it", "status": "active"}
                    goal_phase = 1
                elif goal_phase == 1:
                    current_turn_id = "goal-turn-2"
                    goal = {"objective": "ship it", "status": "active"}
                    goal_phase = 2
                else:
                    goal = {"objective": "ship it", "status": "complete"}
                    goal_active = False
            write({"id": req_id, "result": {"thread": {"id": "thread-goal", "name": "goal", "path": "/tmp/thread-goal.json", "cwd": "/tmp/project", "archived": False, "ephemeral": False, "status": {"type": "idle"}, "currentTurnId": current_turn_id, "goal": goal}}})
        else:
            write({"id": req_id, "result": {"thread": {"id": thread_id, "name": thread_name, "path": "/tmp/thread.json", "cwd": "/tmp/project", "archived": False, "ephemeral": False, "status": {"type": "idle"}}}})
    elif method == "thread/name/set":
        thread_name = params.get("name", "")
        write({"id": req_id, "result": {}})
    elif method == "thread/compact/start" or method == "thread/archive" or method == "thread/unarchive" or method == "account/logout" or method == "account/login/cancel" or method == "turn/steer" or method == "turn/interrupt":
        if method == "account/logout":
            logged_out = True
        write({"id": req_id, "result": {}})
    elif method == "account/read":
        if logged_out:
            write({"id": req_id, "result": {"account": None, "requiresOpenaiAuth": True}})
        else:
            write({"id": req_id, "result": {"account": {"type": "chatgpt", "status": "active"}, "requiresOpenaiAuth": False}})
    elif method == "account/login/start":
        kind = params.get("type")
        if kind == "chatgpt":
            write({"method": "account/login/completed", "params": {"loginId": "login-browser", "account": {"type": "chatgpt", "status": "ok"}}})
            write({"id": req_id, "result": {"loginId": "login-browser", "authUrl": "https://example.test/login"}})
        elif kind == "chatgptDeviceCode":
            write({"method": "account/login/completed", "params": {"loginId": "login-device", "account": {"type": "chatgpt", "status": "ok"}}})
            write({"id": req_id, "result": {"loginId": "login-device", "verificationUrl": "https://example.test/device", "userCode": "ABCD-EFGH"}})
        else:
            write({"id": req_id, "result": {"type": "apiKey"}})
    elif method == "turn/start":
        thread_id = params["threadId"]
        if goal_active and thread_id == "thread-goal":
            write({"id": req_id, "error": {"code": -32600, "message": "thread has an active goal operation: thread-goal"}})
            continue
        turn_counter += 1
        turn_id = f"turn-{turn_counter}"
        write({"method": "turn/started", "params": {"threadId": thread_id, "turn": {"id": turn_id, "status": "in_progress", "startedAt": 10}}})
        write({"id": req_id, "result": {"turn": {"id": turn_id, "status": "in_progress", "startedAt": 10}}})
        write({"method": "item/completed", "params": {"threadId": thread_id, "turnId": turn_id, "item": {"id": "item-1", "type": "agent_message", "text": "draft"}}})
        write({"method": "item/completed", "params": {"threadId": thread_id, "turnId": turn_id, "item": {"id": "item-2", "type": "agent_message", "text": "final answer", "phase": "final_answer"}}})
        write({"method": "thread/tokenUsage/updated", "params": {"threadId": thread_id, "turnId": turn_id, "tokenUsage": {"last": {"input_tokens": 1, "cached_input_tokens": 0, "output_tokens": 2}}}})
        write({"method": "turn/completed", "params": {"threadId": thread_id, "turn": {"id": turn_id, "status": "completed", "startedAt": 10, "completedAt": 12, "durationMs": 2000}}})
    elif method == "thread/goal/get":
        write({"id": req_id, "result": {"goal": {"objective": "ship it", "status": "active"}}})
    elif method == "thread/goal/clear":
        goal_active = False
        goal_phase = 0
        write({"id": req_id, "result": {"cleared": True}})
    elif method == "thread/goal/set":
        goal_active = params.get("status") == "active"
        if goal_active:
            goal_phase = 0
        write({"id": req_id, "result": {"goal": {"objective": params.get("objective", "ship it"), "status": params.get("status", "active")}}})
        if goal_active:
            write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "goal": {"objective": params.get("objective", "ship it"), "status": "active"}}})
            write({"method": "turn/started", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "in_progress", "startedAt": 20}}})
            write({"method": "item/completed", "params": {"threadId": "thread-goal", "turnId": "goal-turn-1", "item": {"id": "goal-item-1", "type": "reasoning", "text": "phase one"}}})
            write({"method": "turn/completed", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-1", "status": "completed", "startedAt": 20, "completedAt": 21, "durationMs": 1000}}})
            write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "goal": {"objective": params.get("objective", "ship it"), "status": "active"}}})
            write({"method": "turn/started", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-2", "status": "in_progress", "startedAt": 22}}})
            write({"method": "item/completed", "params": {"threadId": "thread-goal", "turnId": "goal-turn-2", "item": {"id": "goal-item-2", "type": "agent_message", "text": "goal final", "phase": "final_answer"}}})
            write({"method": "thread/tokenUsage/updated", "params": {"threadId": "thread-goal", "turnId": "goal-turn-2", "tokenUsage": {"last": {"input_tokens": 2, "cached_input_tokens": 0, "output_tokens": 3}}}})
            write({"method": "turn/completed", "params": {"threadId": "thread-goal", "turn": {"id": "goal-turn-2", "status": "completed", "startedAt": 22, "completedAt": 24, "durationMs": 2000}}})
            write({"method": "thread/goal/updated", "params": {"threadId": "thread-goal", "goal": {"objective": params.get("objective", "ship it"), "status": "complete"}}})
            write({"method": "thread/goal/cleared", "params": {"threadId": "thread-goal"}})
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
		content := "#!/bin/sh\nAPI_CAPTURE_DIR=" + shellQuote(captures.root) + " exec " + shellQuote(path) + " \"$@\"\n"
		if err := os.WriteFile(wrapper, []byte(content), 0o755); err != nil {
			t.Fatalf("write wrapper: %v", err)
		}
		return wrapper, captures
	}
	t.Setenv("API_CAPTURE_DIR", captures.root)
	return path, captures
}

func writeSemanticServer(t *testing.T, mode string) string {
	t.Helper()

	script := `#!/usr/bin/env python3
import json, sys, threading, time

mode = sys.argv[1]

def write(payload):
    sys.stdout.write(json.dumps(payload, separators=(",", ":")) + "\n")
    sys.stdout.flush()

def delayed_finish():
    time.sleep(0.15)
    write({"method": "item/completed", "params": {"threadId": "thread-1", "turnId": "turn-1", "item": {"id": "item-1", "type": "agent_message", "text": "done", "phase": "final_answer"}}})
    write({"method": "turn/completed", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "completed", "startedAt": 10, "completedAt": 12, "durationMs": 2000}}})

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    message = json.loads(line)
    req_id = message.get("id")
    method = message.get("method")

    if method == "initialize":
        write({"id": req_id, "result": {"protocolVersion": "2026-08-03", "userAgent": "semantic-test", "serverInfo": {"name": "helper-app-server", "version": "1.0.0"}}})
    elif method == "initialized":
        continue
    elif method == "thread/start":
        write({"id": req_id, "result": {"thread": {"id": "thread-1"}}})
    elif method == "model/list":
        write({"id": req_id, "result": {"data": [{"id": "gpt-test", "model": "gpt-test", "displayName": "GPT Test", "description": "test", "hidden": False, "isDefault": True, "supportedReasoningEfforts": ["low", "high"]}]}})
    elif method == "turn/start":
        write({"method": "turn/started", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "in_progress", "startedAt": 10}}})
        write({"id": req_id, "result": {"turn": {"id": "turn-1", "status": "in_progress", "startedAt": 10}}})
        if mode == "delayed_turn":
            threading.Thread(target=delayed_finish, daemon=True).start()
        elif mode == "phase_fallback":
            write({"method": "item/completed", "params": {"threadId": "thread-1", "turnId": "turn-1", "item": {"id": "item-1", "type": "agent_message", "text": "first fallback"}}})
            write({"method": "item/completed", "params": {"threadId": "thread-1", "turnId": "turn-1", "item": {"id": "item-2", "type": "agent_message", "text": "second fallback"}}})
            write({"method": "turn/completed", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "completed", "startedAt": 10, "completedAt": 12, "durationMs": 2000}}})
        elif mode == "commentary_only":
            write({"method": "item/completed", "params": {"threadId": "thread-1", "turnId": "turn-1", "item": {"id": "item-1", "type": "agent_message", "text": "thinking", "phase": "commentary"}}})
            write({"method": "turn/completed", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "completed", "startedAt": 10, "completedAt": 12, "durationMs": 2000}}})
`
	path := filepath.Join(t.TempDir(), "semantic-codex.py")
	if err := os.WriteFile(path, []byte(script), 0o755); err != nil {
		t.Fatalf("write semantic server: %v", err)
	}
	if runtime.GOOS == "windows" {
		return path
	}
	wrapper := filepath.Join(t.TempDir(), "codex")
	content := "#!/bin/sh\nexec " + shellQuote(path) + " " + shellQuote(mode) + " \"$@\"\n"
	if err := os.WriteFile(wrapper, []byte(content), 0o755); err != nil {
		t.Fatalf("write semantic wrapper: %v", err)
	}
	return wrapper
}
