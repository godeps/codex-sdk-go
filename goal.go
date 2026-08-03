package codex

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"sync"
	"time"
)

const goalStartTimeout = 30 * time.Second

// GoalUpdate describes one low-level thread goal mutation.
type GoalUpdate struct {
	Objective   *string
	Status      *GoalStatus
	TokenBudget *int
}

type goalOperation struct {
	client   *Client
	threadID string

	mu            sync.Mutex
	logicalTurnID string
	currentTurnID string
	lastTurnID    string
	pendingTurns  []string
	status        GoalStatus
	cleared       bool
	failed        error

	signalCh        chan struct{}
	ctx             context.Context
	cancel          context.CancelFunc
	done            chan struct{}
	cleanupOnce     sync.Once
	registeredTurns map[string]struct{}
}

func newGoalOperation(client *Client, threadID string) *goalOperation {
	ctx, cancel := context.WithCancel(context.Background())
	return &goalOperation{
		client:          client,
		threadID:        threadID,
		signalCh:        make(chan struct{}, 1),
		ctx:             ctx,
		cancel:          cancel,
		done:            make(chan struct{}),
		registeredTurns: make(map[string]struct{}),
	}
}

func (g *goalOperation) notify() {
	select {
	case g.signalCh <- struct{}{}:
	default:
	}
}

func (g *goalOperation) fail(err error) {
	if err == nil {
		return
	}
	g.mu.Lock()
	if g.failed == nil {
		g.failed = err
	}
	g.mu.Unlock()
	g.notify()
}

func (g *goalOperation) closeRoute() {
	g.cleanupOnce.Do(func() {
		g.cancel()
		g.client.transport.unregisterGoal(g.threadID)
	})
}

func (g *goalOperation) observeGoalNotification(method string, params json.RawMessage) {
	g.mu.Lock()
	defer g.mu.Unlock()

	switch method {
	case "thread/goal/updated":
		var payload struct {
			TurnID string `json:"turnId"`
			Goal   struct {
				Status string `json:"status"`
			} `json:"goal"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			if g.failed == nil {
				g.failed = err
			}
			g.notify()
			return
		}
		g.status = GoalStatus(payload.Goal.Status)
		if g.status == GoalStatusActive {
			g.cleared = false
		}
		if payload.TurnID != "" {
			g.noteTurnID(payload.TurnID)
		}
	case "thread/goal/cleared":
		g.cleared = true
	case "turn/started":
		var payload struct {
			Turn struct {
				ID string `json:"id"`
			} `json:"turn"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			if g.failed == nil {
				g.failed = err
			}
			g.notify()
			return
		}
		if payload.Turn.ID != "" {
			g.noteTurnID(payload.Turn.ID)
		}
	case "turn/completed":
		var payload struct {
			Turn struct {
				ID string `json:"id"`
			} `json:"turn"`
		}
		if err := json.Unmarshal(params, &payload); err != nil {
			if g.failed == nil {
				g.failed = err
			}
			g.notify()
			return
		}
		if payload.Turn.ID != "" && g.currentTurnID == payload.Turn.ID {
			g.currentTurnID = ""
			g.lastTurnID = payload.Turn.ID
		}
	}
	g.notify()
}

func (g *goalOperation) noteTurnID(turnID string) {
	if turnID == "" {
		return
	}
	if g.logicalTurnID == "" {
		g.logicalTurnID = turnID
		g.currentTurnID = turnID
		g.lastTurnID = turnID
		return
	}
	g.currentTurnID = turnID
	g.lastTurnID = turnID
	if turnID == g.logicalTurnID {
		return
	}
	for _, existing := range g.pendingTurns {
		if existing == turnID {
			return
		}
	}
	g.pendingTurns = append(g.pendingTurns, turnID)
}

func (g *goalOperation) routeLoop() {
	defer close(g.done)
	for {
		raw, err := g.client.transport.nextGoal(g.ctx, g.threadID)
		if err != nil {
			if errors.Is(err, context.Canceled) {
				return
			}
			g.fail(err)
			return
		}
		method, params, splitErr := splitNotification(raw)
		if splitErr != nil {
			g.fail(splitErr)
			return
		}
		if turnID := goalRouteTurnID(method, params); turnID != "" {
			g.mu.Lock()
			_, exists := g.registeredTurns[turnID]
			if !exists {
				g.registeredTurns[turnID] = struct{}{}
			}
			g.mu.Unlock()
			if !exists {
				if err := g.client.transport.registerTurn(turnID); err != nil {
					g.fail(err)
					return
				}
			}
		}
		g.observeGoalNotification(method, params)
	}
}

func (g *goalOperation) awaitStart(ctx context.Context) (string, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	timeout := time.NewTimer(goalStartTimeout)
	defer timeout.Stop()

	for {
		g.mu.Lock()
		turnID := g.logicalTurnID
		failed := g.failed
		terminal := g.cleared || goalTerminal(g.status)
		g.mu.Unlock()

		switch {
		case failed != nil:
			return "", failed
		case turnID != "":
			return turnID, nil
		case terminal:
			return "", errors.New("codex: goal ended without starting a turn")
		}

		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-timeout.C:
			return "", errors.New("codex: timed out waiting for goal turn to start")
		case <-g.signalCh:
		}
	}
}

func (g *goalOperation) finishPhysicalTurn(turnID string) {
	g.mu.Lock()
	if g.currentTurnID == turnID {
		g.currentTurnID = ""
	}
	g.mu.Unlock()
	g.notify()
}

func (g *goalOperation) awaitContinuation(ctx context.Context, previousTurnID string) (string, bool, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	for {
		g.mu.Lock()
		var nextTurnID string
		if len(g.pendingTurns) > 0 {
			nextTurnID = g.pendingTurns[0]
		}
		failed := g.failed
		terminal := nextTurnID == "" && g.currentTurnID == "" && (g.cleared || goalTerminal(g.status))
		g.mu.Unlock()

		switch {
		case failed != nil:
			return "", false, failed
		case nextTurnID != "" && nextTurnID != previousTurnID:
			g.mu.Lock()
			if len(g.pendingTurns) > 0 && g.pendingTurns[0] == nextTurnID {
				g.pendingTurns = g.pendingTurns[1:]
			}
			g.mu.Unlock()
			return nextTurnID, false, nil
		case terminal:
			return "", true, nil
		}

		select {
		case <-ctx.Done():
			return "", false, ctx.Err()
		case <-g.signalCh:
		}
	}
}

func (g *goalOperation) interruptTarget() string {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.currentTurnID != "" {
		return g.currentTurnID
	}
	return g.lastTurnID
}

func goalTerminal(status GoalStatus) bool {
	switch status {
	case GoalStatusPaused, GoalStatusBlocked, GoalStatusUsageLimited, GoalStatusBudgetLimited, GoalStatusComplete:
		return true
	default:
		return false
	}
}

func goalRouteTurnID(method string, params json.RawMessage) string {
	switch method {
	case "thread/goal/updated":
		return notificationGoalTurnID(params)
	case "turn/started":
		return notificationTurnIDLocal(params)
	default:
		return ""
	}
}

// GoalHandle is the high-level logical goal operation bound to one thread.
type GoalHandle struct {
	client *Client
	thread *Thread
	state  *goalOperation

	closeOnce sync.Once
}

// GoalStream is the logical pull stream over one goal operation.
type GoalStream struct {
	handle *GoalHandle

	current       *TurnStream
	currentTurnID string
	done          bool
}

// GetGoalContext reads the stored thread goal.
func (t *Thread) GetGoalContext(ctx context.Context) (*Goal, error) {
	if err := t.ensurePrepared(ctx); err != nil {
		return nil, err
	}
	return t.rootClient().GetGoal(ctx, t.ID())
}

// SetGoalContext updates the stored thread goal.
func (t *Thread) SetGoalContext(ctx context.Context, update GoalUpdate) (*Goal, error) {
	if err := t.ensurePrepared(ctx); err != nil {
		return nil, err
	}
	return t.rootClient().SetGoal(ctx, t.ID(), update)
}

// ClearGoalContext clears the stored thread goal.
func (t *Thread) ClearGoalContext(ctx context.Context) error {
	if err := t.ensurePrepared(ctx); err != nil {
		return err
	}
	return t.rootClient().ClearGoal(ctx, t.ID())
}

// PauseGoalContext pauses the stored active goal.
func (t *Thread) PauseGoalContext(ctx context.Context) (*Goal, error) {
	if err := t.ensurePrepared(ctx); err != nil {
		return nil, err
	}
	return t.rootClient().PauseGoal(ctx, t.ID())
}

// StartGoalContext starts one logical goal operation on this thread.
func (t *Thread) StartGoalContext(ctx context.Context, objective string) (*GoalHandle, error) {
	if err := t.ensurePrepared(ctx); err != nil {
		return nil, err
	}
	return t.rootClient().StartGoal(ctx, t.ID(), objective)
}

// GetGoal reads the stored thread goal.
func (c *Client) GetGoal(ctx context.Context, threadID string) (*Goal, error) {
	var response struct {
		Goal map[string]any `json:"goal"`
	}
	if err := c.transport.request(ctx, "thread/goal/get", map[string]any{"threadId": threadID}, &response); err != nil {
		return nil, err
	}
	return decodeGoalFromAny(response.Goal), nil
}

// SetGoal updates the stored thread goal.
func (c *Client) SetGoal(ctx context.Context, threadID string, update GoalUpdate) (*Goal, error) {
	payload := map[string]any{"threadId": threadID}
	if update.Objective != nil {
		payload["objective"] = *update.Objective
	}
	if update.Status != nil {
		payload["status"] = string(*update.Status)
	}
	if update.TokenBudget != nil {
		payload["tokenBudget"] = *update.TokenBudget
	}
	var response struct {
		Goal map[string]any `json:"goal"`
	}
	if err := c.transport.request(ctx, "thread/goal/set", payload, &response); err != nil {
		return nil, err
	}
	return decodeGoalFromAny(response.Goal), nil
}

// ClearGoal clears the stored thread goal.
func (c *Client) ClearGoal(ctx context.Context, threadID string) error {
	return c.transport.request(ctx, "thread/goal/clear", map[string]any{"threadId": threadID}, nil)
}

// PauseGoal pauses the active goal on one thread.
func (c *Client) PauseGoal(ctx context.Context, threadID string) (*Goal, error) {
	status := GoalStatusPaused
	return c.SetGoal(ctx, threadID, GoalUpdate{Status: &status})
}

// StartGoal starts one logical goal operation and waits for its first runtime turn.
func (c *Client) StartGoal(ctx context.Context, threadID string, objective string) (*GoalHandle, error) {
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}

	var handle *GoalHandle
	err := c.withThreadLock(threadID, func() error {
		record, err := c.ReadThread(ctx, threadID, false)
		if err != nil {
			return err
		}
		if record.Status != ThreadStatusIdle {
			return &InvalidRequestError{RPCError: &RPCError{Code: -32600, Message: "thread must be idle before starting a goal: " + threadID}}
		}
		if record.Ephemeral || record.Path == "" {
			return &InvalidRequestError{RPCError: &RPCError{Code: -32600, Message: "thread must be persisted before starting a goal: " + threadID}}
		}

		state := newGoalOperation(c, threadID)
		if err := c.transport.registerGoal(threadID); err != nil {
			return err
		}
		go state.routeLoop()

		c.goalMu.Lock()
		if _, exists := c.goals[threadID]; exists {
			c.goalMu.Unlock()
			state.closeRoute()
			<-state.done
			return &InvalidRequestError{RPCError: &RPCError{Code: -32600, Message: "thread has an active goal operation: " + threadID}}
		}
		c.goals[threadID] = state
		c.goalMu.Unlock()

		cleanup := true
		defer func() {
			if cleanup {
				c.cancelGoalOperation(state)
				c.unregisterGoal(state)
			}
		}()

		if err := c.ClearGoal(ctx, threadID); err != nil {
			return err
		}
		status := GoalStatusActive
		if _, err := c.SetGoal(ctx, threadID, GoalUpdate{Objective: &objective, Status: &status}); err != nil {
			return err
		}
		if _, err := state.awaitStart(ctx); err != nil {
			return err
		}

		handle = &GoalHandle{
			client: c,
			thread: newClientThread(c, ThreadOptions{}, threadID, true),
			state:  state,
		}
		cleanup = false
		return nil
	})
	return handle, err
}

func (c *Client) unregisterGoal(state *goalOperation) {
	if state == nil {
		return
	}
	state.closeRoute()
	c.goalMu.Lock()
	delete(c.goals, state.threadID)
	c.goalMu.Unlock()
	<-state.done
}

func (c *Client) cancelGoalOperation(state *goalOperation) {
	if state == nil {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	_, _ = c.PauseGoal(ctx, state.threadID)

	turnID := state.interruptTarget()
	if turnID == "" {
		return
	}
	if err := c.interruptTurn(ctx, state.threadID, turnID); err != nil {
		nextTurnID := parseTurnIDMismatch(err)
		if nextTurnID == "" || nextTurnID == turnID {
			return
		}
		state.mu.Lock()
		state.currentTurnID = nextTurnID
		state.lastTurnID = nextTurnID
		state.mu.Unlock()
		_ = c.interruptTurn(ctx, state.threadID, nextTurnID)
	}
}

// StreamContext returns the logical goal stream.
func (h *GoalHandle) StreamContext(_ context.Context) *GoalStream {
	return &GoalStream{handle: h}
}

// RunContext consumes the logical goal stream and returns its final collected result.
func (h *GoalHandle) RunContext(ctx context.Context) (*TurnResult, error) {
	stream := h.StreamContext(ctx)
	defer stream.Close()
	return collectGoalResult(ctx, stream)
}

// CancelContext performs best-effort pause then interrupt cleanup.
func (h *GoalHandle) CancelContext(_ context.Context) error {
	if h == nil || h.state == nil {
		return nil
	}
	h.client.cancelGoalOperation(h.state)
	h.Close()
	return nil
}

// Close releases logical goal routing state exactly once.
func (h *GoalHandle) Close() {
	if h == nil {
		return
	}
	h.closeOnce.Do(func() {
		if h.state != nil {
			h.client.unregisterGoal(h.state)
		}
	})
}

// Next returns the next logical goal event.
func (s *GoalStream) Next(ctx context.Context) (*ThreadEvent, error) {
	if s == nil || s.handle == nil || s.handle.state == nil {
		return nil, ErrTransportClosed
	}
	if ctx == nil {
		ctx = context.Background()
	}
	if s.done {
		return nil, io.EOF
	}

	for {
		if s.current == nil {
			if s.currentTurnID == "" {
				s.currentTurnID = s.handle.state.logicalTurnID
			}
			if s.currentTurnID == "" {
				s.currentTurnID = s.handle.state.interruptTarget()
			}
			if s.currentTurnID == "" {
				s.done = true
				s.handle.Close()
				return nil, io.EOF
			}
			handle := &TurnHandle{
				client:   s.handle.client,
				threadID: s.handle.state.threadID,
				id:       s.currentTurnID,
			}
			stream, err := handle.StreamContext(ctx)
			if err != nil {
				s.handle.Close()
				return nil, err
			}
			s.current = stream
		}

		event, err := s.current.Next(ctx)
		if err != nil {
			_ = s.current.Close()
			s.current = nil
			s.handle.Close()
			return nil, err
		}
		event = rewriteLogicalGoalEvent(event, s.handle.state.logicalTurnID)

		switch event.Type {
		case "turn.started":
			if s.currentTurnID == s.handle.state.logicalTurnID {
				return event, nil
			}
			continue
		case "item.completed", "thread.token_usage.updated":
			return event, nil
		case "turn.completed", "turn.failed":
			_ = s.current.Close()
			s.current = nil
			s.handle.state.finishPhysicalTurn(s.currentTurnID)
			if event.Type == "turn.failed" {
				s.done = true
				s.handle.Close()
				return event, nil
			}
			nextTurnID, terminal, nextErr := s.handle.state.awaitContinuation(ctx, s.currentTurnID)
			if nextErr != nil {
				s.handle.Close()
				return nil, nextErr
			}
			if terminal {
				s.done = true
				s.handle.Close()
				return event, nil
			}
			if err := s.handle.client.transport.registerTurn(nextTurnID); err != nil {
				s.handle.Close()
				return nil, err
			}
			s.currentTurnID = nextTurnID
			continue
		default:
			return event, nil
		}
	}
}

// Close releases the logical goal operation.
func (s *GoalStream) Close() error {
	if s == nil || s.handle == nil {
		return nil
	}
	s.handle.Close()
	return nil
}

func collectGoalResult(ctx context.Context, stream *GoalStream) (*TurnResult, error) {
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
		return nil, errors.New("codex: goal completed event not received")
	}
	if completed.Status == TurnStatusFailed {
		if turnErr != nil && turnErr.Message != "" {
			return nil, errors.New(turnErr.Message)
		}
		return nil, errors.New("codex: goal turn failed")
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

func rewriteLogicalGoalEvent(event *ThreadEvent, logicalTurnID string) *ThreadEvent {
	if event == nil || logicalTurnID == "" {
		return event
	}
	out := *event
	out.TurnID = logicalTurnID
	if out.Turn != nil {
		turn := *out.Turn
		turn.ID = logicalTurnID
		out.Turn = &turn
	}
	return &out
}
