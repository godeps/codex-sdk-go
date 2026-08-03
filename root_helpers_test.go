package codex

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestOptionHelpersAndContextFallback(t *testing.T) {
	t.Parallel()

	options, err := resolveClientOptions(
		WithCodexPath("/bin/codex"),
		WithBaseURL("https://example.test"),
		WithAPIKey("sk-test"),
		WithConfig(map[string]any{"a": 1}),
		WithEnv(map[string]string{"A": "1"}),
		WithRuntimeCacheRoot("/tmp/cache"),
		WithRuntimeVersion("0.144.4"),
		WithAllowPATH(true),
	)
	if err != nil {
		t.Fatalf("resolveClientOptions: %v", err)
	}
	if options.CodexPathOverride != "/bin/codex" || options.BaseURL != "https://example.test" || options.APIKey != "sk-test" {
		t.Fatalf("unexpected options: %#v", options)
	}
	if options.RuntimeCacheRoot != "/tmp/cache" || options.RuntimeVersion != "0.144.4" || !options.AllowPATH {
		t.Fatalf("unexpected runtime options: %#v", options)
	}
	options.Config["a"] = 2
	if got := options.Config["a"]; got != 2 {
		t.Fatalf("config copy failed: %#v", options.Config)
	}

	var turnOpts TurnOptions
	if turnOpts.ContextOrBackground() == nil {
		t.Fatal("ContextOrBackground() = nil")
	}
	ctx := context.WithValue(context.Background(), "k", "v")
	turnOpts.Context = ctx
	if got := turnOpts.ContextOrBackground(); got != ctx {
		t.Fatalf("ContextOrBackground() = %v, want %v", got, ctx)
	}
}

func TestErrorHelpersAndParseTurnMismatch(t *testing.T) {
	t.Parallel()

	rpcErr := &RPCError{Code: -32602, Message: "bad"}
	if got := rpcErr.Error(); got == "" {
		t.Fatal("RPCError.Error() empty")
	}
	if got := (&ServerBusyError{RPCError: rpcErr}).Unwrap(); got != rpcErr {
		t.Fatalf("Unwrap() = %#v, want %#v", got, rpcErr)
	}
	if got := (&RetryLimitExceededError{RPCError: rpcErr}).Unwrap(); got != rpcErr {
		t.Fatalf("Unwrap() = %#v, want %#v", got, rpcErr)
	}
	if got := (&ParseError{RPCError: rpcErr}).Unwrap(); got != rpcErr {
		t.Fatalf("ParseError.Unwrap() = %#v, want %#v", got, rpcErr)
	}
	if got := (&InvalidRequestError{RPCError: rpcErr}).Unwrap(); got != rpcErr {
		t.Fatalf("InvalidRequestError.Unwrap() = %#v, want %#v", got, rpcErr)
	}
	if got := (&MethodNotFoundError{RPCError: rpcErr}).Unwrap(); got != rpcErr {
		t.Fatalf("MethodNotFoundError.Unwrap() = %#v, want %#v", got, rpcErr)
	}
	if got := (&InternalRPCError{RPCError: rpcErr}).Unwrap(); got != rpcErr {
		t.Fatalf("InternalRPCError.Unwrap() = %#v, want %#v", got, rpcErr)
	}
	if got := parseTurnIDMismatch(&InvalidRequestError{RPCError: &RPCError{Message: "expected active turn id `a` but found `b`"}}); got != "b" {
		t.Fatalf("parseTurnIDMismatch() = %q", got)
	}
	if got := parseTurnIDMismatch(errors.New("nope")); got != "" {
		t.Fatalf("parseTurnIDMismatch(non-rpc) = %q", got)
	}
}

func TestDecodeHelpersAndNotificationParsers(t *testing.T) {
	t.Parallel()

	raw := map[string]any{
		"id":          "turn-1",
		"status":      "completed",
		"startedAt":   10.0,
		"completedAt": 12.0,
		"durationMs":  2000.0,
		"error":       map[string]any{"message": "boom"},
		"items": []any{
			map[string]any{"id": "m1", "type": "agent_message", "text": "hello", "phase": "final_answer"},
			map[string]any{"id": "w1", "type": "web_search", "query": "q"},
			map[string]any{"id": "t1", "type": "todo_list", "items": []any{map[string]any{"text": "x", "completed": true}}},
			map[string]any{"id": "e1", "type": "error", "message": "oops"},
		},
	}
	record := decodeTurnRecord(raw)
	if record.ID != "turn-1" || record.Status != TurnStatusCompleted || record.StartedAt == nil || record.CompletedAt == nil || record.Duration != 2*time.Second {
		t.Fatalf("decodeTurnRecord() = %#v", record)
	}
	if len(record.Items) != 4 {
		t.Fatalf("decodeTurnRecord items = %#v", record.Items)
	}
	thread := decodeThreadRecord(map[string]any{
		"id":            "thread-1",
		"name":          "named",
		"path":          "/tmp/thread",
		"cwd":           "/tmp",
		"archived":      true,
		"ephemeral":     false,
		"currentTurnId": "turn-1",
		"status":        map[string]any{"type": "idle"},
		"goal":          map[string]any{"objective": "ship", "status": "complete", "tokenBudget": 1.0, "tokensUsed": 2.0, "updatedAt": 3.0},
		"turns":         []any{raw},
	})
	if thread.ID != "thread-1" || thread.Goal == nil || len(thread.Turns) != 1 {
		t.Fatalf("decodeThreadRecord() = %#v", thread)
	}
	if account := decodeAccount(map[string]any{"type": "chatgpt", "status": "ok"}); account == nil || account.Type != "chatgpt" {
		t.Fatalf("decodeAccount() = %#v", account)
	}
	if got := notificationTurnIDLocal(json.RawMessage(`{"turn":{"id":"turn-2"}}`)); got != "turn-2" {
		t.Fatalf("notificationTurnIDLocal() = %q", got)
	}
	if got := notificationGoalTurnID(json.RawMessage(`{"turnId":"turn-3"}`)); got != "turn-3" {
		t.Fatalf("notificationGoalTurnID() = %q", got)
	}
	if got := inferNotificationMethod(json.RawMessage(`{"turn":{"id":"turn-2","status":"completed"}}`)); got != "turn/completed" {
		t.Fatalf("inferNotificationMethod() = %q", got)
	}
	if got := inferNotificationMethod(json.RawMessage(`{"broken":true}`)); got != "" {
		t.Fatalf("inferNotificationMethod(broken) = %q", got)
	}
	if got, err := buildNotificationEnvelope("turn/started", json.RawMessage(`{"turn":{"id":"turn-2"}}`)); err != nil || len(got) == 0 {
		t.Fatalf("buildNotificationEnvelope() error = %v", err)
	}
}

func TestThreadPayloadAndFieldHelpers(t *testing.T) {
	t.Parallel()

	preset := ApprovalPresetAutoReview
	network := true
	web := true
	ephemeral := true
	payload := threadPayload(ThreadOptions{
		Config:                map[string]any{"custom": "x"},
		ApprovalPreset:        &preset,
		Model:                 "gpt-test",
		ModelProvider:         "openai",
		SandboxMode:           SandboxWorkspaceWrite,
		WorkingDirectory:      "/tmp/project",
		SkipGitRepoCheck:      true,
		ModelReasoningEffort:  ReasoningHigh,
		NetworkAccessEnabled:  &network,
		WebSearchEnabled:      &web,
		ApprovalPolicy:        ApprovalOnFailure,
		AdditionalDirectories: []string{"/a", "/b"},
		BaseInstructions:      "base",
		DeveloperInstructions: "dev",
		Ephemeral:             &ephemeral,
		Personality:           "reviewer",
		ServiceName:           "svc",
		ServiceTier:           "premium",
		SessionStartSource:    "cli",
		ThreadSource:          map[string]any{"kind": "user"},
	})
	if payload["model"] != "gpt-test" || payload["modelProvider"] != "openai" || payload["cwd"] != "/tmp/project" {
		t.Fatalf("threadPayload() = %#v", payload)
	}
	if payload["approvalPolicy"] != "on-request" || payload["approvalsReviewer"] != "auto_review" {
		t.Fatalf("approval payload = %#v", payload)
	}
	config := payload["config"].(map[string]any)
	if config["custom"] != "x" || config["model_reasoning_effort"] != ReasoningHigh {
		t.Fatalf("config payload = %#v", config)
	}

	if got := intField(map[string]any{"a": int64(3)}, "a"); got != 3 {
		t.Fatalf("intField(int64) = %d", got)
	}
	if got := secondsField(map[string]any{"a": 4}, "a"); got == nil || got.Unix() != 4 {
		t.Fatalf("secondsField(int) = %v", got)
	}
	if got := durationField(map[string]any{"a": int64(250)}, "a"); got != 250*time.Millisecond {
		t.Fatalf("durationField(int64) = %v", got)
	}

	env := buildEnvMap(map[string]string{"PATH": "/x"}, "https://example.test", "sk")
	if env["PATH"] != "/x" || env["OPENAI_BASE_URL"] != "https://example.test" || env["CODEX_API_KEY"] != "sk" {
		t.Fatalf("buildEnvMap() = %#v", env)
	}
}

func TestParseThreadItemAndNotificationHelpers(t *testing.T) {
	t.Parallel()

	items := []string{
		`{"id":"m","type":"agent_message","text":"hi","phase":"final_answer"}`,
		`{"id":"c","type":"command_execution","command":"pwd","aggregated_output":"x","status":"completed"}`,
		`{"id":"f","type":"file_change","changes":[{"path":"a","kind":"add"}],"status":"completed"}`,
		`{"id":"t","type":"mcp_tool_call","server":"s","tool":"x","status":"completed"}`,
		`{"id":"w","type":"web_search","query":"q"}`,
		`{"id":"o","type":"todo_list","items":[{"text":"x","completed":false}]}`,
		`{"id":"e","type":"error","message":"boom"}`,
	}
	for _, raw := range items {
		item, err := parseThreadItem(json.RawMessage(raw))
		if err != nil {
			t.Fatalf("parseThreadItem(%s) error = %v", raw, err)
		}
		if item.ItemType() == "" {
			t.Fatalf("ItemType(%T) empty", item)
		}
	}
	if _, err := parseThreadItem(json.RawMessage(`{"type":1}`)); err == nil {
		t.Fatal("parseThreadItem(invalid) error = nil")
	}
	if _, _, err := decodeTurnNotification("thread-1", json.RawMessage(`{"method":"item/completed","params":{"threadId":"thread-1","turnId":"turn-1","item":{"type":1}}}`), nil); err == nil {
		t.Fatal("decodeTurnNotification(invalid item) error = nil")
	}
	if method, _, err := splitNotification(json.RawMessage(`{"params":{}}`)); err == nil || method != "" {
		t.Fatalf("splitNotification(invalid) method=%q err=%v", method, err)
	}
	if got := notificationTurnIDLocal(json.RawMessage(`{"turnId":"turn-1"}`)); got != "turn-1" {
		t.Fatalf("notificationTurnIDLocal(turnId) = %q", got)
	}
}

func TestLegacyAndContextWrappersErrorPaths(t *testing.T) {
	t.Parallel()

	thread := NewThread(NewCodexExec("/missing/codex", nil), CodexOptions{}, ThreadOptions{}, "")
	if _, err := thread.ReadContext(context.Background(), false); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("ReadContext() error = %v", err)
	}
	if err := thread.SetNameContext(context.Background(), "x"); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("SetNameContext() error = %v", err)
	}
	if err := thread.CompactContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("CompactContext() error = %v", err)
	}
	if err := thread.ArchiveContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("ArchiveContext() error = %v", err)
	}
	if err := thread.UnarchiveContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("UnarchiveContext() error = %v", err)
	}
	if _, err := thread.ForkContext(context.Background(), ThreadOptions{}); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("ForkContext() error = %v", err)
	}
	if _, err := thread.GetGoalContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("GetGoalContext() error = %v", err)
	}
	if _, err := thread.SetGoalContext(context.Background(), GoalUpdate{}); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("SetGoalContext() error = %v", err)
	}
	if err := thread.ClearGoalContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("ClearGoalContext() error = %v", err)
	}
	if _, err := thread.PauseGoalContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("PauseGoalContext() error = %v", err)
	}
	if _, err := thread.StartGoalContext(context.Background(), "ship"); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("StartGoalContext() error = %v", err)
	}
	if _, err := thread.Read(false); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Read() error = %v", err)
	}
	if err := thread.SetName("x"); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("SetName() error = %v", err)
	}
	if err := thread.Compact(); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Compact() error = %v", err)
	}

	if _, err := (&ChatGPTLoginHandle{}).WaitContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("ChatGPTLoginHandle.WaitContext() error = %v", err)
	}
	if err := (&ChatGPTLoginHandle{}).CancelContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("ChatGPTLoginHandle.CancelContext() error = %v", err)
	}
	if _, err := (&DeviceCodeLoginHandle{}).WaitContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("DeviceCodeLoginHandle.WaitContext() error = %v", err)
	}
	if err := (&DeviceCodeLoginHandle{}).CancelContext(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("DeviceCodeLoginHandle.CancelContext() error = %v", err)
	}

	var client Client
	if err := client.CloseContext(nil); err != nil {
		t.Fatalf("CloseContext(nil transport) error = %v", err)
	}
	if err := client.WaitContext(nil); err != nil {
		t.Fatalf("WaitContext(nil transport) error = %v", err)
	}
	client.transport = newSDKClient(CodexOptions{})
	if err := client.CloseContext(nil); err == nil {
		t.Fatal("CloseContext(nil ctx) error = nil")
	}
	if err := client.WaitContext(nil); err == nil {
		t.Fatal("WaitContext(nil ctx) error = nil")
	}
	if _, err := client.Account(nil, false); err == nil {
		t.Fatal("Account(nil) error = nil")
	}
	if err := client.Logout(nil); err == nil {
		t.Fatal("Logout(nil) error = nil")
	}
	if err := client.LoginAPIKey(nil, "sk"); err == nil {
		t.Fatal("LoginAPIKey(nil) error = nil")
	}
	if _, err := client.LoginChatGPT(nil); err == nil {
		t.Fatal("LoginChatGPT(nil) error = nil")
	}
	if _, err := client.LoginDeviceCode(nil); err == nil {
		t.Fatal("LoginDeviceCode(nil) error = nil")
	}
	var nilClient *Client
	if err := nilClient.Close(); err != nil {
		t.Fatalf("nil Client Close() error = %v", err)
	}
	if err := nilClient.Wait(); err != nil {
		t.Fatalf("nil Client Wait() error = %v", err)
	}
}

func TestThreadItemsAndUnknownPayload(t *testing.T) {
	t.Parallel()

	cases := []ThreadItem{
		&UnknownItem{Type: "u"},
		&CommandExecutionItem{Type: "command_execution"},
		&FileChangeItem{Type: "file_change"},
		&McpToolCallItem{Type: "mcp_tool_call"},
		&WebSearchItem{Type: "web_search"},
		&ErrorItem{Type: "error"},
		&TodoListItem{Type: "todo_list"},
		&ReasoningItem{Type: "reasoning"},
		&AgentMessageItem{Type: "agent_message"},
	}
	for _, item := range cases {
		if item.ItemType() == "" {
			t.Fatalf("ItemType(%T) empty", item)
		}
	}

	var event ThreadEvent
	if err := json.Unmarshal([]byte(`{"type":"item.completed","item":{"id":"1","type":"future","payload":{"ok":true}}}`), &event); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	unknown, ok := event.Item.(*UnknownItem)
	if !ok || len(unknown.Raw) == 0 {
		t.Fatalf("unknown item = %#v", event.Item)
	}
}

func TestWaitCompatLoginFallbackAndTerminalErrors(t *testing.T) {
	t.Parallel()

	transport := newSDKClient(CodexOptions{CodexPathOverride: "/missing/codex"})
	if _, err := waitCompatLogin(context.Background(), transport, "login-1"); err == nil {
		t.Fatal("waitCompatLogin() error = nil")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := waitCompatLogin(ctx, transport, "login-1"); err == nil {
		t.Fatal("waitCompatLogin(canceled) error = nil")
	}
}

func TestTurnStreamClosedAndResultErrors(t *testing.T) {
	t.Parallel()

	stream := &TurnStream{}
	if _, err := stream.Next(context.Background()); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Next(nil handle) error = %v", err)
	}
	stream = &TurnStream{closed: true, handle: &TurnHandle{client: &Client{}}}
	if _, err := stream.Next(context.Background()); !errors.Is(err, ErrStreamClosed) {
		t.Fatalf("Next(closed) error = %v", err)
	}
	if _, err := collectTurnResult(context.Background(), &TurnStream{terminalEOF: true, handle: &TurnHandle{client: &Client{}}}, "turn-1"); err == nil {
		t.Fatal("collectTurnResult() error = nil")
	}
}

func TestLegacyPublicAPIErrorsAndNilMetadata(t *testing.T) {
	t.Parallel()

	var nilCodex *Codex
	if got := nilCodex.Metadata(); got != nil {
		t.Fatalf("nil Codex Metadata() = %#v", got)
	}

	codex := NewCodex(CodexOptions{CodexPathOverride: "/missing/codex"})
	if got := codex.Metadata(); got != nil {
		t.Fatalf("Metadata before start = %#v", got)
	}
	if _, err := codex.Models(true); err == nil {
		t.Fatal("Models() error = nil")
	}
	if _, err := codex.Account(false); err == nil {
		t.Fatal("Account() error = nil")
	}
	if _, err := codex.StartChatGPTLogin(); err == nil {
		t.Fatal("StartChatGPTLogin() error = nil")
	}
	if _, err := codex.StartChatGPTDeviceCodeLogin(); err == nil {
		t.Fatal("StartChatGPTDeviceCodeLogin() error = nil")
	}
	if err := codex.LoginAPIKey("sk"); err == nil {
		t.Fatal("LoginAPIKey() error = nil")
	}
	var zeroCodex Codex
	if err := zeroCodex.Close(); err != nil {
		t.Fatalf("zero Codex Close() error = %v", err)
	}

	thread := codex.StartThread(ThreadOptions{})
	if _, err := thread.Read(false); err == nil {
		t.Fatal("Thread.Read() error = nil")
	}
	if err := thread.SetName("x"); err == nil {
		t.Fatal("Thread.SetName() error = nil")
	}
	if err := thread.Compact(); err == nil {
		t.Fatal("Thread.Compact() error = nil")
	}
}

func TestClientCloseWaitAndRPCErrorNilReceiver(t *testing.T) {
	t.Parallel()

	var zeroClient Client
	if err := zeroClient.Close(); err != nil {
		t.Fatalf("zero Client Close() error = %v", err)
	}
	if err := zeroClient.Wait(); err != nil {
		t.Fatalf("zero Client Wait() error = %v", err)
	}
	var rpcErr *RPCError
	if got := rpcErr.Error(); got == "" {
		t.Fatal("nil RPCError Error() empty")
	}
}

func TestRealRetryWaitCanceled(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := realRetryWait(ctx, time.Second); !errors.Is(err, context.Canceled) {
		t.Fatalf("realRetryWait() error = %v", err)
	}
}

func TestRetryOnOverloadPublicWrapperAndGoalHelpers(t *testing.T) {
	t.Parallel()

	value, err := RetryOnOverload(context.Background(), RetryOptions{
		MaxAttempts:  1,
		InitialDelay: time.Millisecond,
		MaxDelay:     time.Millisecond,
		JitterRatio:  0,
	}, func(context.Context) (string, error) {
		return "ok", nil
	})
	if err != nil || value != "ok" {
		t.Fatalf("RetryOnOverload() value=%q err=%v", value, err)
	}

	op := newGoalOperation(&Client{}, "thread-1")
	op.fail(errors.New("boom"))
	if _, err := op.awaitStart(context.Background()); err == nil {
		t.Fatal("awaitStart() error = nil")
	}

	op = newGoalOperation(&Client{}, "thread-1")
	op.observeGoalNotification("thread/goal/updated", json.RawMessage(`{"turnId":"turn-1","goal":{"status":"active"}}`))
	if turnID, err := op.awaitStart(context.Background()); err != nil || turnID != "turn-1" {
		t.Fatalf("awaitStart() turnID=%q err=%v", turnID, err)
	}
	op.finishPhysicalTurn("turn-1")
	op.observeGoalNotification("thread/goal/updated", json.RawMessage(`{"turnId":"turn-2","goal":{"status":"active"}}`))
	if turnID, terminal, err := op.awaitContinuation(context.Background(), "turn-1"); err != nil || terminal || turnID != "turn-2" {
		t.Fatalf("awaitContinuation() turnID=%q terminal=%v err=%v", turnID, terminal, err)
	}
	if target := op.interruptTarget(); target != "turn-2" {
		t.Fatalf("interruptTarget() = %q", target)
	}
}

func TestResolveApprovalSettingsAndNilHelpers(t *testing.T) {
	t.Parallel()

	preset := ApprovalPresetAutoReview
	if policy, reviewer := resolveApprovalSettings(&preset, "", true); policy != "on-request" || reviewer != "auto_review" {
		t.Fatalf("resolveApprovalSettings(auto_review) = %q %q", policy, reviewer)
	}
	preset = ApprovalPresetDenyAll
	if policy, reviewer := resolveApprovalSettings(&preset, "", true); policy != "never" || reviewer != "" {
		t.Fatalf("resolveApprovalSettings(deny_all) = %q %q", policy, reviewer)
	}
	if policy, reviewer := resolveApprovalSettings(nil, ApprovalOnFailure, true); policy != "on-failure" || reviewer != "" {
		t.Fatalf("resolveApprovalSettings(legacy) = %q %q", policy, reviewer)
	}
	if policy, reviewer := resolveApprovalSettings(nil, "", false); policy != "" || reviewer != "" {
		t.Fatalf("resolveApprovalSettings(empty) = %q %q", policy, reviewer)
	}
	if got := unixSecondsTime(0); got != nil {
		t.Fatalf("unixSecondsTime(0) = %v", got)
	}
	if got := orString("", "fallback"); got != "fallback" {
		t.Fatalf("orString() = %q", got)
	}
}

func TestDecodeTurnNotificationAndGoalResultErrors(t *testing.T) {
	t.Parallel()

	latest := &Usage{InputTokens: 1}
	if event, terminal, err := decodeTurnNotification("thread-1", json.RawMessage(`{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"in_progress","startedAt":10}}}`), latest); err != nil || terminal || event.Type != "turn.started" {
		t.Fatalf("turn started decode = %#v terminal=%v err=%v", event, terminal, err)
	}
	if event, terminal, err := decodeTurnNotification("thread-1", json.RawMessage(`{"method":"thread/tokenUsage/updated","params":{"threadId":"thread-1","turnId":"turn-1","tokenUsage":{"last":{"input_tokens":1,"cached_input_tokens":0,"output_tokens":2}}}}`), latest); err != nil || terminal || event.Usage == nil {
		t.Fatalf("usage decode = %#v terminal=%v err=%v", event, terminal, err)
	}
	if event, terminal, err := decodeTurnNotification("thread-1", json.RawMessage(`{"method":"turn/completed","params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"failed","error":{"message":"boom"}}}}`), latest); err != nil || !terminal || event.Type != "turn.failed" {
		t.Fatalf("failed decode = %#v terminal=%v err=%v", event, terminal, err)
	}
	if _, _, err := decodeTurnNotification("thread-1", json.RawMessage(`{"method":"item/completed","params":{"threadId":"thread-1","turnId":"turn-1","item":{"type":"reasoning","id":"r1","text":"ok"}}}`), latest); err != nil {
		t.Fatalf("item decode error = %v", err)
	}
	if event, _, err := decodeTurnNotification("thread-1", json.RawMessage(`{"broken":true}`), latest); err == nil && event != nil {
		t.Fatalf("unexpected decode result = %#v", event)
	}

	stream := &GoalStream{done: true, handle: &GoalHandle{}}
	if _, err := collectGoalResult(context.Background(), stream); err == nil {
		t.Fatal("collectGoalResult() error = nil")
	}
}
