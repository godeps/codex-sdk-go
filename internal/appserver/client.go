package appserver

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/godeps/codex-sdk-go/internal/router"
	"github.com/godeps/codex-sdk-go/protocol"
)

const mib = 1024 * 1024

var (
	ErrTransportClosed = errors.New("appserver: transport closed")
	ErrLimitExceeded   = errors.New("appserver: limit exceeded")
)

var defaultHandledServerRequests = map[string]struct{}{
	"account/chatgptAuthTokens/refresh":     {},
	"applyPatchApproval":                    {},
	"attestation/generate":                  {},
	"execCommandApproval":                   {},
	"item/commandExecution/requestApproval": {},
	"item/fileChange/requestApproval":       {},
	"item/permissions/requestApproval":      {},
	"item/tool/call":                        {},
	"item/tool/requestUserInput":            {},
	"mcpServer/elicitation/request":         {},
}

// RPCError captures a JSON-RPC error response from the app-server.
type RPCError struct {
	Code    int
	Message string
	Data    json.RawMessage
}

func (e *RPCError) Error() string {
	if e == nil {
		return "appserver: json-rpc error"
	}
	return fmt.Sprintf("appserver: json-rpc error %d: %s", e.Code, e.Message)
}

// ApprovalHandler resolves app-server initiated requests.
type ApprovalHandler func(ctx context.Context, method string, params json.RawMessage) (json.RawMessage, error)

// ApprovalPolicy controls the built-in request handler behavior.
type ApprovalPolicy int

const (
	ApprovalPolicyDeny ApprovalPolicy = iota
	ApprovalPolicyAutoReview
)

// Limits bounds the transport's in-memory state.
type Limits struct {
	MaxLineBytes             int
	MaxStderrLines           int
	MaxStderrBytes           int
	MaxServerRequestHandlers int
	ShutdownWait             time.Duration
	KillWait                 time.Duration
	Router                   router.Limits
}

// DefaultLimits returns the fixed transport defaults.
func DefaultLimits() Limits {
	return Limits{
		MaxLineBytes:             16 * mib,
		MaxStderrLines:           400,
		MaxStderrBytes:           2 * mib,
		MaxServerRequestHandlers: 64,
		ShutdownWait:             2 * time.Second,
		KillWait:                 2 * time.Second,
		Router:                   router.DefaultLimits(),
	}
}

func normalizeLimits(limits Limits) Limits {
	defaults := DefaultLimits()
	if limits.MaxLineBytes <= 0 {
		limits.MaxLineBytes = defaults.MaxLineBytes
	}
	if limits.MaxStderrLines <= 0 {
		limits.MaxStderrLines = defaults.MaxStderrLines
	}
	if limits.MaxStderrBytes <= 0 {
		limits.MaxStderrBytes = defaults.MaxStderrBytes
	}
	if limits.MaxServerRequestHandlers <= 0 {
		limits.MaxServerRequestHandlers = defaults.MaxServerRequestHandlers
	}
	if limits.ShutdownWait <= 0 {
		limits.ShutdownWait = defaults.ShutdownWait
	}
	if limits.KillWait <= 0 {
		limits.KillWait = defaults.KillWait
	}
	limits.Router = router.DefaultLimitsIfZero(limits.Router)
	return limits
}

// Config configures the child app-server process.
type Config struct {
	ExecutablePath string
	LaunchArgs     []string
	Cwd            string
	Env            map[string]string
	ConfigFlags    []string
	ClientName     string
	ClientTitle    string
	ClientVersion  string
	Approval       ApprovalHandler
	ApprovalPolicy ApprovalPolicy
	Limits         Limits
}

// Metadata is the initialize response payload subset the SDK relies on.
type Metadata struct {
	ProtocolVersion string `json:"protocolVersion,omitempty"`
	UserAgent       string `json:"userAgent"`
	PlatformFamily  string `json:"platformFamily,omitempty"`
	PlatformOs      string `json:"platformOs,omitempty"`
	ServerInfo      *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo,omitempty"`
}

type serverRequest struct {
	id     any
	method string
	params json.RawMessage
}

// Client is a synchronous JSON-RPC transport over `codex app-server --listen stdio://`.
type Client struct {
	cfg    Config
	limits Limits
	router *router.MessageRouter

	mu       sync.Mutex
	proc     *exec.Cmd
	stdin    io.WriteCloser
	stdout   io.ReadCloser
	stderr   io.ReadCloser
	started  bool
	closed   bool
	waitErr  error
	waitDone chan struct{}
	initOnce sync.Once
	initMeta *Metadata
	initErr  error

	writeMu   sync.Mutex
	requestID uint64

	stderrMu    sync.Mutex
	stderrLines []string
	stderrBytes int

	serverRequests chan serverRequest
	workersWG      sync.WaitGroup
	handlerCtx     context.Context
	handlerCancel  context.CancelFunc
	workerMu       sync.Mutex
	workerCount    int
}

// New constructs an unopened app-server client.
func New(cfg Config) *Client {
	limits := normalizeLimits(cfg.Limits)
	return &Client{
		cfg:            cfg,
		limits:         limits,
		router:         router.New(limits.Router),
		waitDone:       make(chan struct{}),
		serverRequests: make(chan serverRequest, limits.MaxServerRequestHandlers),
	}
}

// Start launches the child process once and performs initialize/initialized.
func (c *Client) Start(ctx context.Context) (*Metadata, error) {
	if err := c.startProcess(); err != nil {
		return nil, err
	}
	c.initOnce.Do(func() {
		c.initMeta, c.initErr = c.initialize(ctx)
	})
	if c.initErr != nil {
		_ = c.Close()
		return nil, c.initErr
	}
	return c.initMeta, nil
}

// Request sends one JSON-RPC request and waits for its response.
func (c *Client) Request(ctx context.Context, method string, params any) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.ensureUsable(); err != nil {
		return nil, err
	}
	id := fmt.Sprintf("req-%d", atomic.AddUint64(&c.requestID, 1))
	waiter, err := c.router.CreateResponseWaiter(id)
	if err != nil {
		return nil, err
	}
	msg := map[string]any{"id": id, "method": method}
	if params != nil {
		msg["params"] = params
	}
	if err := c.writeMessage(msg); err != nil {
		c.router.DiscardResponseWaiter(id)
		return nil, err
	}
	select {
	case <-ctx.Done():
		c.router.DiscardResponseWaiter(id)
		return nil, ctx.Err()
	case response := <-waiter:
		return response.Result, response.Err
	}
}

// Notify sends one JSON-RPC notification.
func (c *Client) Notify(_ context.Context, method string, params any) error {
	if err := c.ensureUsable(); err != nil {
		return err
	}
	msg := map[string]any{"method": method}
	if params != nil {
		msg["params"] = params
	}
	return c.writeMessage(msg)
}

// RegisterTurn starts routing notifications for one turn.
func (c *Client) RegisterTurn(turnID string) error { return c.router.RegisterTurn(turnID) }

// UnregisterTurn stops routing notifications for one turn.
func (c *Client) UnregisterTurn(turnID string) { c.router.UnregisterTurn(turnID) }

// NextTurn waits for the next turn-scoped notification payload.
func (c *Client) NextTurn(turnID string, ctxs ...context.Context) (json.RawMessage, error) {
	return c.router.NextTurn(turnID, ctxs...)
}

// RegisterLogin starts routing notifications for one login attempt.
func (c *Client) RegisterLogin(loginID string) error { return c.router.RegisterLogin(loginID) }

// RegisterGoal starts routing notifications for one thread-scoped goal operation.
func (c *Client) RegisterGoal(threadID string) error { return c.router.RegisterGoal(threadID) }

// UnregisterLogin stops routing notifications for one login attempt.
func (c *Client) UnregisterLogin(loginID string) { c.router.UnregisterLogin(loginID) }

// UnregisterGoal stops routing notifications for one thread-scoped goal operation.
func (c *Client) UnregisterGoal(threadID string) { c.router.UnregisterGoal(threadID) }

// NextLogin waits for the next login-scoped notification payload.
func (c *Client) NextLogin(loginID string, ctxs ...context.Context) (json.RawMessage, error) {
	return c.router.NextLogin(loginID, ctxs...)
}

// NextGoal waits for the next goal-scoped notification payload.
func (c *Client) NextGoal(threadID string, ctxs ...context.Context) (json.RawMessage, error) {
	return c.router.NextGoal(threadID, ctxs...)
}

// NextGlobal waits for the next unscoped notification payload.
func (c *Client) NextGlobal(ctxs ...context.Context) (json.RawMessage, error) {
	return c.router.NextGlobal(ctxs...)
}

// Close shuts down the child process and wakes blocked waiters.
func (c *Client) Close() error {
	c.mu.Lock()
	if !c.started {
		c.mu.Unlock()
		return nil
	}
	if c.closed {
		waitDone := c.waitDone
		c.mu.Unlock()
		<-waitDone
		return c.Wait()
	}
	c.closed = true
	cmd := c.proc
	stdin := c.stdin
	waitDone := c.waitDone
	shutdownWait := c.limits.ShutdownWait
	killWait := c.limits.KillWait
	c.mu.Unlock()

	if stdin != nil {
		_ = stdin.Close()
	}
	if c.handlerCancel != nil {
		c.handlerCancel()
	}
	select {
	case <-waitDone:
	case <-time.After(shutdownWait):
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Signal(os.Interrupt)
		}
	}
	select {
	case <-waitDone:
	case <-time.After(killWait):
		if cmd != nil && cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
	}
	return c.Wait()
}

// Wait blocks for process exit and returns the terminal process error.
func (c *Client) Wait() error {
	<-c.waitDone
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.waitErr
}

// StderrTail returns the bounded stderr ring as one newline-joined string.
func (c *Client) StderrTail() string {
	c.stderrMu.Lock()
	defer c.stderrMu.Unlock()
	return redactSecrets(strings.Join(c.stderrLines, "\n"))
}

func (c *Client) startProcess() error {
	c.mu.Lock()
	if c.started {
		c.mu.Unlock()
		return nil
	}
	args, err := c.launchArgs()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	cmd := exec.CommandContext(context.Background(), args[0], args[1:]...)
	cmd.Dir = c.cfg.Cwd
	cmd.Env = buildEnv(c.cfg.Env)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		c.mu.Unlock()
		return err
	}
	if err := cmd.Start(); err != nil {
		c.mu.Unlock()
		return err
	}

	c.proc = cmd
	c.stdin = stdin
	c.stdout = stdout
	c.stderr = stderr
	c.started = true
	c.closed = false
	c.waitErr = nil
	c.waitDone = make(chan struct{})
	c.serverRequests = make(chan serverRequest, c.limits.MaxServerRequestHandlers)
	c.handlerCtx, c.handlerCancel = context.WithCancel(context.Background())
	c.mu.Unlock()
	go c.readLoop()
	go c.drainStderr()
	go c.waitLoop()
	return nil
}

func (c *Client) launchArgs() ([]string, error) {
	if len(c.cfg.LaunchArgs) != 0 {
		return append([]string(nil), c.cfg.LaunchArgs...), nil
	}
	if c.cfg.ExecutablePath == "" {
		return nil, errors.New("appserver: executable path is required")
	}
	args := []string{c.cfg.ExecutablePath}
	for _, flag := range c.cfg.ConfigFlags {
		args = append(args, "--config", flag)
	}
	args = append(args, "app-server", "--listen", "stdio://")
	return args, nil
}

func (c *Client) initialize(ctx context.Context) (*Metadata, error) {
	payload, err := c.Request(ctx, "initialize", map[string]any{
		"clientInfo": map[string]any{
			"name":    orDefault(c.cfg.ClientName, "codex_go_sdk"),
			"title":   orDefault(c.cfg.ClientTitle, "Codex Go SDK"),
			"version": orDefault(c.cfg.ClientVersion, "dev"),
		},
		"capabilities": map[string]any{
			"experimentalApi": true,
		},
	})
	if err != nil {
		return nil, err
	}
	var metadata Metadata
	if err := json.Unmarshal(payload, &metadata); err != nil {
		return nil, err
	}
	if err := normalizeInitializeMetadata(&metadata); err != nil {
		return nil, errors.New("appserver: initialize returned incomplete metadata")
	}
	if err := c.Notify(context.Background(), "initialized", nil); err != nil {
		return nil, err
	}
	return &metadata, nil
}

func (c *Client) ensureUsable() error {
	c.mu.Lock()
	defer c.mu.Unlock()
	switch {
	case !c.started:
		return ErrTransportClosed
	case c.closed:
		return ErrTransportClosed
	default:
		return nil
	}
}

func (c *Client) writeMessage(msg map[string]any) error {
	encoded, err := json.Marshal(msg)
	if err != nil {
		return err
	}
	if len(encoded) > c.limits.MaxLineBytes {
		return ErrLimitExceeded
	}

	c.writeMu.Lock()
	defer c.writeMu.Unlock()

	c.mu.Lock()
	stdin := c.stdin
	c.mu.Unlock()
	if stdin == nil {
		return ErrTransportClosed
	}
	if _, err := stdin.Write(append(encoded, '\n')); err != nil {
		return fmt.Errorf("%w: %v", ErrTransportClosed, err)
	}
	return nil
}

func (c *Client) readLoop() {
	scanner := bufio.NewScanner(c.stdout)
	initial := minInt(64*1024, c.limits.MaxLineBytes)
	if initial < 1024 {
		initial = c.limits.MaxLineBytes
	}
	scanner.Buffer(make([]byte, 0, initial), c.limits.MaxLineBytes)
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		var envelope struct {
			ID     any             `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
			Result json.RawMessage `json:"result"`
			Error  *struct {
				Code    int             `json:"code"`
				Message string          `json:"message"`
				Data    json.RawMessage `json:"data"`
			} `json:"error"`
		}
		if err := json.Unmarshal(line, &envelope); err != nil {
			c.fail(fmt.Errorf("%w: invalid json: %v", ErrTransportClosed, err))
			return
		}
		switch {
		case envelope.Method != "" && envelope.ID != nil:
			req := serverRequest{
				id:     envelope.ID,
				method: envelope.Method,
				params: append(json.RawMessage(nil), envelope.Params...),
			}
			select {
			case c.serverRequests <- req:
				c.ensureServerRequestWorkers()
			default:
				c.fail(fmt.Errorf("%w: %w: server request handler queue limit exceeded", ErrTransportClosed, ErrLimitExceeded))
				return
			}
		case envelope.Method != "":
			if err := c.router.RouteNotification(envelope.Method, envelope.Params); err != nil {
				c.fail(fmt.Errorf("%w: %w: %v", ErrTransportClosed, ErrLimitExceeded, err))
				return
			}
		case envelope.ID != nil:
			id := fmt.Sprint(envelope.ID)
			if envelope.Error != nil {
				c.router.RouteResponse(id, nil, &RPCError{
					Code:    envelope.Error.Code,
					Message: envelope.Error.Message,
					Data:    append(json.RawMessage(nil), envelope.Error.Data...),
				})
				continue
			}
			c.router.RouteResponse(id, envelope.Result, nil)
		}
	}
	if err := scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			c.fail(fmt.Errorf("%w: %w: %v", ErrTransportClosed, ErrLimitExceeded, err))
			return
		}
		c.fail(fmt.Errorf("%w: %v", ErrTransportClosed, err))
	}
}

func (c *Client) serverRequestWorker() {
	defer c.workersWG.Done()
	defer func() {
		c.workerMu.Lock()
		c.workerCount--
		c.workerMu.Unlock()
	}()
	for req := range c.serverRequests {
		c.handleServerRequest(c.handlerCtx, req.id, req.method, req.params)
	}
}

func (c *Client) ensureServerRequestWorkers() {
	c.workerMu.Lock()
	defer c.workerMu.Unlock()
	for c.workerCount < c.limits.MaxServerRequestHandlers && c.workerCount <= len(c.serverRequests) {
		c.workerCount++
		c.workersWG.Add(1)
		go c.serverRequestWorker()
	}
}

func (c *Client) handleServerRequest(ctx context.Context, id any, method string, params json.RawMessage) {
	var (
		result json.RawMessage
		err    error
	)
	if c.cfg.Approval != nil {
		result, err = c.cfg.Approval(ctx, method, append(json.RawMessage(nil), params...))
	} else {
		result, err = defaultApprovalHandler(method, c.cfg.ApprovalPolicy)
	}
	if err != nil {
		_ = c.writeMessage(map[string]any{
			"id": id,
			"error": map[string]any{
				"code":    -32000,
				"message": err.Error(),
			},
		})
		return
	}
	if len(result) == 0 {
		result = json.RawMessage(`{}`)
	}
	var decoded any
	if err := json.Unmarshal(result, &decoded); err != nil {
		decoded = map[string]any{}
	}
	_ = c.writeMessage(map[string]any{"id": id, "result": decoded})
}

func (c *Client) drainStderr() {
	scanner := bufio.NewScanner(c.stderr)
	initial := minInt(16*1024, c.limits.MaxStderrBytes)
	if initial < 1024 {
		initial = minInt(1024, c.limits.MaxStderrBytes)
	}
	scanner.Buffer(make([]byte, 0, initial), c.limits.MaxStderrBytes)
	for scanner.Scan() {
		line := redactSecrets(scanner.Text())
		c.stderrMu.Lock()
		c.stderrLines = append(c.stderrLines, line)
		c.stderrBytes += len(line)
		for len(c.stderrLines) > c.limits.MaxStderrLines || c.stderrBytes > c.limits.MaxStderrBytes {
			if len(c.stderrLines) == 0 {
				break
			}
			c.stderrBytes -= len(c.stderrLines[0])
			c.stderrLines = c.stderrLines[1:]
		}
		c.stderrMu.Unlock()
	}
}

func (c *Client) waitLoop() {
	c.mu.Lock()
	cmd := c.proc
	c.mu.Unlock()

	var err error
	if cmd != nil {
		err = cmd.Wait()
	}
	if c.handlerCancel != nil {
		c.handlerCancel()
	}
	close(c.serverRequests)
	c.workersWG.Wait()

	c.mu.Lock()
	if err != nil {
		if tail := c.StderrTail(); tail != "" {
			err = fmt.Errorf("%w: %s", err, tail)
		}
	}
	c.waitErr = err
	waitDone := c.waitDone
	c.mu.Unlock()

	if err != nil {
		c.fail(fmt.Errorf("%w: %v", ErrTransportClosed, err))
	} else {
		c.fail(ErrTransportClosed)
	}
	close(waitDone)
}

func (c *Client) fail(err error) {
	c.router.FailAll(err)
}

func buildEnv(override map[string]string) []string {
	env := make(map[string]string)
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	for key, value := range override {
		env[key] = value
	}
	out := make([]string, 0, len(env))
	for key, value := range env {
		out = append(out, key+"="+value)
	}
	return out
}

func defaultApprovalHandler(method string, policy ApprovalPolicy) (json.RawMessage, error) {
	switch method {
	case "item/commandExecution/requestApproval":
		if policy == ApprovalPolicyAutoReview {
			return json.RawMessage(`{"decision":"accept"}`), nil
		}
		return json.RawMessage(`{"decision":"cancel"}`), nil
	case "item/fileChange/requestApproval":
		if policy == ApprovalPolicyAutoReview {
			return json.RawMessage(`{"decision":"accept"}`), nil
		}
		return json.RawMessage(`{"decision":"cancel"}`), nil
	case "item/permissions/requestApproval":
		return nil, errors.New("appserver: permissions request denied")
	case "applyPatchApproval":
		if policy == ApprovalPolicyAutoReview {
			return json.RawMessage(`{"decision":"approved"}`), nil
		}
		return json.RawMessage(`{"decision":"abort"}`), nil
	case "execCommandApproval":
		if policy == ApprovalPolicyAutoReview {
			return json.RawMessage(`{"decision":"approved"}`), nil
		}
		return json.RawMessage(`{"decision":"abort"}`), nil
	case "item/tool/requestUserInput":
		return nil, errors.New("appserver: tool user input denied")
	case "mcpServer/elicitation/request":
		return json.RawMessage(`{"action":"cancel","content":null}`), nil
	case "item/tool/call":
		return json.RawMessage(`{"success":false,"contentItems":[]}`), nil
	case "account/chatgptAuthTokens/refresh":
		return nil, errors.New("appserver: chatgpt token refresh unsupported")
	case "attestation/generate":
		return nil, errors.New("appserver: attestation generation unsupported")
	default:
		return nil, fmt.Errorf("appserver: unsupported server request %q", method)
	}
}

func redactSecrets(input string) string {
	out := redactTokenLike(input, "sk-", "[REDACTED_API_KEY]")
	for _, key := range []string{"authorization", "access_token", "access token", "refresh_token", "refresh token", "password", "userCode", "user code", "authUrl", "auth url", "verificationUrl", "verification url"} {
		out = redactKeyValue(out, key)
	}
	for _, key := range []string{"authUrl", "verificationUrl", "userCode"} {
		out = redactJSONValue(out, key)
	}
	return out
}

func DefaultHandledServerRequests() map[string]struct{} {
	out := make(map[string]struct{}, len(defaultHandledServerRequests))
	for method := range defaultHandledServerRequests {
		out[method] = struct{}{}
	}
	return out
}

func ServerRequestRegistry() map[string]protocol.MethodSpec {
	return protocol.ServerRequestRegistry
}

func orDefault(value string, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func normalizeInitializeMetadata(metadata *Metadata) error {
	if metadata == nil {
		return errors.New("appserver: initialize returned incomplete metadata")
	}
	userAgent := strings.TrimSpace(metadata.UserAgent)
	if userAgent == "" {
		return errors.New("appserver: initialize returned incomplete metadata")
	}
	metadata.UserAgent = userAgent
	if metadata.ServerInfo != nil {
		metadata.ServerInfo.Name = strings.TrimSpace(metadata.ServerInfo.Name)
		metadata.ServerInfo.Version = strings.TrimSpace(metadata.ServerInfo.Version)
	}
	if metadata.ServerInfo == nil || metadata.ServerInfo.Name == "" || metadata.ServerInfo.Version == "" {
		name, version := splitUserAgent(userAgent)
		if metadata.ServerInfo == nil {
			metadata.ServerInfo = &struct {
				Name    string `json:"name"`
				Version string `json:"version"`
			}{}
		}
		if metadata.ServerInfo.Name == "" {
			metadata.ServerInfo.Name = name
		}
		if metadata.ServerInfo.Version == "" {
			metadata.ServerInfo.Version = version
		}
	}
	if metadata.ServerInfo == nil {
		return errors.New("appserver: initialize returned incomplete metadata")
	}
	if metadata.ServerInfo.Name == "" || metadata.ServerInfo.Version == "" {
		return errors.New("appserver: initialize returned incomplete metadata")
	}
	return nil
}

func splitUserAgent(userAgent string) (string, string) {
	raw := strings.TrimSpace(userAgent)
	if raw == "" {
		return "", ""
	}
	if slash := strings.Index(raw, "/"); slash >= 0 {
		name := strings.TrimSpace(raw[:slash])
		version := strings.TrimSpace(raw[slash+1:])
		return name, version
	}
	if fields := strings.Fields(raw); len(fields) >= 2 {
		return strings.TrimSpace(fields[0]), strings.TrimSpace(strings.Join(fields[1:], " "))
	}
	return raw, ""
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func redactJSONValue(input string, key string) string {
	marker := `"` + key + `":"`
	offset := 0
	for {
		idx := strings.Index(strings.ToLower(input[offset:]), strings.ToLower(marker))
		if idx < 0 {
			return input
		}
		idx += offset
		start := idx + len(marker)
		end := start
		for end < len(input) && input[end] != '"' {
			end++
		}
		input = input[:start] + "[REDACTED]" + input[end:]
		offset = start + len("[REDACTED]")
	}
}

func redactKeyValue(input string, key string) string {
	search := strings.ToLower(key)
	offset := 0
	for {
		idx := strings.Index(strings.ToLower(input[offset:]), search)
		if idx < 0 {
			return input
		}
		idx += offset
		pos := idx + len(search)
		for pos < len(input) && (input[pos] == ' ' || input[pos] == ':' || input[pos] == '=' || input[pos] == '"') {
			pos++
		}
		end := pos
		for end < len(input) && input[end] != ' ' && input[end] != '\n' && input[end] != '\r' && input[end] != '"' {
			end++
		}
		input = input[:pos] + "[REDACTED]" + input[end:]
		offset = pos + len("[REDACTED]")
	}
}

func redactTokenLike(input string, prefix string, replacement string) string {
	for {
		idx := strings.Index(input, prefix)
		if idx < 0 {
			return input
		}
		end := idx + len(prefix)
		for end < len(input) {
			ch := input[end]
			if !(ch == '-' || ch == '_' || ch >= '0' && ch <= '9' || ch >= 'A' && ch <= 'Z' || ch >= 'a' && ch <= 'z') {
				break
			}
			end++
		}
		input = input[:idx] + replacement + input[end:]
	}
}
