package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"github.com/godeps/codex-sdk-go/internal/appserver"
	"github.com/godeps/codex-sdk-go/internal/runtimebin"
)

type threadStartLock struct {
	mu    sync.Mutex
	users int
}

type sdkClient struct {
	options        CodexOptions
	resolveRuntime func(CodexOptions) (resolvedExecutable, error)

	mu       sync.Mutex
	client   *appserver.Client
	metadata *appserver.Metadata
}

type resolvedExecutable struct {
	path     string
	pathDirs []string
}

// Client is the context-first root API backed by the long-lived app-server runtime.
type Client struct {
	transport *sdkClient

	lockMu      sync.Mutex
	threadLocks map[string]*threadStartLock

	goalMu sync.Mutex
	goals  map[string]*goalOperation
}

// NewClient starts the app-server immediately and returns the initialized root client.
func NewClient(ctx context.Context, opts ...Option) (*Client, error) {
	options, err := resolveClientOptions(opts...)
	if err != nil {
		return nil, err
	}
	transport := newManagedSDKClient(options)
	if err := transport.ensureStarted(ctx); err != nil {
		return nil, err
	}
	return &Client{
		transport:   transport,
		threadLocks: make(map[string]*threadStartLock),
		goals:       make(map[string]*goalOperation),
	}, nil
}

func newSDKClient(options CodexOptions) *sdkClient {
	return &sdkClient{
		options: options,
		resolveRuntime: func(options CodexOptions) (resolvedExecutable, error) {
			return resolvedExecutable{path: findCodexPathWithOverride(options.CodexPathOverride)}, nil
		},
	}
}

func newManagedSDKClient(options CodexOptions) *sdkClient {
	return &sdkClient{
		options:        options,
		resolveRuntime: resolveManagedExecutable,
	}
}

func (c *sdkClient) ensureStarted(ctx context.Context) error {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()

	if client == nil {
		configFlags, err := flattenConfigOverrides(c.options.Config)
		if err != nil {
			return err
		}
		resolved, err := c.resolveRuntime(c.options)
		if err != nil {
			return err
		}
		env := buildEnvMap(c.options.Env, c.options.BaseURL, c.options.APIKey)
		if len(resolved.pathDirs) > 0 {
			runtimebin.PrependPathDirs(env, resolved.pathDirs)
		}
		client = appserver.New(appserver.Config{
			ExecutablePath: resolved.path,
			Env:            env,
			ConfigFlags:    configFlags,
			ClientName:     goSDKOriginator,
			ClientVersion:  "dev",
		})
		c.mu.Lock()
		if c.client == nil {
			c.client = client
		} else {
			client = c.client
		}
		c.mu.Unlock()
	}

	metadata, err := client.Start(ctx)
	if err != nil {
		return err
	}
	c.mu.Lock()
	c.metadata = metadata
	c.mu.Unlock()
	return nil
}

func (c *sdkClient) request(ctx context.Context, method string, params any, out any) error {
	if err := c.ensureStarted(ctx); err != nil {
		return err
	}
	raw, err := c.client.Request(ctx, method, params)
	if err != nil {
		return translateAppServerError(err)
	}
	if out == nil {
		return nil
	}
	if err := json.Unmarshal(raw, out); err != nil {
		return err
	}
	return nil
}

func (c *sdkClient) registerTurn(turnID string) error {
	if err := c.ensureStarted(context.Background()); err != nil {
		return err
	}
	c.client.RegisterTurn(turnID)
	return nil
}

func (c *sdkClient) unregisterTurn(turnID string) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client != nil {
		client.UnregisterTurn(turnID)
	}
}

func (c *sdkClient) nextTurn(ctx context.Context, turnID string) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.ensureStarted(ctx); err != nil {
		return nil, err
	}
	raw, err := c.client.NextTurn(turnID, ctx)
	return raw, translateAppServerError(err)
}

func (c *sdkClient) registerLogin(loginID string) error {
	if err := c.ensureStarted(context.Background()); err != nil {
		return err
	}
	return c.client.RegisterLogin(loginID)
}

func (c *sdkClient) registerGoal(threadID string) error {
	if err := c.ensureStarted(context.Background()); err != nil {
		return err
	}
	return c.client.RegisterGoal(threadID)
}

func (c *sdkClient) unregisterLogin(loginID string) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client != nil {
		client.UnregisterLogin(loginID)
	}
}

func (c *sdkClient) unregisterGoal(threadID string) {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client != nil {
		client.UnregisterGoal(threadID)
	}
}

func (c *sdkClient) nextLogin(ctx context.Context, loginID string) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.ensureStarted(ctx); err != nil {
		return nil, err
	}
	raw, err := c.client.NextLogin(loginID, ctx)
	return raw, translateAppServerError(err)
}

func (c *sdkClient) nextGoal(ctx context.Context, threadID string) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if err := c.ensureStarted(ctx); err != nil {
		return nil, err
	}
	raw, err := c.client.NextGoal(threadID, ctx)
	return raw, translateAppServerError(err)
}

func (c *sdkClient) metadataSnapshot() *Metadata {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.metadata == nil {
		return nil
	}
	return &Metadata{
		ProtocolVersion: c.metadata.ProtocolVersion,
		UserAgent:       c.metadata.UserAgent,
		ServerInfo: func() *ServerInfo {
			if c.metadata.ServerInfo == nil {
				return nil
			}
			return &ServerInfo{
				Name:    c.metadata.ServerInfo.Name,
				Version: c.metadata.ServerInfo.Version,
			}
		}(),
	}
}

func (c *sdkClient) close() error {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return nil
	}
	return translateAppServerError(client.Close())
}

func (c *sdkClient) wait() error {
	c.mu.Lock()
	client := c.client
	c.mu.Unlock()
	if client == nil {
		return nil
	}
	return translateAppServerError(client.Wait())
}

func (c *Client) threadLock(threadID string) *threadStartLock {
	c.lockMu.Lock()
	defer c.lockMu.Unlock()
	entry := c.threadLocks[threadID]
	if entry == nil {
		entry = &threadStartLock{}
		c.threadLocks[threadID] = entry
	}
	entry.users++
	return entry
}

func (c *Client) releaseThreadLock(threadID string, entry *threadStartLock) {
	c.lockMu.Lock()
	defer c.lockMu.Unlock()
	entry.users--
	if entry.users == 0 {
		delete(c.threadLocks, threadID)
	}
}

func (c *Client) withThreadLock(threadID string, fn func() error) error {
	entry := c.threadLock(threadID)
	defer c.releaseThreadLock(threadID, entry)
	entry.mu.Lock()
	defer entry.mu.Unlock()
	return fn()
}

// Metadata reports the initialize result captured when the client started.
func (c *Client) Metadata() *Metadata {
	if c == nil || c.transport == nil {
		return nil
	}
	return c.transport.metadataSnapshot()
}

// Close stops the shared app-server process.
func (c *Client) Close() error {
	if c == nil || c.transport == nil {
		return nil
	}
	return c.transport.close()
}

// CloseContext stops the shared app-server process, respecting caller cancellation while waiting.
func (c *Client) CloseContext(ctx context.Context) error {
	if c == nil || c.transport == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("codex: nil context")
	}
	done := make(chan error, 1)
	go func() { done <- c.transport.close() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// Wait blocks until the underlying app-server exits.
func (c *Client) Wait() error {
	if c == nil || c.transport == nil {
		return nil
	}
	return c.transport.wait()
}

// WaitContext blocks until the underlying app-server exits or ctx is canceled.
func (c *Client) WaitContext(ctx context.Context) error {
	if c == nil || c.transport == nil {
		return nil
	}
	if ctx == nil {
		return errors.New("codex: nil context")
	}
	done := make(chan error, 1)
	go func() { done <- c.transport.wait() }()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case err := <-done:
		return err
	}
}

// ListModels returns one page from model/list.
func (c *Client) ListModels(ctx context.Context, includeHidden bool) (*ModelPage, error) {
	var raw struct {
		Data       []map[string]any `json:"data"`
		NextCursor string           `json:"nextCursor"`
	}
	if err := c.transport.request(ctx, "model/list", map[string]any{
		"includeHidden": includeHidden,
	}, &raw); err != nil {
		return nil, err
	}
	page := &ModelPage{NextCursor: raw.NextCursor}
	for _, model := range raw.Data {
		page.Data = append(page.Data, decodeModelInfo(model))
	}
	return page, nil
}

// StartThread creates a new persisted thread immediately.
func (c *Client) StartThread(ctx context.Context, options ThreadOptions) (*Thread, error) {
	payload := threadPayload(options)
	var response struct {
		Thread map[string]any `json:"thread"`
	}
	if err := c.transport.request(ctx, "thread/start", payload, &response); err != nil {
		return nil, err
	}
	record := decodeThreadRecord(response.Thread)
	return newClientThread(c, options, record.ID, true), nil
}

// ResumeThread resumes an existing thread immediately.
func (c *Client) ResumeThread(ctx context.Context, threadID string, options ThreadOptions) (*Thread, error) {
	payload := threadPayload(options)
	payload["threadId"] = threadID
	var response struct {
		Thread map[string]any `json:"thread"`
	}
	if err := c.transport.request(ctx, "thread/resume", payload, &response); err != nil {
		return nil, err
	}
	record := decodeThreadRecord(response.Thread)
	return newClientThread(c, options, record.ID, true), nil
}

// ReadThread returns the current persisted thread snapshot.
func (c *Client) ReadThread(ctx context.Context, threadID string, includeTurns bool) (*ThreadRecord, error) {
	var response struct {
		Thread map[string]any `json:"thread"`
	}
	if err := c.transport.request(ctx, "thread/read", map[string]any{
		"threadId":     threadID,
		"includeTurns": includeTurns,
	}, &response); err != nil {
		return nil, err
	}
	record := decodeThreadRecord(response.Thread)
	return &record, nil
}

// ListThreads returns one page from thread/list.
func (c *Client) ListThreads(ctx context.Context, options ThreadListOptions) (*ThreadPage, error) {
	payload := map[string]any{}
	if options.Archived != nil {
		payload["archived"] = *options.Archived
	}
	if options.Cursor != "" {
		payload["cursor"] = options.Cursor
	}
	if len(options.CWD) > 0 {
		payload["cwd"] = options.CWD
	}
	if options.Limit > 0 {
		payload["limit"] = options.Limit
	}
	if len(options.ModelProviders) > 0 {
		payload["modelProviders"] = append([]string(nil), options.ModelProviders...)
	}
	if options.SearchTerm != "" {
		payload["searchTerm"] = options.SearchTerm
	}
	if options.SortDirection != "" {
		payload["sortDirection"] = options.SortDirection
	}
	if options.SortKey != "" {
		payload["sortKey"] = options.SortKey
	}
	if len(options.SourceKinds) > 0 {
		payload["sourceKinds"] = append([]string(nil), options.SourceKinds...)
	}
	if options.UseStateDBOnly != nil {
		payload["useStateDbOnly"] = *options.UseStateDBOnly
	}
	var response struct {
		Data            []map[string]any `json:"data"`
		NextCursor      string           `json:"nextCursor"`
		BackwardsCursor string           `json:"backwardsCursor"`
	}
	if err := c.transport.request(ctx, "thread/list", payload, &response); err != nil {
		return nil, err
	}
	page := &ThreadPage{NextCursor: response.NextCursor, BackwardsCursor: response.BackwardsCursor}
	for _, item := range response.Data {
		page.Data = append(page.Data, decodeThreadRecord(item))
	}
	return page, nil
}

// ForkThread creates a new thread from an existing one.
func (c *Client) ForkThread(ctx context.Context, threadID string, options ThreadOptions) (*Thread, error) {
	payload := threadPayload(options)
	payload["threadId"] = threadID
	var response struct {
		Thread map[string]any `json:"thread"`
	}
	if err := c.transport.request(ctx, "thread/fork", payload, &response); err != nil {
		return nil, err
	}
	record := decodeThreadRecord(response.Thread)
	return newClientThread(c, options, record.ID, true), nil
}

// ArchiveThread archives one thread.
func (c *Client) ArchiveThread(ctx context.Context, threadID string) error {
	return c.transport.request(ctx, "thread/archive", map[string]any{"threadId": threadID}, nil)
}

// UnarchiveThread restores one archived thread.
func (c *Client) UnarchiveThread(ctx context.Context, threadID string) error {
	return c.transport.request(ctx, "thread/unarchive", map[string]any{"threadId": threadID}, nil)
}

// SetThreadName sets the persisted thread name.
func (c *Client) SetThreadName(ctx context.Context, threadID string, name string) error {
	return c.transport.request(ctx, "thread/name/set", map[string]any{
		"threadId": threadID,
		"name":     name,
	}, nil)
}

// CompactThread requests compaction for one thread.
func (c *Client) CompactThread(ctx context.Context, threadID string) error {
	return c.transport.request(ctx, "thread/compact/start", map[string]any{
		"threadId": threadID,
	}, nil)
}

func (c *Client) startTurn(ctx context.Context, threadID string, input Input, options TurnOptions) (*TurnHandle, error) {
	var handle *TurnHandle
	err := c.withThreadLock(threadID, func() error {
		c.goalMu.Lock()
		if _, exists := c.goals[threadID]; exists {
			c.goalMu.Unlock()
			return &InvalidRequestError{RPCError: &RPCError{Code: -32600, Message: "thread has an active goal operation: " + threadID}}
		}
		c.goalMu.Unlock()

		payload, err := buildTurnPayload(threadID, input, options)
		if err != nil {
			return err
		}
		var response struct {
			Turn turnWire `json:"turn"`
		}
		if err := c.transport.request(ctx, "turn/start", payload, &response); err != nil {
			return err
		}
		if response.Turn.ID == "" {
			return io.ErrUnexpectedEOF
		}
		if err := c.transport.registerTurn(response.Turn.ID); err != nil {
			return err
		}
		handle = &TurnHandle{
			client:   c,
			threadID: threadID,
			id:       response.Turn.ID,
			started:  decodeTurnState(response.Turn),
		}
		return nil
	})
	return handle, err
}

func (c *Client) steerTurn(ctx context.Context, threadID string, turnID string, input Input) error {
	wireInput, err := normalizeAppServerInput(input)
	if err != nil {
		return err
	}
	return c.transport.request(ctx, "turn/steer", map[string]any{
		"threadId":       threadID,
		"expectedTurnId": turnID,
		"input":          wireInput,
	}, nil)
}

func (c *Client) interruptTurn(ctx context.Context, threadID string, turnID string) error {
	return c.transport.request(ctx, "turn/interrupt", map[string]any{
		"threadId": threadID,
		"turnId":   turnID,
	}, nil)
}

func (c *Client) waitForLogin(ctx context.Context, loginID string) (*LoginResult, error) {
	if err := c.transport.registerLogin(loginID); err != nil {
		return nil, err
	}
	defer c.transport.unregisterLogin(loginID)
	for {
		if ctx != nil && ctx.Err() != nil {
			return nil, ctx.Err()
		}
		raw, err := c.transport.nextLogin(ctx, loginID)
		if err != nil {
			return nil, err
		}
		method, params, err := splitNotification(raw)
		if err != nil {
			var payload struct {
				LoginID string         `json:"loginId"`
				Account map[string]any `json:"account"`
			}
			if decodeErr := json.Unmarshal(raw, &payload); decodeErr == nil && payload.LoginID != "" {
				return &LoginResult{LoginID: payload.LoginID, Account: decodeAccount(payload.Account)}, nil
			}
			return nil, err
		}
		if method != "account/login/completed" {
			continue
		}
		var payload struct {
			LoginID string         `json:"loginId"`
			Account map[string]any `json:"account"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, err
		}
		return &LoginResult{
			LoginID: payload.LoginID,
			Account: decodeAccount(payload.Account),
		}, nil
	}
}

func (c *Client) loginStart(ctx context.Context, payload map[string]any, out any) error {
	return c.transport.request(ctx, "account/login/start", payload, out)
}

func parseTurnIDMismatch(err error) string {
	var message string
	var invalid *InvalidRequestError
	if errors.As(err, &invalid) {
		message = invalid.Message
	} else {
		var rpcErr *RPCError
		if !errors.As(err, &rpcErr) {
			return ""
		}
		message = rpcErr.Message
	}
	prefix := "expected active turn id"
	if !strings.HasPrefix(message, prefix) {
		return ""
	}
	lastTick := strings.LastIndex(message, "`")
	if lastTick == -1 {
		marker := " but found "
		idx := strings.LastIndex(message, marker)
		if idx == -1 {
			return ""
		}
		return strings.Trim(strings.TrimSpace(message[idx+len(marker):]), "`'\"")
	}
	prevTick := strings.LastIndex(message[:lastTick], "`")
	if prevTick == -1 || prevTick+1 >= lastTick {
		return ""
	}
	return message[prevTick+1 : lastTick]
}

func resolveClientOptions(opts ...Option) (CodexOptions, error) {
	var options CodexOptions
	for _, option := range opts {
		if option == nil {
			continue
		}
		if err := option.applyCodexOption(&options); err != nil {
			return CodexOptions{}, err
		}
	}
	return options, nil
}

func resolveManagedExecutable(options CodexOptions) (resolvedExecutable, error) {
	env := options.Env
	if env == nil {
		env = nil
	}
	version := options.RuntimeVersion
	if version == "" {
		version = runtimebin.DefaultRuntimeVersion
	}
	resolved, err := runtimebin.ResolveRuntime(runtimebin.ResolveOptions{
		ExplicitBinary: options.CodexPathOverride,
		Env:            env,
		CacheRoot:      options.RuntimeCacheRoot,
		RuntimeVersion: version,
		AllowPATH:      options.AllowPATH,
	})
	if err != nil {
		return resolvedExecutable{}, err
	}
	return resolvedExecutable{path: resolved.BinaryPath, pathDirs: append([]string(nil), resolved.PathDirs...)}, nil
}

func unixSecondsTime(ts int64) *time.Time {
	if ts == 0 {
		return nil
	}
	value := time.Unix(ts, 0).UTC()
	return &value
}

func buildEnvMap(override map[string]string, baseURL string, apiKey string) map[string]string {
	env := map[string]string{}
	if override != nil {
		for key, value := range override {
			env[key] = value
		}
	} else {
		for _, entry := range buildEnv(nil, "", "") {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}
	if _, ok := env[internalOriginatorEnv]; !ok {
		env[internalOriginatorEnv] = goSDKOriginator
	}
	if baseURL != "" {
		env["OPENAI_BASE_URL"] = baseURL
	}
	if apiKey != "" {
		env["CODEX_API_KEY"] = apiKey
	}
	return env
}

func findCodexPathWithOverride(path string) string {
	if path != "" {
		return path
	}
	return findCodexPath()
}

func translateAppServerError(err error) error {
	if err == nil {
		return nil
	}
	switch {
	case strings.Contains(err.Error(), appserver.ErrTransportClosed.Error()):
		return fmt.Errorf("%w: %v", ErrTransportClosed, err)
	case strings.Contains(err.Error(), appserver.ErrLimitExceeded.Error()):
		return fmt.Errorf("%w: %v", ErrLimitExceeded, err)
	default:
		var rpcErr *appserver.RPCError
		if errors.As(err, &rpcErr) {
			return &RPCError{Code: rpcErr.Code, Message: rpcErr.Message, Data: rpcErr.Data}
		}
		return err
	}
}
