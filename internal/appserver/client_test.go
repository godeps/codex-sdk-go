package appserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/godeps/codex-sdk-go/internal/router"
	"github.com/godeps/codex-sdk-go/protocol"
)

func TestClientStartInitializes(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "basic"})
	defer func() { _ = client.Close() }()

	metadata, err := client.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if metadata.ProtocolVersion != "2026-08-03" {
		t.Fatalf("ProtocolVersion = %q, want %q", metadata.ProtocolVersion, "2026-08-03")
	}
	if metadata.ServerInfo == nil || metadata.ServerInfo.Name != "helper-app-server" {
		t.Fatalf("ServerInfo = %#v", metadata.ServerInfo)
	}
}

func TestClientStartAcceptsDerivedMetadataFromUserAgent(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "derived-initialize"})
	defer func() { _ = client.Close() }()

	metadata, err := client.Start(context.Background())
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if metadata.UserAgent != "helper-app-server/1.0.0" {
		t.Fatalf("UserAgent = %q", metadata.UserAgent)
	}
	if metadata.ServerInfo == nil || metadata.ServerInfo.Name != "helper-app-server" || metadata.ServerInfo.Version != "1.0.0" {
		t.Fatalf("ServerInfo = %#v", metadata.ServerInfo)
	}
}

func TestClientStartRejectsIncompleteMetadataAndReapsChild(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "empty-initialize"})
	start := time.Now()
	_, err := client.Start(context.Background())
	if err == nil || !strings.Contains(err.Error(), "incomplete metadata") {
		t.Fatalf("Start() error = %v, want incomplete metadata", err)
	}
	waitDone := make(chan error, 1)
	go func() { waitDone <- client.Wait() }()
	select {
	case <-waitDone:
	case <-time.After(500 * time.Millisecond):
		t.Fatal("Wait() blocked after init failure, want reaped child")
	}
	if elapsed := time.Since(start); elapsed > 2*time.Second {
		t.Fatalf("Start() cleanup took %v, want < 2s", elapsed)
	}
}

func TestClientCorrelatesOneThousandConcurrentRequestsWithShuffledResponses(t *testing.T) {
	const requests = 1000
	client := New(Config{
		Limits: Limits{
			Router: router.Limits{MaxResponseWaiters: requests},
		},
	})
	client.started = true
	reader, writer := io.Pipe()
	client.stdin = writer
	defer writer.Close()

	type pending struct {
		ID string
		N  int
	}
	requestsSeen := make(chan pending, requests)
	serverDone := make(chan struct{})
	go func() {
		defer close(serverDone)
		decoder := json.NewDecoder(reader)
		var seen []pending
		for len(seen) < requests {
			var msg struct {
				ID     string `json:"id"`
				Method string `json:"method"`
				Params struct {
					N int `json:"n"`
				} `json:"params"`
			}
			if err := decoder.Decode(&msg); err != nil {
				return
			}
			if msg.Method != "test/echo" {
				continue
			}
			seen = append(seen, pending{ID: msg.ID, N: msg.Params.N})
		}
		sort.Slice(seen, func(i, j int) bool { return seen[i].N > seen[j].N })
		for _, req := range seen {
			client.router.RouteResponse("unknown-"+req.ID, json.RawMessage(`{"ignored":true}`), nil)
			client.router.RouteResponse(req.ID, json.RawMessage(fmt.Sprintf(`{"n":%d}`, req.N)), nil)
			client.router.RouteResponse(req.ID, json.RawMessage(`{"late":true}`), nil)
		}
		for _, req := range seen {
			requestsSeen <- req
		}
	}()

	results := make([]int, requests)
	var wg sync.WaitGroup
	for i := 0; i < requests; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			raw, err := client.Request(context.Background(), "test/echo", map[string]any{"n": i})
			if err != nil {
				t.Errorf("Request(%d) error = %v", i, err)
				return
			}
			var response struct {
				N int `json:"n"`
			}
			if err := json.Unmarshal(raw, &response); err != nil {
				t.Errorf("Unmarshal(%d) error = %v", i, err)
				return
			}
			results[i] = response.N
		}(i)
	}
	wg.Wait()
	<-serverDone
	close(requestsSeen)
	for req := range requestsSeen {
		if got := results[req.N]; got != req.N {
			t.Fatalf("Request(%d) received n=%d", req.N, got)
		}
	}
}

func TestClientCanceledRequestDiscardsWaiterPromptly(t *testing.T) {
	client := New(Config{
		Limits: Limits{
			Router: router.Limits{MaxResponseWaiters: 1},
		},
	})
	client.started = true
	reader, writer := io.Pipe()
	client.stdin = writer
	defer writer.Close()

	decoded := make(chan string, 2)
	go func() {
		decoder := json.NewDecoder(reader)
		for {
			var msg struct {
				ID string `json:"id"`
			}
			if err := decoder.Decode(&msg); err != nil {
				return
			}
			decoded <- msg.ID
		}
	}()

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.Request(ctx, "test/cancel", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Request(canceled) error = %v", err)
	}
	secondDone := make(chan struct{})
	go func() {
		defer close(secondDone)
		raw, err := client.Request(context.Background(), "test/ok", nil)
		if err != nil {
			t.Errorf("second Request() error = %v", err)
			return
		}
		if string(raw) != `{"ok":true}` {
			t.Errorf("second Request() = %s", raw)
		}
	}()
	firstID := <-decoded
	secondID := <-decoded
	client.router.RouteResponse(firstID, json.RawMessage(`{"ignored":true}`), nil)
	client.router.RouteResponse(secondID, json.RawMessage(`{"ok":true}`), nil)
	<-secondDone
}

func TestClientRoutesApprovalAndEarlyTurnNotification(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "approval-turn", policy: ApprovalPolicyAutoReview})
	defer func() { _ = client.Close() }()

	if _, err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	if _, err := client.Request(context.Background(), "turn/start", map[string]any{"threadId": "thread-1"}); err != nil {
		t.Fatalf("Request(turn/start) error = %v", err)
	}
	if err := client.RegisterTurn("turn-1"); err != nil {
		t.Fatalf("RegisterTurn() error = %v", err)
	}
	raw, err := client.NextTurn("turn-1")
	if err != nil {
		t.Fatalf("NextTurn() error = %v", err)
	}
	if !strings.Contains(string(raw), `"turn":{"id":"turn-1"}`) {
		t.Fatalf("NextTurn() = %s, want routed turn payload", raw)
	}
}

func TestClientRoutesEarlyLoginNotification(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "login"})
	defer func() { _ = client.Close() }()

	if _, err := client.Start(context.Background()); err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	raw, err := client.Request(context.Background(), "account/login/start", map[string]any{"type": "chatgpt"})
	if err != nil {
		t.Fatalf("Request(account/login/start) error = %v", err)
	}
	var response struct {
		LoginID string `json:"loginId"`
	}
	if err := json.Unmarshal(raw, &response); err != nil {
		t.Fatalf("Unmarshal(login response) error = %v", err)
	}
	if err := client.RegisterLogin(response.LoginID); err != nil {
		t.Fatalf("RegisterLogin() error = %v", err)
	}
	notification, err := client.NextLogin(response.LoginID)
	if err != nil {
		t.Fatalf("NextLogin() error = %v", err)
	}
	if !strings.Contains(string(notification), `"loginId":"login-1"`) {
		t.Fatalf("NextLogin() = %s, want login notification", notification)
	}
}

func TestClientRoutesGoalNotifications(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "goal"})
	defer func() { _ = client.Close() }()
	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := client.RegisterGoal("thread-1"); err != nil {
		t.Fatal(err)
	}
	if err := client.RegisterTurn("turn-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Request(context.Background(), "thread/goal/set", map[string]any{"threadId": "thread-1", "objective": "goal"}); err != nil {
		t.Fatalf("Request(thread/goal/set) error = %v", err)
	}
	wantMethods := []string{"thread/goal/updated", "turn/started", "turn/completed", "thread/goal/cleared"}
	for _, want := range wantMethods {
		raw, err := client.NextGoal("thread-1")
		if err != nil {
			t.Fatalf("NextGoal(%s) error = %v", want, err)
		}
		if !strings.Contains(string(raw), `"method":"`+want+`"`) {
			t.Fatalf("NextGoal() = %s, want method %s", raw, want)
		}
	}
	turnRaw, err := client.NextTurn("turn-1")
	if err != nil {
		t.Fatalf("NextTurn() error = %v", err)
	}
	if !strings.Contains(string(turnRaw), `"method":"turn/started"`) {
		t.Fatalf("NextTurn() = %s, want mirrored turn envelope", turnRaw)
	}
}

func TestClientNextGlobalContextCancel(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "basic"})
	defer func() { _ = client.Close() }()
	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := client.NextGlobal(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("NextGlobal() error = %v, want context canceled", err)
	}
}

func TestClientStderrTailLimitsLinesAndBytes(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "stderr-spam"})
	defer func() { _ = client.Close() }()
	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	time.Sleep(100 * time.Millisecond)
	tail := client.StderrTail()
	lines := strings.Split(strings.TrimSpace(tail), "\n")
	if len(lines) > DefaultLimits().MaxStderrLines {
		t.Fatalf("stderr lines = %d, want <= %d", len(lines), DefaultLimits().MaxStderrLines)
	}
	if len([]byte(tail)) > DefaultLimits().MaxStderrBytes+DefaultLimits().MaxStderrLines {
		t.Fatalf("stderr bytes = %d, want <= %d", len([]byte(tail)), DefaultLimits().MaxStderrBytes)
	}
	if len(lines) == 0 || !strings.Contains(lines[len(lines)-1], "stderr-line-599") {
		t.Fatalf("stderr tail does not retain latest lines: last=%q", lines[len(lines)-1])
	}
}

func TestClientMalformedJSONBroadcastsBlockedWaiters(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "malformed"})
	defer func() { _ = client.Close() }()
	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := client.NextGlobal()
		errCh <- err
	}()
	err := <-errCh
	if err == nil || !strings.Contains(err.Error(), "invalid json") {
		t.Fatalf("NextGlobal() error = %v, want invalid json", err)
	}
}

func TestClientOversizeLineBroadcastsBlockedWaiters(t *testing.T) {
	client := New(Config{
		Limits: Limits{MaxLineBytes: 512},
	})
	line := strings.Repeat("x", 2048)
	client.stdout = io.NopCloser(strings.NewReader(`{"method":"thread/goal/updated","params":{"threadId":"thread-1","blob":"` + line + `"}}` + "\n"))
	client.readLoop()
	_, err := client.NextGlobal()
	if err == nil || (!strings.Contains(err.Error(), "token too long") && !strings.Contains(err.Error(), ErrTransportClosed.Error())) {
		t.Fatalf("NextGlobal() error = %v, want oversize failure", err)
	}
}

func TestClientEOFBroadcastsBlockedWaiters(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "eof"})
	defer func() { _ = client.Close() }()
	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := client.NextGlobal()
		errCh <- err
	}()
	err := <-errCh
	if !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("NextGlobal() error = %v, want transport closed", err)
	}
}

func TestClientGlobalOverflowTerminatesTransportAfterFirstOverflow(t *testing.T) {
	client := New(Config{
		Limits: Limits{
			Router: router.Limits{
				MaxGlobalEvents: 1,
				MaxGlobalBytes:  1024,
				MaxTotalEvents:  8,
				MaxTotalBytes:   4096,
			},
		},
	})
	client.stdout = io.NopCloser(strings.NewReader(strings.Join([]string{
		`{"method":"thread/goal/updated","params":{"threadId":"thread-1"}}`,
		`{"method":"thread/goal/cleared","params":{"threadId":"thread-2"}}`,
	}, "\n") + "\n"))
	client.readLoop()

	first, err := client.NextGlobal()
	if err != nil {
		t.Fatalf("first NextGlobal() error = %v", err)
	}
	if !strings.Contains(string(first), `"threadId":"thread-1"`) {
		t.Fatalf("first NextGlobal() = %s", first)
	}
	_, err = client.NextGlobal()
	if err == nil || !errors.Is(err, ErrLimitExceeded) || !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("second NextGlobal() error = %v, want limit exceeded transport close", err)
	}
}

func TestClientTotalOverflowTerminatesTransportAfterFirstOverflow(t *testing.T) {
	client := New(Config{
		Limits: Limits{
			Router: router.Limits{
				MaxRouteEvents:  4,
				MaxRouteBytes:   1024,
				MaxGlobalEvents: 4,
				MaxGlobalBytes:  1024,
				MaxTotalEvents:  1,
				MaxTotalBytes:   1024,
			},
		},
	})
	if err := client.RegisterTurn("turn-1"); err != nil {
		t.Fatal(err)
	}
	client.stdout = io.NopCloser(strings.NewReader(strings.Join([]string{
		`{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"turn-1"}}}`,
		`{"method":"thread/goal/updated","params":{"threadId":"thread-1"}}`,
	}, "\n") + "\n"))
	client.readLoop()

	first, err := client.NextTurn("turn-1")
	if err != nil {
		t.Fatalf("first NextTurn() error = %v", err)
	}
	if !strings.Contains(string(first), `"turn":{"id":"turn-1"}`) {
		t.Fatalf("first NextTurn() = %s", first)
	}
	_, err = client.NextGlobal()
	if err == nil || !errors.Is(err, ErrLimitExceeded) || !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("NextGlobal() error = %v, want limit exceeded transport close", err)
	}
}

func TestClientDefaultLimitsMatchAcceptanceSpec(t *testing.T) {
	limits := DefaultLimits()
	if limits.MaxLineBytes != 16*mib {
		t.Fatalf("MaxLineBytes = %d, want %d", limits.MaxLineBytes, 16*mib)
	}
	if limits.MaxStderrLines != 400 || limits.MaxStderrBytes != 2*mib {
		t.Fatalf("stderr limits = (%d,%d), want (400,%d)", limits.MaxStderrLines, limits.MaxStderrBytes, 2*mib)
	}
	if limits.MaxServerRequestHandlers != 64 {
		t.Fatalf("MaxServerRequestHandlers = %d, want 64", limits.MaxServerRequestHandlers)
	}
	if limits.ShutdownWait != 2*time.Second || limits.KillWait != 2*time.Second {
		t.Fatalf("shutdown waits = (%v,%v), want (2s,2s)", limits.ShutdownWait, limits.KillWait)
	}
}

func TestClientServerRequestHandlerDoesNotBlockReader(t *testing.T) {
	release := make(chan struct{})
	client := newHelperClient(t, helperOptions{
		mode: "approval-turn",
		handler: func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
			select {
			case <-release:
			case <-ctx.Done():
				return nil, ctx.Err()
			}
			return json.RawMessage(`{"decision":"accept"}`), nil
		},
	})
	defer func() {
		close(release)
		_ = client.Close()
	}()

	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() {
		_, err := client.Request(context.Background(), "turn/start", map[string]any{"threadId": "thread-1"})
		done <- err
	}()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Request(turn/start) error = %v", err)
		}
	case <-time.After(200 * time.Millisecond):
		t.Fatal("Request(turn/start) blocked behind approval handler")
	}
}

func TestClientServerRequestConcurrencyIsBounded(t *testing.T) {
	var active atomic.Int32
	var maxActive atomic.Int32
	release := make(chan struct{})
	client := newHelperClient(t, helperOptions{
		mode: "approval-burst",
		limits: Limits{
			MaxServerRequestHandlers: 64,
		},
		handler: func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
			current := active.Add(1)
			for {
				seen := maxActive.Load()
				if current <= seen || maxActive.CompareAndSwap(seen, current) {
					break
				}
			}
			select {
			case <-release:
			case <-ctx.Done():
				active.Add(-1)
				return nil, ctx.Err()
			}
			active.Add(-1)
			return json.RawMessage(`{"decision":"accept"}`), nil
		},
	})
	defer func() {
		close(release)
		_ = client.Close()
	}()

	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	errCh := make(chan error, 1)
	go func() {
		_, err := client.NextGlobal()
		errCh <- err
	}()
	err := <-errCh
	if err == nil || !strings.Contains(err.Error(), "handler queue limit exceeded") {
		t.Fatalf("NextGlobal() error = %v, want handler queue limit", err)
	}
	if got := maxActive.Load(); got > 64 {
		t.Fatalf("max active handlers = %d, want <= 64", got)
	}
}

func TestDefaultApprovalHandlerPoliciesAndUnknownMethod(t *testing.T) {
	cases := []struct {
		name   string
		method string
		policy ApprovalPolicy
		want   string
		err    bool
	}{
		{name: "deny command", method: "item/commandExecution/requestApproval", policy: ApprovalPolicyDeny, want: `{"decision":"cancel"}`},
		{name: "auto command", method: "item/commandExecution/requestApproval", policy: ApprovalPolicyAutoReview, want: `{"decision":"accept"}`},
		{name: "deny file change", method: "item/fileChange/requestApproval", policy: ApprovalPolicyDeny, want: `{"decision":"cancel"}`},
		{name: "apply patch auto", method: "applyPatchApproval", policy: ApprovalPolicyAutoReview, want: `{"decision":"approved"}`},
		{name: "exec command deny", method: "execCommandApproval", policy: ApprovalPolicyDeny, want: `{"decision":"abort"}`},
		{name: "mcp cancel", method: "mcpServer/elicitation/request", policy: ApprovalPolicyDeny, want: `{"action":"cancel","content":null}`},
		{name: "dynamic tool", method: "item/tool/call", policy: ApprovalPolicyDeny, want: `{"success":false,"contentItems":[]}`},
		{name: "permissions denied", method: "item/permissions/requestApproval", policy: ApprovalPolicyDeny, err: true},
		{name: "tool input denied", method: "item/tool/requestUserInput", policy: ApprovalPolicyDeny, err: true},
		{name: "refresh unsupported", method: "account/chatgptAuthTokens/refresh", policy: ApprovalPolicyDeny, err: true},
		{name: "attestation unsupported", method: "attestation/generate", policy: ApprovalPolicyDeny, err: true},
		{name: "unknown", method: "unknown/method", policy: ApprovalPolicyDeny, err: true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, err := defaultApprovalHandler(tc.method, tc.policy)
			if tc.err {
				if err == nil {
					t.Fatal("defaultApprovalHandler() error = nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("defaultApprovalHandler() error = %v", err)
			}
			if string(got) != tc.want {
				t.Fatalf("defaultApprovalHandler() = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestDefaultHandledServerRequestsMatchProtocolRegistry(t *testing.T) {
	got := DefaultHandledServerRequests()
	if len(got) != len(protocol.ServerRequestRegistry) {
		t.Fatalf("handled methods = %d, protocol registry = %d", len(got), len(protocol.ServerRequestRegistry))
	}
	for method := range protocol.ServerRequestRegistry {
		if _, ok := got[method]; !ok {
			t.Fatalf("missing handled server request %q", method)
		}
	}
	for method := range got {
		if _, ok := protocol.ServerRequestRegistry[method]; !ok {
			t.Fatalf("extra handled server request %q", method)
		}
	}
}

func TestRedactSecrets(t *testing.T) {
	input := strings.Join([]string{
		`authorization=Bearer sk-live-secret`,
		`access_token=tok-123`,
		`password=hunter2`,
		`{"authUrl":"https://example.test/login?token=secret","verificationUrl":"https://example.test/device","userCode":"ABCD-EFGH"}`,
	}, "\n")
	output := redactSecrets(input)
	for _, secret := range []string{"sk-live-secret", "tok-123", "hunter2", "ABCD-EFGH", "https://example.test/login?token=secret"} {
		if strings.Contains(output, secret) {
			t.Fatalf("redaction leaked secret %q in %q", secret, output)
		}
	}
	if !strings.Contains(output, "[REDACTED]") && !strings.Contains(output, "[REDACTED_API_KEY]") {
		t.Fatalf("redaction output = %q, want redaction markers", output)
	}
}

func TestRPCErrorFormattingAndHelpers(t *testing.T) {
	var nilErr *RPCError
	if got := nilErr.Error(); got != "appserver: json-rpc error" {
		t.Fatalf("nil RPCError.Error() = %q", got)
	}
	if got := (&RPCError{Code: 7, Message: "boom"}).Error(); got != "appserver: json-rpc error 7: boom" {
		t.Fatalf("RPCError.Error() = %q", got)
	}
	if got := orDefault("", "fallback"); got != "fallback" {
		t.Fatalf("orDefault(empty) = %q", got)
	}
	if got := minInt(1, 2); got != 1 {
		t.Fatalf("minInt() = %d", got)
	}
	if len(ServerRequestRegistry()) != len(protocol.ServerRequestRegistry) {
		t.Fatalf("ServerRequestRegistry size mismatch")
	}
}

func TestNormalizeLimitsKeepsExplicitFields(t *testing.T) {
	got := normalizeLimits(Limits{
		MaxLineBytes:   7,
		MaxStderrBytes: 9,
		Router: router.Limits{
			MaxRouteEvents: 11,
		},
	})
	if got.MaxLineBytes != 7 || got.MaxStderrBytes != 9 {
		t.Fatalf("normalizeLimits() lost explicit fields: %#v", got)
	}
	if got.MaxServerRequestHandlers != DefaultLimits().MaxServerRequestHandlers {
		t.Fatalf("normalizeLimits() MaxServerRequestHandlers = %d, want default %d", got.MaxServerRequestHandlers, DefaultLimits().MaxServerRequestHandlers)
	}
	if got.Router.MaxRouteEvents != 11 || got.Router.MaxGlobalBytes != router.DefaultLimits().MaxGlobalBytes {
		t.Fatalf("normalizeLimits() router = %#v", got.Router)
	}
}

func TestNormalizeLimitsAllDefaults(t *testing.T) {
	got := normalizeLimits(Limits{})
	want := DefaultLimits()
	if got.MaxLineBytes != want.MaxLineBytes || got.MaxStderrLines != want.MaxStderrLines || got.MaxStderrBytes != want.MaxStderrBytes || got.MaxServerRequestHandlers != want.MaxServerRequestHandlers || got.ShutdownWait != want.ShutdownWait || got.KillWait != want.KillWait {
		t.Fatalf("normalizeLimits({}) = %#v, want %#v", got, want)
	}
}

func TestClientCloseIsIdempotent(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "basic"})
	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("first Close() error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("second Close() error = %v", err)
	}
}

func TestClientCloseTimeoutBranchesAndWait(t *testing.T) {
	client := New(Config{Limits: Limits{ShutdownWait: time.Millisecond, KillWait: time.Millisecond}})
	client.started = true
	client.waitDone = make(chan struct{})
	go func() {
		time.Sleep(5 * time.Millisecond)
		close(client.waitDone)
	}()
	if err := client.Close(); err != nil {
		t.Fatalf("Close(timeout branch) error = %v", err)
	}

	client = New(Config{})
	client.waitDone = make(chan struct{})
	client.waitErr = errors.New("wait err")
	close(client.waitDone)
	if err := client.Wait(); err == nil {
		t.Fatal("Wait() error = nil")
	}
}

func TestClientStartErrors(t *testing.T) {
	if _, err := New(Config{}).Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil for missing executable")
	}
	client := newHelperClient(t, helperOptions{mode: "bad-initialize"})
	defer func() { _ = client.Close() }()
	if _, err := client.Start(context.Background()); err == nil {
		t.Fatal("Start() error = nil for malformed initialize payload")
	}
}

func TestClientDirectErrorPaths(t *testing.T) {
	client := New(Config{})
	if _, err := client.Request(nil, "initialize", nil); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Request(before start) error = %v", err)
	}
	if err := client.Notify(context.Background(), "initialized", nil); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Notify(before start) error = %v", err)
	}
	if err := client.Close(); err != nil {
		t.Fatalf("Close(before start) error = %v", err)
	}

	client = New(Config{ExecutablePath: filepath.Join(t.TempDir(), "missing")})
	if err := client.startProcess(); err == nil {
		t.Fatal("startProcess() missing binary error = nil")
	}

	client = New(Config{})
	client.started = true
	if err := client.startProcess(); err != nil {
		t.Fatalf("startProcess(started) error = %v", err)
	}
}

func TestBuildEnvAndLaunchArgs(t *testing.T) {
	t.Setenv("INHERITED_ENV", "present")
	env := buildEnv(map[string]string{"CUSTOM": "1"})
	if !containsEntry(env, "CUSTOM=1") || !containsEntry(env, "INHERITED_ENV=present") {
		t.Fatalf("buildEnv() = %#v", env)
	}

	client := New(Config{
		ExecutablePath: "codex",
		ConfigFlags:    []string{`model="gpt-5"`},
	})
	args, err := client.launchArgs()
	if err != nil {
		t.Fatalf("launchArgs() error = %v", err)
	}
	want := []string{"codex", "--config", `model="gpt-5"`, "app-server", "--listen", "stdio://"}
	if !reflect.DeepEqual(args, want) {
		t.Fatalf("launchArgs() = %#v, want %#v", args, want)
	}
	if _, err := New(Config{}).launchArgs(); err == nil {
		t.Fatal("launchArgs() error = nil for missing executable")
	}
	overrideArgs, err := New(Config{LaunchArgs: []string{"custom", "app-server"}}).launchArgs()
	if err != nil || !reflect.DeepEqual(overrideArgs, []string{"custom", "app-server"}) {
		t.Fatalf("override launchArgs() = %#v, %v", overrideArgs, err)
	}
}

func TestRequestWriteAndCancelPaths(t *testing.T) {
	client := New(Config{})
	client.started = true
	client.router.FailAll(errors.New("failed"))
	if _, err := client.Request(context.Background(), "x", nil); err == nil {
		t.Fatal("Request(failed router) error = nil")
	}

	client = New(Config{})
	client.started = true
	if _, err := client.Request(context.Background(), "x", nil); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("Request(nil stdin) error = %v", err)
	}

	reader, writer := io.Pipe()
	defer reader.Close()
	client.stdin = writer
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	go io.ReadAll(reader)
	if _, err := client.Request(ctx, "x", nil); !errors.Is(err, context.Canceled) {
		t.Fatalf("Request(canceled) error = %v", err)
	}
	_ = writer.Close()
}

func TestWriteMessageAndDrainStderrHelpers(t *testing.T) {
	client := New(Config{Limits: Limits{MaxLineBytes: 16, MaxStderrBytes: 16}})
	if err := client.writeMessage(map[string]any{"bad": func() {}}); err == nil {
		t.Fatal("writeMessage(marshal) error = nil")
	}
	client.started = true
	if err := client.writeMessage(map[string]any{"k": strings.Repeat("x", 64)}); !errors.Is(err, ErrLimitExceeded) {
		t.Fatalf("writeMessage(limit) error = %v", err)
	}
	client.stderr = io.NopCloser(strings.NewReader("authorization=Bearer sk-secret\nlonglonglonglonglong\n"))
	client.drainStderr()
	if got := client.StderrTail(); strings.Contains(got, "sk-secret") {
		t.Fatalf("StderrTail() leaked secret: %q", got)
	}

	reader, writer := io.Pipe()
	client.stdin = writer
	_ = reader.Close()
	client.limits.MaxLineBytes = 1024
	if err := client.writeMessage(map[string]any{"k": "v"}); err == nil {
		t.Fatal("writeMessage(closed pipe) error = nil")
	}
}

func TestHandleServerRequestWritesResultOrError(t *testing.T) {
	client := New(Config{ApprovalPolicy: ApprovalPolicyAutoReview})
	reader, writer := io.Pipe()
	client.stdin = writer

	done := make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(reader)
		done <- string(buf)
	}()
	client.handleServerRequest(context.Background(), "1", "item/commandExecution/requestApproval", json.RawMessage(`{"command":["pwd"]}`))
	_ = writer.Close()
	if got := <-done; !strings.Contains(got, `"decision":"accept"`) {
		t.Fatalf("handleServerRequest() output = %s", got)
	}

	reader, writer = io.Pipe()
	client.stdin = writer
	done = make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(reader)
		done <- string(buf)
	}()
	client.handleServerRequest(context.Background(), "2", "unknown/method", nil)
	_ = writer.Close()
	if got := <-done; !strings.Contains(got, `"error"`) {
		t.Fatalf("handleServerRequest() error output = %s", got)
	}

	reader, writer = io.Pipe()
	client.stdin = writer
	done = make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(reader)
		done <- string(buf)
	}()
	client.cfg.Approval = func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
		return json.RawMessage(`not-json`), nil
	}
	client.handleServerRequest(context.Background(), "3", "item/tool/call", nil)
	_ = writer.Close()
	if got := <-done; !strings.Contains(got, `"result":{}`) {
		t.Fatalf("handleServerRequest() fallback output = %s", got)
	}

	reader, writer = io.Pipe()
	client.stdin = writer
	done = make(chan string, 1)
	go func() {
		buf, _ := io.ReadAll(reader)
		done <- string(buf)
	}()
	client.cfg.Approval = func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
		return nil, nil
	}
	client.handleServerRequest(context.Background(), "4", "item/tool/call", nil)
	_ = writer.Close()
	if got := <-done; !strings.Contains(got, `"result":{}`) {
		t.Fatalf("handleServerRequest(nil result) output = %s", got)
	}
}

func TestRequestReturnsRPCErrorAndCloseRedactsSecrets(t *testing.T) {
	client := newHelperClient(t, helperOptions{mode: "rpc-error"})
	defer func() { _ = client.Close() }()
	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := client.Request(context.Background(), "model/list", nil); err == nil {
		t.Fatal("Request() rpc error = nil")
	} else {
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) || rpcErr.Code != -32000 {
			t.Fatalf("Request() error = %v, want RPCError -32000", err)
		}
	}

	client = newHelperClient(t, helperOptions{mode: "stderr-exit-secret"})
	if _, err := client.Start(context.Background()); err != nil {
		t.Fatal(err)
	}
	if err := client.Close(); err == nil {
		t.Fatal("Close() error = nil for failing helper")
	} else if strings.Contains(err.Error(), "sk-live-secret") || strings.Contains(err.Error(), "ABCD-EFGH") {
		t.Fatalf("Close() leaked secret: %v", err)
	}
}

type helperOptions struct {
	mode    string
	policy  ApprovalPolicy
	limits  Limits
	handler ApprovalHandler
}

func newHelperClient(t *testing.T, opts helperOptions) *Client {
	t.Helper()
	tempDir := t.TempDir()
	scriptPath := filepath.Join(tempDir, "helper.py")
	script := `#!/usr/bin/env python3
import json
import os
import sys
import time

mode = os.environ.get("APP_SERVER_HELPER_MODE", "basic")
burst_count = int(os.environ.get("APP_SERVER_BURST_COUNT", "130"))

def write(payload):
    sys.stdout.write(json.dumps(payload, separators=(",", ":")) + "\n")
    sys.stdout.flush()

if mode == "stderr-spam":
    for i in range(600):
        sys.stderr.write(f"stderr-line-{i}-" + ("x" * 4096) + "\n")
    sys.stderr.flush()

for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    message = json.loads(line)
    req_id = message.get("id")
    method = message.get("method")

    if method == "initialize":
        if mode == "bad-initialize":
            write({"id": req_id, "result": {"protocolVersion": 1}})
        elif mode == "empty-initialize":
            write({"id": req_id, "result": {"protocolVersion": "", "userAgent": "", "serverInfo": {"name": ""}}})
        elif mode == "derived-initialize":
            write({"id": req_id, "result": {"userAgent": "helper-app-server/1.0.0"}})
        else:
            write({
                "id": req_id,
                "result": {
                    "protocolVersion": "2026-08-03",
                    "userAgent": "helper",
                    "serverInfo": {"name": "helper-app-server", "version": "1.0.0"},
                },
            })
    elif method == "initialized":
        if mode == "approval-turn":
            write({"id": "approval-1", "method": "item/commandExecution/requestApproval", "params": {"command": ["pwd"]}})
        elif mode == "approval-burst":
            for i in range(burst_count):
                write({"id": f"approval-{i}", "method": "item/commandExecution/requestApproval", "params": {"command": ["pwd", str(i)]}})
        elif mode == "malformed":
            sys.stdout.write("{bad-json\n")
            sys.stdout.flush()
        elif mode == "eof":
            sys.stdout.flush()
            sys.exit(0)
        elif mode == "stderr-exit-secret":
            sys.stderr.write('authorization=Bearer sk-live-secret userCode=ABCD-EFGH\n')
            sys.stderr.flush()
            sys.exit(1)
    elif method == "model/list" and mode == "rpc-error":
        write({"id": req_id, "error": {"code": -32000, "message": "server busy"}})
    elif method == "turn/start":
        write({"method": "turn/started", "params": {"turn": {"id": "turn-1"}, "threadId": "thread-1"}})
        write({"id": req_id, "result": {"turn": {"id": "turn-1"}}})
    elif method == "account/login/start":
        write({"method": "account/login/completed", "params": {"loginId": "login-1", "account": {"status": "ok"}}})
        write({"id": req_id, "result": {"loginId": "login-1", "authUrl": "https://example.test/login"}})
    elif method == "thread/goal/set":
        write({"method": "thread/goal/updated", "params": {"threadId": "thread-1", "goal": {"status": "active"}}})
        write({"method": "turn/started", "params": {"threadId": "thread-1", "turn": {"id": "turn-1"}}})
        write({"method": "turn/completed", "params": {"threadId": "thread-1", "turn": {"id": "turn-1"}}})
        write({"method": "thread/goal/cleared", "params": {"threadId": "thread-1"}})
        write({"id": req_id, "result": {"goal": {"status": "active"}}})
`
	if err := os.WriteFile(scriptPath, []byte(script), 0o755); err != nil {
		t.Fatalf("os.WriteFile(helper) error = %v", err)
	}
	limits := opts.limits
	if limits == (Limits{}) {
		limits = DefaultLimits()
	}
	return New(Config{
		LaunchArgs: []string{"python3", scriptPath},
		Env: map[string]string{
			"APP_SERVER_HELPER_MODE": opts.mode,
			"APP_SERVER_BURST_COUNT": "130",
		},
		Approval:       opts.handler,
		ApprovalPolicy: opts.policy,
		Limits:         limits,
	})
}

func containsEntry(entries []string, want string) bool {
	for _, entry := range entries {
		if entry == want {
			return true
		}
	}
	return false
}
