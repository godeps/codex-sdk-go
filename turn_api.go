package codex

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"time"
)

type turnWire struct {
	ID          string            `json:"id"`
	Status      string            `json:"status"`
	Error       *ThreadError      `json:"error"`
	StartedAt   int64             `json:"startedAt"`
	CompletedAt int64             `json:"completedAt"`
	DurationMs  int64             `json:"durationMs"`
	Items       []json.RawMessage `json:"items"`
}

// TurnStream is the context-first pull stream over routed turn notifications.
type TurnStream struct {
	handle *TurnHandle

	latestUsage *Usage
	terminalErr error
	terminalEOF bool
	closed      bool
}

// ID returns the live turn identifier.
func (h *TurnHandle) ID() string {
	if h == nil {
		return ""
	}
	return h.id
}

// Stream starts a pull-based stream over this turn.
func (h *TurnHandle) Stream() (*TurnStream, error) {
	return h.StreamContext(context.Background())
}

// StreamContext starts a pull-based stream over this turn.
func (h *TurnHandle) StreamContext(_ context.Context) (*TurnStream, error) {
	if h == nil || h.client == nil {
		return nil, ErrTransportClosed
	}
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.consumed {
		return nil, errors.New("codex: turn stream already consumed")
	}
	h.consumed = true
	return &TurnStream{handle: h}, nil
}

// Next returns one routed turn notification.
func (s *TurnStream) Next(ctx context.Context) (*ThreadEvent, error) {
	if s == nil || s.handle == nil {
		return nil, ErrTransportClosed
	}
	if s.terminalErr != nil {
		return nil, s.terminalErr
	}
	if s.terminalEOF {
		return nil, io.EOF
	}
	if s.closed {
		return nil, ErrStreamClosed
	}
	if ctx != nil && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	raw, err := s.handle.client.transport.nextTurn(ctx, s.handle.id)
	if err != nil {
		s.terminalErr = err
		_ = s.Close()
		return nil, err
	}
	event, terminal, err := decodeTurnNotification(s.handle.threadID, raw, s.latestUsage)
	if err != nil {
		s.terminalErr = err
		_ = s.Close()
		return nil, err
	}
	if event != nil && event.Usage != nil {
		s.latestUsage = event.Usage
	}
	if terminal {
		s.terminalEOF = true
	}
	return event, nil
}

// Close stops consuming routed notifications and unregisters the turn route.
func (s *TurnStream) Close() error {
	if s == nil || s.handle == nil {
		return nil
	}
	if s.closed {
		return nil
	}
	s.closed = true
	s.handle.client.transport.unregisterTurn(s.handle.id)
	return nil
}

// RunContext waits for the handle's turn to finish and collects its final result.
func (h *TurnHandle) RunContext(ctx context.Context) (*TurnResult, error) {
	stream, err := h.StreamContext(ctx)
	if err != nil {
		return nil, err
	}
	defer stream.Close()
	return collectTurnResult(ctx, stream, h.id)
}

// Steer injects additional input into the active turn.
func (h *TurnHandle) Steer(input Input) error {
	return h.SteerContext(context.Background(), input)
}

// SteerContext injects additional input into the active turn.
func (h *TurnHandle) SteerContext(ctx context.Context, input Input) error {
	return h.client.steerTurn(ctx, h.threadID, h.id, input)
}

// Interrupt requests interruption of the active turn.
func (h *TurnHandle) Interrupt() error {
	return h.InterruptContext(context.Background())
}

// InterruptContext requests interruption of the active turn.
func (h *TurnHandle) InterruptContext(ctx context.Context) error {
	return h.client.interruptTurn(ctx, h.threadID, h.id)
}

func collectTurnResult(ctx context.Context, stream *TurnStream, turnID string) (*TurnResult, error) {
	var (
		completed *TurnState
		items     []ThreadItem
		usage     *Usage
		turnErr   *ThreadError
	)
	for {
		event, err := stream.Next(ctx)
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			return nil, err
		}
		switch event.Type {
		case "item.completed":
			if event.Item != nil {
				items = append(items, event.Item)
			}
		case "thread.token_usage.updated":
			usage = event.Usage
		case "turn.completed":
			completed = event.Turn
			if event.Usage != nil {
				usage = event.Usage
			}
		case "turn.failed":
			completed = event.Turn
			turnErr = event.Error
			if event.Usage != nil {
				usage = event.Usage
			}
		}
	}
	if completed == nil {
		return nil, errors.New("codex: turn completed event not received")
	}
	if completed.Status == TurnStatusFailed {
		if turnErr != nil && turnErr.Message != "" {
			return nil, errors.New(turnErr.Message)
		}
		return nil, fmt.Errorf("codex: turn %s failed", turnID)
	}
	return &TurnResult{
		ID:            completed.ID,
		Status:        completed.Status,
		Error:         turnErr,
		StartedAt:     completed.StartedAt,
		CompletedAt:   completed.CompletedAt,
		Duration:      completed.Duration,
		FinalResponse: finalAssistantResponse(items),
		Items:         items,
		Usage:         usage,
	}, nil
}

func buildTurnPayload(threadID string, input Input, options TurnOptions) (map[string]any, error) {
	wireInput, err := normalizeAppServerInput(input)
	if err != nil {
		return nil, err
	}
	payload := map[string]any{
		"threadId": threadID,
		"input":    wireInput,
	}
	approvalPolicy, reviewer := resolveApprovalSettings(options.ApprovalPreset, options.ApprovalPolicy, true)
	if approvalPolicy != "" {
		payload["approvalPolicy"] = approvalPolicy
	}
	if reviewer != "" {
		payload["approvalsReviewer"] = reviewer
	}
	if options.Model != "" {
		payload["model"] = options.Model
	}
	if options.WorkingDirectory != "" {
		payload["cwd"] = options.WorkingDirectory
	}
	if options.ReasoningEffort != "" {
		payload["reasoningEffort"] = options.ReasoningEffort
	}
	if options.Personality != "" {
		payload["personality"] = string(options.Personality)
	}
	if options.SandboxMode != "" {
		payload["sandboxPolicy"] = string(options.SandboxMode)
	}
	if options.ServiceTier != "" {
		payload["serviceTier"] = options.ServiceTier
	}
	if options.ReasoningSummary != "" {
		payload["reasoningSummary"] = string(options.ReasoningSummary)
	}
	if options.OutputSchema != nil {
		if !isJSONObject(options.OutputSchema) {
			return nil, errors.New("outputSchema must be a JSON object")
		}
		payload["outputSchema"] = options.OutputSchema
	}
	return payload, nil
}

func decodeTurnNotification(threadID string, raw json.RawMessage, latestUsage *Usage) (*ThreadEvent, bool, error) {
	method, params, err := splitNotification(raw)
	if err != nil {
		params = raw
		method = inferNotificationMethod(raw)
		if method == "" {
			return nil, false, err
		}
	}
	switch method {
	case "turn/started":
		var payload struct {
			ThreadID string   `json:"threadId"`
			Turn     turnWire `json:"turn"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, false, err
		}
		state := decodeTurnState(payload.Turn)
		return &ThreadEvent{
			Type:     "turn.started",
			Method:   method,
			ThreadID: orString(payload.ThreadID, threadID),
			TurnID:   state.ID,
			Turn:     &state,
			Raw:      append(json.RawMessage(nil), raw...),
		}, false, nil
	case "item/completed":
		var payload struct {
			ThreadID string          `json:"threadId"`
			TurnID   string          `json:"turnId"`
			Item     json.RawMessage `json:"item"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, false, err
		}
		item, err := parseThreadItem(payload.Item)
		if err != nil {
			return nil, false, err
		}
		return &ThreadEvent{
			Type:     "item.completed",
			Method:   method,
			ThreadID: orString(payload.ThreadID, threadID),
			TurnID:   payload.TurnID,
			Item:     item,
			Raw:      append(json.RawMessage(nil), raw...),
		}, false, nil
	case "thread/tokenUsage/updated":
		var payload struct {
			ThreadID   string `json:"threadId"`
			TurnID     string `json:"turnId"`
			TokenUsage struct {
				Last *Usage `json:"last"`
			} `json:"tokenUsage"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, false, err
		}
		return &ThreadEvent{
			Type:     "thread.token_usage.updated",
			Method:   method,
			ThreadID: orString(payload.ThreadID, threadID),
			TurnID:   payload.TurnID,
			Usage:    payload.TokenUsage.Last,
			Raw:      append(json.RawMessage(nil), raw...),
		}, false, nil
	case "turn/completed":
		var payload struct {
			ThreadID string   `json:"threadId"`
			Turn     turnWire `json:"turn"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			return nil, false, err
		}
		state := decodeTurnState(payload.Turn)
		eventType := "turn.completed"
		if state.Status == TurnStatusFailed {
			eventType = "turn.failed"
		}
		return &ThreadEvent{
			Type:     eventType,
			Method:   method,
			ThreadID: orString(payload.ThreadID, threadID),
			TurnID:   state.ID,
			Usage:    latestUsage,
			Error:    state.Error,
			Turn:     &state,
			Raw:      append(json.RawMessage(nil), raw...),
		}, true, nil
	default:
		return &ThreadEvent{
			Type:     method,
			Method:   method,
			ThreadID: threadID,
			Raw:      append(json.RawMessage(nil), raw...),
		}, false, nil
	}
}

func inferNotificationMethod(raw json.RawMessage) string {
	var payload struct {
		Item       json.RawMessage `json:"item"`
		TurnID     string          `json:"turnId"`
		TokenUsage json.RawMessage `json:"tokenUsage"`
		Turn       *struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	switch {
	case len(payload.Item) != 0 && payload.TurnID != "":
		return "item/completed"
	case len(payload.TokenUsage) != 0:
		return "thread/tokenUsage/updated"
	case payload.Turn != nil && payload.Turn.Status != "":
		if payload.Turn.Status == "completed" || payload.Turn.Status == "failed" || payload.Turn.Status == "interrupted" {
			return "turn/completed"
		}
		return "turn/started"
	case payload.Turn != nil && payload.Turn.ID != "":
		return "turn/started"
	default:
		return ""
	}
}

func decodeTurnState(turn turnWire) TurnState {
	return TurnState{
		ID:          turn.ID,
		Status:      TurnStatus(turn.Status),
		Error:       turn.Error,
		StartedAt:   unixSecondsTime(turn.StartedAt),
		CompletedAt: unixSecondsTime(turn.CompletedAt),
		Duration:    time.Duration(turn.DurationMs) * time.Millisecond,
	}
}

func finalAssistantResponse(items []ThreadItem) string {
	fallback := ""
	for i := len(items) - 1; i >= 0; i-- {
		message, ok := items[i].(*AgentMessageItem)
		if !ok {
			continue
		}
		if message.Phase == "final_answer" {
			return message.Text
		}
		if message.Phase == "" && fallback == "" {
			fallback = message.Text
		}
	}
	return fallback
}

func splitNotification(raw json.RawMessage) (string, json.RawMessage, error) {
	var envelope struct {
		Method string          `json:"method"`
		Params json.RawMessage `json:"params"`
	}
	if err := json.Unmarshal(raw, &envelope); err != nil {
		return "", nil, err
	}
	if envelope.Method == "" {
		return "", nil, errors.New("codex: notification missing method")
	}
	return envelope.Method, envelope.Params, nil
}

func buildNotificationEnvelope(method string, params json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"method": method,
		"params": json.RawMessage(append(json.RawMessage(nil), params...)),
	})
}

func notificationGoalTurnID(raw json.RawMessage) string {
	var payload struct {
		TurnID string `json:"turnId"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return payload.TurnID
}

func notificationTurnIDLocal(raw json.RawMessage) string {
	var payload struct {
		TurnID string `json:"turnId"`
		Turn   *struct {
			ID string `json:"id"`
		} `json:"turn"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	if payload.TurnID != "" {
		return payload.TurnID
	}
	if payload.Turn != nil {
		return payload.Turn.ID
	}
	return ""
}

func orString(value string, fallback string) string {
	if value != "" {
		return value
	}
	return fallback
}
