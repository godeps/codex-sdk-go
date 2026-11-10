package codex

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"testing"
	"time"
)

// TestHostIntegrationHelperProcess speaks the actual stdio protocol in a child.
func TestHostIntegrationHelperProcess(t *testing.T) {
	if os.Getenv("CODEX_HOST_HELPER") != "1" {
		return
	}
	scanner := bufio.NewScanner(os.Stdin)
	enc := json.NewEncoder(os.Stdout)
	var pending json.RawMessage
	for scanner.Scan() {
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Result map[string]any  `json:"result"`
			Error  json.RawMessage `json:"error"`
		}
		if json.Unmarshal(scanner.Bytes(), &msg) != nil {
			os.Exit(2)
		}
		switch msg.Method {
		case "initialize":
			_ = enc.Encode(map[string]any{"id": msg.ID, "result": map[string]any{
				"protocolVersion": "2026-08-03",
				"userAgent":       "host-helper/" + os.Getenv("CODEX_PARENT_CANARY"),
				"serverInfo":      map[string]any{"name": "host-helper", "version": "1"},
			}})
		case "model/list":
			pending = append(json.RawMessage(nil), msg.ID...)
			_ = enc.Encode(map[string]any{
				"id": "host-request-7", "method": "item/commandExecution/requestApproval",
				"params": map[string]any{"threadId": "thread-1", "turnId": "turn-1", "itemId": "item-1", "command": "pwd"},
			})
		case "":
			if string(msg.ID) != `"host-request-7"` || pending == nil {
				continue
			}
			decision, _ := msg.Result["decision"].(string)
			if len(msg.Error) > 0 {
				decision = "rpc_error"
			}
			_ = enc.Encode(map[string]any{"id": pending, "result": map[string]any{
				"data": []any{map[string]any{"id": decision, "model": decision, "displayName": decision}},
			}})
			pending = nil
		}
	}
	os.Exit(0)
}

func hostHelperOptions(t *testing.T, env map[string]string) []Option {
	t.Helper()
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	return []Option{
		WithCodexPath(executable), WithEnv(env),
		WithCommandHook(func(cmd *exec.Cmd) error {
			if cmd.Process != nil {
				return errors.New("hook ran after process start")
			}
			cmd.Args = []string{executable, "-test.run=^TestHostIntegrationHelperProcess$"}
			cmd.Env = append(cmd.Env, "CODEX_HOST_HELPER=1")
			return nil
		}),
	}
}

func TestPublicServerRequestHandlerRoundTrip(t *testing.T) {
	for _, tc := range []struct {
		name    string
		handler ServerRequestHandler
		want    string
	}{
		{"default-denies", nil, "cancel"},
		{"custom-allows", func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error) {
			var p struct {
				ItemID string `json:"itemId"`
			}
			if method != "item/commandExecution/requestApproval" || json.Unmarshal(params, &p) != nil || p.ItemID != "item-1" {
				return nil, errors.New("incorrect server request")
			}
			return json.RawMessage(`{"decision":"accept"}`), nil
		}, "accept"},
		{"handler-error", func(context.Context, string, json.RawMessage) (json.RawMessage, error) {
			return nil, errors.New("host denied")
		}, "rpc_error"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			opts := append(hostHelperOptions(t, map[string]string{}), WithServerRequestHandler(tc.handler))
			client, err := NewClient(ctx, opts...)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			page, err := client.ListModels(ctx, false)
			if err != nil {
				t.Fatal(err)
			}
			if len(page.Data) != 1 || page.Data[0].ID != tc.want {
				t.Fatalf("response = %#v, want %s", page, tc.want)
			}
		})
	}
}

func TestPublicExplicitEnvironmentDoesNotLeakParent(t *testing.T) {
	t.Setenv("CODEX_PARENT_CANARY", "parent-only-canary")
	for _, tc := range []struct {
		name string
		env  map[string]string
		want string
	}{
		{"nil-inherits", nil, "parent-only-canary"},
		{"empty-replaces", map[string]string{}, ""},
		{"explicit-replaces", map[string]string{"CODEX_PARENT_CANARY": "explicit"}, "explicit"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			client, err := NewClient(ctx, hostHelperOptions(t, tc.env)...)
			if err != nil {
				t.Fatal(err)
			}
			defer client.Close()
			if got := client.Metadata().UserAgent; got != "host-helper/"+tc.want {
				t.Fatalf("child user agent = %q, want %q", got, "host-helper/"+tc.want)
			}
		})
	}
}

func TestPublicCommandHookCanPreventStart(t *testing.T) {
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	rejected := errors.New("host process policy rejected")
	calls := 0
	_, err = NewClient(context.Background(), WithCodexPath(executable), WithCommandHook(func(cmd *exec.Cmd) error {
		calls++
		if cmd.Process != nil {
			t.Error("child started before hook")
		}
		return rejected
	}))
	if !errors.Is(err, rejected) || calls != 1 {
		t.Fatalf("err=%v calls=%d", err, calls)
	}
}

func TestPublicCloseCancelsServerRequestHandler(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	entered, stopped := make(chan struct{}), make(chan struct{})
	opts := append(hostHelperOptions(t, map[string]string{}), WithServerRequestHandler(func(ctx context.Context, _ string, _ json.RawMessage) (json.RawMessage, error) {
		close(entered)
		<-ctx.Done()
		close(stopped)
		return nil, ctx.Err()
	}))
	client, err := NewClient(ctx, opts...)
	if err != nil {
		t.Fatal(err)
	}
	defer client.Close()
	requestDone := make(chan error, 1)
	go func() { _, err := client.ListModels(ctx, false); requestDone <- err }()
	select {
	case <-entered:
	case <-ctx.Done():
		t.Fatal("handler never entered")
	}
	if err := client.Close(); err != nil {
		t.Fatal(err)
	}
	select {
	case <-stopped:
	case <-ctx.Done():
		t.Fatal("handler ignored shutdown")
	}
	select {
	case err := <-requestDone:
		if err == nil {
			t.Fatal("blocked request unexpectedly succeeded")
		}
	case <-ctx.Done():
		t.Fatal("blocked request not released")
	}
}
