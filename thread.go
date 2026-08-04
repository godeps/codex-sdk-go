package codex

import (
	"context"
	"errors"
	"sync"
)

// TurnHandle controls one live app-server turn.
type TurnHandle struct {
	client   *Client
	threadID string
	id       string
	started  TurnState

	mu       sync.Mutex
	consumed bool
}

// Thread represents a conversation with the agent.
type Thread struct {
	client *Client
	id     string
}

// ID returns the current thread identifier.
func (t *Thread) ID() string {
	if t == nil {
		return ""
	}
	return t.id
}

func (t *Thread) rootClient() *Client {
	if t == nil {
		return nil
	}
	return t.client
}

func newClientThread(client *Client, id string) *Thread {
	return &Thread{
		client: client,
		id:     id,
	}
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
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}
	if err := t.ensureMaterialized(); err != nil {
		return nil, err
	}
	client := t.rootClient()
	if client == nil {
		return nil, ErrTransportClosed
	}
	return client.startTurn(ctx, t.ID(), input, turnOptions)
}

// RunContext starts and collects a turn through the pull-stream API.
func (t *Thread) RunContext(ctx context.Context, input Input, turnOptions TurnOptions) (*TurnResult, error) {
	handle, err := t.StartTurnContext(ctx, input, turnOptions)
	if err != nil {
		return nil, err
	}
	return handle.RunContext(ctx)
}

func (t *Thread) ensureMaterialized() error {
	if t.rootClient() == nil || t.ID() == "" {
		return ErrTransportClosed
	}
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
