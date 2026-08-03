package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

// Turn represents the v0.1 compatibility result returned by Thread.Run.
type Turn struct {
	Items         []ThreadItem
	FinalResponse string
	Usage         *Usage
}

// StreamedTurn is the v0.1 compatibility streaming adapter returned by RunStreamed.
type StreamedTurn struct {
	Events <-chan ThreadEvent
	Done   <-chan error
}

// TurnHandle controls one live app-server turn.
type TurnHandle struct {
	client   *Client
	thread   *Thread
	threadID string
	id       string
	started  TurnState

	mu       sync.Mutex
	consumed bool
}

// Thread represents a conversation with the agent.
type Thread struct {
	exec    *CodexExec
	options CodexOptions
	codex   *Codex
	client  *Client

	mu            sync.RWMutex
	id            string
	threadOptions ThreadOptions
	prepared      bool
}

// ID returns the current thread identifier.
func (t *Thread) ID() string {
	if t == nil {
		return ""
	}
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.id
}

func (t *Thread) setID(id string) {
	t.mu.Lock()
	t.id = id
	t.mu.Unlock()
}

func (t *Thread) markPrepared(id string) {
	t.mu.Lock()
	t.id = id
	t.prepared = true
	t.mu.Unlock()
}

func (t *Thread) isPrepared() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.prepared
}

func (t *Thread) rootClient() *Client {
	if t.client != nil {
		return t.client
	}
	if t.codex != nil {
		return t.codex.sharedClient()
	}
	return nil
}

// NewThread constructs a compatibility thread backed by `codex exec`.
func NewThread(exec *CodexExec, options CodexOptions, threadOptions ThreadOptions, id string) *Thread {
	return &Thread{
		exec:          exec,
		options:       options,
		id:            id,
		threadOptions: threadOptions,
	}
}

func newManagedThread(codex *Codex, threadOptions ThreadOptions, id string) *Thread {
	return &Thread{
		codex:         codex,
		id:            id,
		threadOptions: threadOptions,
	}
}

func newClientThread(client *Client, threadOptions ThreadOptions, id string, prepared bool) *Thread {
	return &Thread{
		client:        client,
		id:            id,
		threadOptions: threadOptions,
		prepared:      prepared,
	}
}

// RunStreamed sends input to the agent and streams v0.1 compatibility events.
func (t *Thread) RunStreamed(input Input, turnOptions TurnOptions) (*StreamedTurn, error) {
	if t.exec != nil {
		return t.runStreamedCompat(input, turnOptions)
	}
	handle, err := t.StartTurn(input, turnOptions)
	if err != nil {
		return nil, err
	}
	return handle.streamAdapter(turnOptions.Context)
}

// StartTurn starts one turn and returns a live handle.
func (t *Thread) StartTurn(input Input, turnOptions TurnOptions) (*TurnHandle, error) {
	ctx := turnOptions.Context
	if ctx == nil {
		ctx = context.Background()
	}
	return t.StartTurnContext(ctx, input, turnOptions)
}

// StartTurnContext starts one turn with an explicit context.
func (t *Thread) StartTurnContext(ctx context.Context, input Input, turnOptions TurnOptions) (*TurnHandle, error) {
	if t.exec != nil {
		return nil, ErrTransportClosed
	}
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}
	if err := t.ensurePrepared(ctx); err != nil {
		return nil, err
	}
	client := t.rootClient()
	if client == nil {
		return nil, ErrTransportClosed
	}
	return client.startTurn(ctx, t.ID(), input, turnOptions)
}

// Run sends input to the agent and returns the completed v0.1 result.
func (t *Thread) Run(input Input, turnOptions TurnOptions) (*Turn, error) {
	if t.exec != nil {
		return t.runCompat(input, turnOptions)
	}
	handle, err := t.StartTurn(input, turnOptions)
	if err != nil {
		return nil, err
	}
	result, err := handle.RunContext(turnOptions.ContextOrBackground())
	if err != nil {
		return nil, err
	}
	return &Turn{
		Items:         result.Items,
		FinalResponse: result.FinalResponse,
		Usage:         result.Usage,
	}, nil
}

// RunContext starts and collects a turn through the pull-stream API.
func (t *Thread) RunContext(ctx context.Context, input Input, turnOptions TurnOptions) (*TurnResult, error) {
	handle, err := t.StartTurnContext(ctx, input, turnOptions)
	if err != nil {
		return nil, err
	}
	return handle.RunContext(ctx)
}

func (t *Thread) runCompat(input Input, turnOptions TurnOptions) (*Turn, error) {
	streamed, err := t.runStreamedCompat(input, turnOptions)
	if err != nil {
		return nil, err
	}

	var items []ThreadItem
	var usage *Usage
	var turnFailure *ThreadError

	for event := range streamed.Events {
		switch event.Type {
		case "item.completed":
			if event.Item != nil {
				items = append(items, event.Item)
			}
		case "turn.completed", "thread.token_usage.updated":
			usage = event.Usage
		case "turn.failed":
			turnFailure = event.Error
			if event.Usage != nil {
				usage = event.Usage
			}
		}
	}
	if err := <-streamed.Done; err != nil {
		return nil, err
	}
	if turnFailure != nil {
		return nil, errors.New(turnFailure.Message)
	}
	return &Turn{
		Items:         items,
		FinalResponse: finalAssistantResponse(items),
		Usage:         usage,
	}, nil
}

func (t *Thread) runStreamedCompat(input Input, turnOptions TurnOptions) (*StreamedTurn, error) {
	schemaFile, err := createOutputSchemaFile(turnOptions.OutputSchema)
	if err != nil {
		return nil, err
	}

	prompt, images := normalizeCompatInput(input)
	options := t.threadOptions
	stream, err := t.exec.Run(CodexExecArgs{
		Input:                 prompt,
		BaseURL:               t.options.BaseURL,
		APIKey:                t.options.APIKey,
		Config:                t.options.Config,
		ThreadID:              t.ID(),
		Images:                images,
		Model:                 options.Model,
		SandboxMode:           options.SandboxMode,
		WorkingDirectory:      options.WorkingDirectory,
		SkipGitRepoCheck:      options.SkipGitRepoCheck,
		OutputSchemaFile:      schemaFile.SchemaPath,
		ModelReasoningEffort:  options.ModelReasoningEffort,
		Context:               turnOptions.Context,
		NetworkAccessEnabled:  options.NetworkAccessEnabled,
		WebSearchMode:         options.WebSearchMode,
		WebSearchEnabled:      options.WebSearchEnabled,
		ApprovalPolicy:        options.ApprovalPolicy,
		AdditionalDirectories: options.AdditionalDirectories,
	})
	if err != nil {
		_ = schemaFile.Cleanup()
		return nil, err
	}

	events := make(chan ThreadEvent)
	done := make(chan error, 1)
	go func() {
		defer close(events)
		defer close(done)

		var streamErr error
		for line := range stream.Lines {
			var event ThreadEvent
			if err := json.Unmarshal([]byte(line), &event); err != nil {
				if streamErr == nil {
					streamErr = fmt.Errorf("failed to parse item: %s: %w", line, err)
				}
				continue
			}
			if event.Type == "thread.started" && event.ThreadID != "" {
				t.setID(event.ThreadID)
			}
			events <- event
		}

		if streamErr == nil {
			streamErr = stream.Wait()
		} else {
			_ = stream.Wait()
		}
		if err := schemaFile.Cleanup(); err != nil && streamErr == nil {
			streamErr = err
		}
		done <- streamErr
	}()
	return &StreamedTurn{Events: events, Done: done}, nil
}

func (t *Thread) ensurePrepared(ctx context.Context) error {
	if t.isPrepared() {
		return nil
	}
	client := t.rootClient()
	if client == nil {
		return ErrTransportClosed
	}
	payload := threadPayload(t.threadOptions)
	method := "thread/start"
	if threadID := t.ID(); threadID != "" {
		method = "thread/resume"
		payload["threadId"] = threadID
	}
	var response struct {
		Thread struct {
			ID string `json:"id"`
		} `json:"thread"`
	}
	if err := client.transport.request(ctx, method, payload, &response); err != nil {
		return err
	}
	if response.Thread.ID == "" {
		return fmt.Errorf("%s returned an empty thread id", method)
	}
	t.markPrepared(response.Thread.ID)
	return nil
}

func threadPayload(options ThreadOptions) map[string]any {
	payload := map[string]any{}
	if options.Model != "" {
		payload["model"] = options.Model
	}
	if options.ModelProvider != "" {
		payload["modelProvider"] = options.ModelProvider
	}
	if options.SandboxMode != "" {
		payload["sandbox"] = string(options.SandboxMode)
	}
	if options.WorkingDirectory != "" {
		payload["cwd"] = options.WorkingDirectory
	}
	if options.BaseInstructions != "" {
		payload["baseInstructions"] = options.BaseInstructions
	}
	if options.DeveloperInstructions != "" {
		payload["developerInstructions"] = options.DeveloperInstructions
	}
	if options.Personality != "" {
		payload["personality"] = string(options.Personality)
	}
	if options.ServiceName != "" {
		payload["serviceName"] = options.ServiceName
	}
	if options.ServiceTier != "" {
		payload["serviceTier"] = options.ServiceTier
	}
	if options.SessionStartSource != "" {
		payload["sessionStartSource"] = options.SessionStartSource
	}
	if options.ThreadSource != nil {
		payload["threadSource"] = cloneMap(options.ThreadSource)
	}
	if options.Ephemeral != nil {
		payload["ephemeral"] = *options.Ephemeral
	}

	approvalPolicy, reviewer := resolveApprovalSettings(options.ApprovalPreset, options.ApprovalPolicy, true)
	if approvalPolicy != "" {
		payload["approvalPolicy"] = approvalPolicy
	}
	if reviewer != "" {
		payload["approvalsReviewer"] = reviewer
	}

	config := cloneMap(options.Config)
	if options.ModelReasoningEffort != "" {
		config["model_reasoning_effort"] = options.ModelReasoningEffort
	}
	if options.SkipGitRepoCheck {
		config["skip_git_repo_check"] = true
	}
	if options.NetworkAccessEnabled != nil {
		config["sandbox_workspace_write"] = map[string]any{
			"network_access": *options.NetworkAccessEnabled,
		}
	}
	switch {
	case options.WebSearchMode != "":
		config["web_search"] = options.WebSearchMode
	case options.WebSearchEnabled != nil && *options.WebSearchEnabled:
		config["web_search"] = WebSearchLive
	case options.WebSearchEnabled != nil:
		config["web_search"] = WebSearchDisabled
	}
	if len(options.AdditionalDirectories) > 0 {
		config["additional_directories"] = append([]string(nil), options.AdditionalDirectories...)
	}
	if len(config) > 0 {
		payload["config"] = config
	}
	return payload
}

func normalizeCompatInput(input Input) (string, []string) {
	if len(input.Items) == 0 {
		return input.Text, nil
	}
	promptParts := make([]string, 0, len(input.Items))
	images := make([]string, 0, len(input.Items))
	for _, item := range input.Items {
		switch item.Type {
		case UserInputText:
			promptParts = append(promptParts, item.Text)
		case UserInputLocalImage:
			images = append(images, item.Path)
		}
	}
	prompt := joinStrings(promptParts, "\n\n")
	if prompt != "" {
		prompt += "\n"
	}
	return prompt, images
}

func joinStrings(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	out := parts[0]
	for i := 1; i < len(parts); i++ {
		out += sep
		out += parts[i]
	}
	return out
}
