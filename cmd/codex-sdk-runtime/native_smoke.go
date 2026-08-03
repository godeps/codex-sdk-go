package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/godeps/codex-sdk-go/internal/appserver"
	"github.com/godeps/codex-sdk-go/internal/runtimebin"
)

type nativeSmokeRequest struct {
	Method string
	Path   string
	Body   []byte
}

type mockResponsesServer struct {
	server   *http.Server
	listener net.Listener
	url      string

	mu       sync.Mutex
	requests []nativeSmokeRequest
}

func newMockResponsesServer() (*mockResponsesServer, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, err
	}
	mock := &mockResponsesServer{listener: listener}
	mux := http.NewServeMux()
	mux.HandleFunc("/v1/models", mock.handleModels)
	mux.HandleFunc("/v1/responses", mock.handleResponses)
	mock.server = &http.Server{Handler: mux}
	mock.url = "http://" + listener.Addr().String()
	go func() {
		_ = mock.server.Serve(listener)
	}()
	return mock, nil
}

func (m *mockResponsesServer) close() error {
	return m.server.Shutdown(context.Background())
}

func (m *mockResponsesServer) requestCount() int {
	m.mu.Lock()
	defer m.mu.Unlock()
	return len(m.requests)
}

func (m *mockResponsesServer) handleModels(w http.ResponseWriter, r *http.Request) {
	m.recordRequest(r, nil)
	writeJSONResponse(w, map[string]any{
		"object": "list",
		"data": []map[string]any{{
			"id":       "mock-model",
			"object":   "model",
			"created":  0,
			"owned_by": "openai",
		}},
	})
}

func (m *mockResponsesServer) handleResponses(w http.ResponseWriter, r *http.Request) {
	body, _ := readRequestBody(r)
	m.recordRequest(r, body)
	w.Header().Set("content-type", "text/event-stream")
	w.WriteHeader(http.StatusOK)
	responseID := "resp-1"
	events := []map[string]any{
		{"type": "response.created", "response": map[string]any{"id": responseID}},
		{
			"type": "response.output_item.done",
			"item": map[string]any{
				"type": "message",
				"role": "assistant",
				"id":   "msg-1",
				"content": []map[string]any{{
					"type": "output_text",
					"text": "runtime smoke ok",
				}},
			},
		},
		{
			"type": "response.completed",
			"response": map[string]any{
				"id": responseID,
				"usage": map[string]any{
					"input_tokens":         2,
					"input_tokens_details": nil,
					"output_tokens":        3,
					"output_tokens_details": map[string]any{
						"reasoning_tokens": 0,
					},
					"total_tokens": 5,
				},
			},
		},
	}
	writer := bufio.NewWriter(w)
	for _, event := range events {
		data, _ := json.Marshal(event)
		_, _ = fmt.Fprintf(writer, "event: %s\ndata: %s\n\n", event["type"], data)
		_ = writer.Flush()
	}
}

func (m *mockResponsesServer) recordRequest(r *http.Request, body []byte) {
	m.mu.Lock()
	defer m.mu.Unlock()
	copied := append([]byte(nil), body...)
	m.requests = append(m.requests, nativeSmokeRequest{
		Method: r.Method,
		Path:   r.URL.Path,
		Body:   copied,
	})
}

func writeJSONResponse(w http.ResponseWriter, payload any) {
	body, _ := json.Marshal(payload)
	w.Header().Set("content-type", "application/json")
	w.Header().Set("content-length", fmt.Sprintf("%d", len(body)))
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(body)
}

func readRequestBody(r *http.Request) ([]byte, error) {
	defer r.Body.Close()
	var buffer bytes.Buffer
	_, err := buffer.ReadFrom(r.Body)
	if err != nil {
		return nil, err
	}
	return buffer.Bytes(), nil
}

func buildNativeSmokeConfig(root string, baseURL string) error {
	codexHome := filepath.Join(root, "codex-home")
	if err := os.MkdirAll(codexHome, 0o755); err != nil {
		return err
	}
	configPath := filepath.Join(codexHome, "config.toml")
	config := fmt.Sprintf(`model = "mock-model"
approval_policy = "never"
sandbox_mode = "read-only"
model_provider = "mock_provider"

[model_providers.mock_provider]
name = "Mock provider for runtime smoke"
base_url = "%s/v1"
wire_api = "responses"
request_max_retries = 0
stream_max_retries = 0
`, baseURL)
	return os.WriteFile(configPath, []byte(config), 0o644)
}

func runNativeSmokeCommand(ctx context.Context, runtimePath string, rootDir string, target runtimebin.TargetSpec, runtimeVersion string, record *runtimebin.ManifestTarget, manifestSHA string) (runtimebin.NativeSmokeResult, error) {
	mockServer, err := newMockResponsesServer()
	if err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	defer func() { _ = mockServer.close() }()

	tempRoot, err := os.MkdirTemp("", "codex-native-smoke-*")
	if err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	defer os.RemoveAll(tempRoot)
	if err := buildNativeSmokeConfig(tempRoot, mockServer.url); err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	workspace := filepath.Join(tempRoot, "workspace")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}

	env := environmentMap()
	env["CODEX_HOME"] = filepath.Join(tempRoot, "codex-home")
	env["CODEX_APP_SERVER_DISABLE_MANAGED_CONFIG"] = "1"
	env["RUST_LOG"] = "warn"
	env["CODEX_API_KEY"] = "runtime-smoke-dummy-key"
	runtimebin.PrependPathDirs(env, runtimebin.TargetPathDirs(rootDir))

	client := appserver.New(appserver.Config{
		ExecutablePath: runtimePath,
		Env:            env,
		ClientName:     "codex_sdk_go_runtime_smoke",
		ClientTitle:    "Codex SDK Go Runtime Smoke",
		ClientVersion:  runtimebin.DefaultSDKVersion,
		Limits:         appserver.DefaultLimits(),
	})
	meta, err := client.Start(ctx)
	if err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	closed := false
	defer func() {
		if !closed {
			_ = client.Close()
		}
	}()

	result := runtimebin.NativeSmokeResult{
		TargetTriple:   target.Triple,
		RuntimeVersion: runtimeVersion,
		RuntimeGOOS:    runtime.GOOS,
		RuntimeGOARCH:  runtime.GOARCH,
		BinaryPath:     runtimePath,
		ManifestSHA256: manifestSHA,
		Runner: runtimebin.RunnerIdentity{
			Hostname:     hostName(),
			RunnerName:   os.Getenv("RUNNER_NAME"),
			RunnerOS:     os.Getenv("RUNNER_OS"),
			RunnerArch:   os.Getenv("RUNNER_ARCH"),
			ImageOS:      os.Getenv("ImageOS"),
			ImageVersion: os.Getenv("ImageVersion"),
		},
	}
	if record != nil {
		result.ArchiveName = record.ArchiveName
		result.ArchiveSHA256 = record.ArchiveSHA256
		result.ArchiveSize = record.ArchiveSize
		result.UpstreamArchiveSHA256 = record.UpstreamArchiveSHA256
		result.UpstreamArchiveSize = record.UpstreamArchiveSize
	}

	versionOut, err := runVersionCommand(ctx, runtimePath)
	if err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	result.VersionOutput = versionOut

	if meta != nil {
		result.InitializeSucceeded = true
		result.ProtocolVersion = meta.ProtocolVersion
		if meta.ServerInfo != nil {
			result.ServerInfo = &struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}{Name: meta.ServerInfo.Name, Version: meta.ServerInfo.Version}
		}
	}

	modelsRaw, err := client.Request(ctx, "model/list", map[string]any{"includeHidden": true})
	if err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	var models struct {
		Data []map[string]any `json:"data"`
	}
	if err := json.Unmarshal(modelsRaw, &models); err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	result.ModelListSucceeded = true
	result.ModelCount = len(models.Data)
	if len(models.Data) > 0 {
		if id, ok := models.Data[0]["id"].(string); ok {
			result.ModelID = id
		}
	}

	threadRaw, err := client.Request(ctx, "thread/start", map[string]any{
		"model":         "mock-model",
		"modelProvider": "mock_provider",
		"sandbox":       "read-only",
		"cwd":           workspace,
	})
	if err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	var threadResponse struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := json.Unmarshal(threadRaw, &threadResponse); err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	result.ThreadStartSucceeded = true
	result.ThreadID = threadResponse.Thread.ID

	turnRaw, err := client.Request(ctx, "turn/start", map[string]any{
		"threadId": result.ThreadID,
		"input": []map[string]any{{
			"type": "text",
			"text": "say hello",
		}},
	})
	if err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	var turnResponse struct {
		Turn struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(turnRaw, &turnResponse); err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	result.TurnStartSucceeded = true
	result.TurnID = turnResponse.Turn.ID
	if err := client.RegisterTurn(result.TurnID); err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	defer client.UnregisterTurn(result.TurnID)
	streamCtx, cancel := context.WithTimeout(ctx, 20*time.Second)
	defer cancel()

	for {
		raw, err := client.NextTurn(result.TurnID, streamCtx)
		if err != nil {
			if err == context.Canceled || err == context.DeadlineExceeded {
				return runtimebin.NativeSmokeResult{}, err
			}
			if err == io.EOF {
				break
			}
			return runtimebin.NativeSmokeResult{}, err
		}
		done, err := accumulateTurnEvent(raw, &result)
		if err != nil {
			return runtimebin.NativeSmokeResult{}, err
		}
		if done {
			break
		}
	}
	result.StreamSucceeded = true
	result.MockRequestCount = mockServer.requestCount()
	if err := client.Close(); err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	closed = true
	result.CloseSucceeded = true
	signatures, err := runtimebin.CollectNativeSignatureEvidence(rootDir, target)
	if err != nil {
		return runtimebin.NativeSmokeResult{}, err
	}
	result.NativeSignatures = signatures
	if result.FinalResponse == "" {
		return runtimebin.NativeSmokeResult{}, fmt.Errorf("native smoke did not produce a final response")
	}
	if result.TurnStatus == "" {
		result.TurnStatus = "completed"
	}
	return result, nil
}

func runVersionCommand(ctx context.Context, runtimePath string) (string, error) {
	cmd := execCommandContext(ctx, runtimePath, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

var execCommandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
	return exec.CommandContext(ctx, name, args...)
}

func hostName() string {
	name, err := os.Hostname()
	if err != nil {
		return ""
	}
	return name
}

func accumulateTurnEvent(raw json.RawMessage, result *runtimebin.NativeSmokeResult) (bool, error) {
	result.StreamEventCount++
	var envelope struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	params := raw
	method := ""
	if err := json.Unmarshal(raw, &envelope); err == nil && envelope.Method != "" {
		method = envelope.Method
		params = envelope.Params
	}
	var payload struct {
		TurnID     string `json:"turnId"`
		ThreadID   string `json:"threadId"`
		TokenUsage *struct {
			Last  map[string]any `json:"last"`
			Total map[string]any `json:"total"`
		} `json:"tokenUsage,omitempty"`
		Usage map[string]any `json:"usage,omitempty"`
		Turn  *struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"turn,omitempty"`
		Item *struct {
			Type  string          `json:"type"`
			Text  string          `json:"text"`
			Phase json.RawMessage `json:"phase,omitempty"`
		} `json:"item,omitempty"`
	}
	if err := json.Unmarshal(params, &payload); err != nil {
		return false, err
	}
	if payload.TokenUsage != nil {
		mergeUsage(&result.Usage, payload.TokenUsage.Last)
		mergeUsage(&result.Usage, payload.TokenUsage.Total)
	}
	mergeUsage(&result.Usage, payload.Usage)
	if payload.Item != nil {
		result.CompletedItemCount++
		if isAgentMessageType(payload.Item.Type) && (messagePhase(payload.Item.Phase) == "final_answer" || result.FinalResponse == "") {
			result.FinalResponse = payload.Item.Text
		}
	}
	if payload.Turn != nil && payload.Turn.Status != "" {
		result.TurnStatus = payload.Turn.Status
		if payload.Turn.Status == "completed" || payload.Turn.Status == "failed" || payload.Turn.Status == "interrupted" {
			return true, nil
		}
	}
	if method == "turn/completed" || method == "turn/failed" {
		return true, nil
	}
	return false, nil
}

func isAgentMessageType(value string) bool {
	return value == "agentMessage" || value == "agent_message"
}

func messagePhase(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var direct string
	if err := json.Unmarshal(raw, &direct); err == nil {
		return direct
	}
	var wrapped struct {
		Value string `json:"value"`
		Type  string `json:"type"`
	}
	if err := json.Unmarshal(raw, &wrapped); err == nil {
		if wrapped.Value != "" {
			return wrapped.Value
		}
		return wrapped.Type
	}
	return ""
}

func mergeUsage(dst **runtimebin.NativeSmokeUsage, fields map[string]any) {
	if len(fields) == 0 {
		return
	}
	usage := ensureUsage(dst)
	assignUsageField(&usage.InputTokens, fields, "input_tokens", "inputTokens")
	assignUsageField(&usage.CachedInputTokens, fields, "cached_input_tokens", "cachedInputTokens")
	assignUsageField(&usage.CacheWriteInputTokens, fields, "cache_write_input_tokens", "cacheWriteInputTokens")
	assignUsageField(&usage.OutputTokens, fields, "output_tokens", "outputTokens")
	assignUsageField(&usage.ReasoningOutputTokens, fields, "reasoning_output_tokens", "reasoningOutputTokens")
	assignUsageField(&usage.TotalTokens, fields, "total_tokens", "totalTokens")
	if usage.TotalTokens == 0 && hasAnyUsage(*usage) {
		usage.TotalTokens = usage.InputTokens + usage.OutputTokens
	}
	*dst = usage
}

func ensureUsage(dst **runtimebin.NativeSmokeUsage) *runtimebin.NativeSmokeUsage {
	if *dst == nil {
		*dst = &runtimebin.NativeSmokeUsage{}
	}
	return *dst
}

func hasAnyUsage(usage runtimebin.NativeSmokeUsage) bool {
	return usage.InputTokens != 0 ||
		usage.CachedInputTokens != 0 ||
		usage.CacheWriteInputTokens != 0 ||
		usage.OutputTokens != 0 ||
		usage.ReasoningOutputTokens != 0 ||
		usage.TotalTokens != 0
}

func assignUsageField(dst *int64, fields map[string]any, keys ...string) {
	for _, key := range keys {
		value, ok := fields[key]
		if !ok {
			continue
		}
		if parsed, ok := numericValue(value); ok {
			*dst = parsed
			return
		}
	}
}

func numericValue(value any) (int64, bool) {
	switch v := value.(type) {
	case int:
		return int64(v), true
	case int32:
		return int64(v), true
	case int64:
		return v, true
	case float64:
		return int64(v), true
	case json.Number:
		parsed, err := v.Int64()
		if err == nil {
			return parsed, true
		}
	}
	return 0, false
}
