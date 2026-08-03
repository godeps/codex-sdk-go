package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
)

const mib = 1024 * 1024

// Limits bounds queued router state so transport failures are explicit.
type Limits struct {
	MaxResponseWaiters int
	MaxRouteEvents     int
	MaxRouteBytes      int
	MaxGlobalEvents    int
	MaxGlobalBytes     int
	MaxTotalEvents     int
	MaxTotalBytes      int
}

// DefaultLimits matches the bounded transport defaults in the acceptance spec.
func DefaultLimits() Limits {
	return Limits{
		MaxResponseWaiters: 1024,
		MaxRouteEvents:     256,
		MaxRouteBytes:      8 * mib,
		MaxGlobalEvents:    1024,
		MaxGlobalBytes:     32 * mib,
		MaxTotalEvents:     4096,
		MaxTotalBytes:      64 * mib,
	}
}

// DefaultLimitsIfZero fills each zero-valued field independently.
func DefaultLimitsIfZero(limits Limits) Limits {
	return normalizeLimits(limits)
}

func normalizeLimits(limits Limits) Limits {
	defaults := DefaultLimits()
	if limits.MaxResponseWaiters <= 0 {
		limits.MaxResponseWaiters = defaults.MaxResponseWaiters
	}
	if limits.MaxRouteEvents <= 0 {
		limits.MaxRouteEvents = defaults.MaxRouteEvents
	}
	if limits.MaxRouteBytes <= 0 {
		limits.MaxRouteBytes = defaults.MaxRouteBytes
	}
	if limits.MaxGlobalEvents <= 0 {
		limits.MaxGlobalEvents = defaults.MaxGlobalEvents
	}
	if limits.MaxGlobalBytes <= 0 {
		limits.MaxGlobalBytes = defaults.MaxGlobalBytes
	}
	if limits.MaxTotalEvents <= 0 {
		limits.MaxTotalEvents = defaults.MaxTotalEvents
	}
	if limits.MaxTotalBytes <= 0 {
		limits.MaxTotalBytes = defaults.MaxTotalBytes
	}
	return limits
}

// Response is one routed JSON-RPC request result.
type Response struct {
	Result json.RawMessage
	Err    error
}

type routeItem struct {
	Raw  json.RawMessage
	Err  error
	Size int
}

type routeState struct {
	queue   chan routeItem
	pending []routeItem
	events  int
	bytes   int
}

type routeKind int

const (
	loginRoute routeKind = iota
	turnRoute
	goalRoute
)

// MessageRouter keeps one stdout reader from competing consumers.
type MessageRouter struct {
	mu sync.Mutex

	limits Limits
	failed error

	responseWaiters map[string]chan Response

	loginRoutes map[string]*routeState
	turnRoutes  map[string]*routeState
	goalRoutes  map[string]*routeState
	loginClosed map[string]struct{}
	turnClosed  map[string]struct{}
	goalClosed  map[string]struct{}

	globalQueue  chan routeItem
	globalEvents int
	globalBytes  int

	totalEvents int
	totalBytes  int
}

// New constructs an empty router.
func New(limits Limits) *MessageRouter {
	limits = normalizeLimits(limits)
	return &MessageRouter{
		limits:          limits,
		responseWaiters: make(map[string]chan Response),
		loginRoutes:     make(map[string]*routeState),
		turnRoutes:      make(map[string]*routeState),
		goalRoutes:      make(map[string]*routeState),
		loginClosed:     make(map[string]struct{}),
		turnClosed:      make(map[string]struct{}),
		goalClosed:      make(map[string]struct{}),
		globalQueue:     make(chan routeItem, limits.MaxGlobalEvents),
	}
}

// CreateResponseWaiter registers a one-shot response queue for one request ID.
func (r *MessageRouter) CreateResponseWaiter(id string) (<-chan Response, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failed != nil {
		return nil, r.failed
	}
	if _, exists := r.responseWaiters[id]; exists {
		return nil, fmt.Errorf("router: response waiter %q already registered", id)
	}
	if len(r.responseWaiters) >= r.limits.MaxResponseWaiters {
		return nil, errors.New("router: response waiter limit exceeded")
	}
	waiter := make(chan Response, 1)
	r.responseWaiters[id] = waiter
	return waiter, nil
}

// DiscardResponseWaiter removes a response waiter after write or context failure.
func (r *MessageRouter) DiscardResponseWaiter(id string) {
	r.mu.Lock()
	delete(r.responseWaiters, id)
	r.mu.Unlock()
}

// RegisterTurn replays early notifications into a dedicated turn queue.
func (r *MessageRouter) RegisterTurn(turnID string) error {
	return r.registerRoute(turnID, turnRoute)
}

// UnregisterTurn releases routing for a completed or canceled turn.
func (r *MessageRouter) UnregisterTurn(turnID string) {
	r.unregisterRoute(turnID, turnRoute)
}

// NextTurn blocks until the next routed turn notification arrives.
func (r *MessageRouter) NextTurn(turnID string, ctxs ...context.Context) (json.RawMessage, error) {
	return r.nextRoute(turnID, turnRoute, "turn", firstContext(ctxs...))
}

// RegisterLogin starts routing notifications for one login attempt.
func (r *MessageRouter) RegisterLogin(loginID string) error {
	return r.registerRoute(loginID, loginRoute)
}

// RegisterGoal starts routing thread-scoped goal notifications.
func (r *MessageRouter) RegisterGoal(threadID string) error {
	return r.registerRoute(threadID, goalRoute)
}

// UnregisterLogin releases routing state for one login attempt.
func (r *MessageRouter) UnregisterLogin(loginID string) {
	r.unregisterRoute(loginID, loginRoute)
}

// UnregisterGoal releases routing state for one logical goal operation.
func (r *MessageRouter) UnregisterGoal(threadID string) {
	r.unregisterRoute(threadID, goalRoute)
}

// NextLogin blocks until the next routed login notification arrives.
func (r *MessageRouter) NextLogin(loginID string, ctxs ...context.Context) (json.RawMessage, error) {
	return r.nextRoute(loginID, loginRoute, "login", firstContext(ctxs...))
}

// NextGoal blocks until the next routed goal notification arrives.
func (r *MessageRouter) NextGoal(threadID string, ctxs ...context.Context) (json.RawMessage, error) {
	return r.nextRoute(threadID, goalRoute, "goal", firstContext(ctxs...))
}

// NextGlobal blocks until the next unscoped notification arrives.
func (r *MessageRouter) NextGlobal(ctxs ...context.Context) (json.RawMessage, error) {
	ctx := firstContext(ctxs...)
	if ctx == nil {
		ctx = context.Background()
	}
	if item, ok := r.tryNextGlobal(); ok {
		return item.Raw, item.Err
	}
	r.mu.Lock()
	failed := r.failed
	r.mu.Unlock()
	if failed != nil {
		return nil, failed
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case item := <-r.globalQueue:
		r.releaseGlobal(item.Size)
		return item.Raw, item.Err
	}
}

// RouteResponse delivers one JSON-RPC response to its waiter.
func (r *MessageRouter) RouteResponse(id string, result json.RawMessage, routeErr error) {
	r.mu.Lock()
	waiter := r.responseWaiters[id]
	delete(r.responseWaiters, id)
	r.mu.Unlock()
	if waiter == nil {
		return
	}
	waiter <- Response{Result: cloneRaw(result), Err: routeErr}
}

// RouteNotification delivers one notification to the appropriate queue.
func (r *MessageRouter) RouteNotification(method string, params json.RawMessage) error {
	envelope, err := notificationEnvelope(method, params)
	if err != nil {
		return err
	}
	item := routeItem{Raw: envelope, Size: len(envelope)}

	if loginID := notificationLoginID(method, params); loginID != "" {
		return r.routeScoped(loginID, item, loginRoute, false)
	}

	if turnID := notificationTurnID(params); turnID != "" {
		if err := r.routeScoped(turnID, item, turnRoute, false); err != nil {
			return err
		}
	}

	if threadID := notificationThreadID(params); threadID != "" && goalRelevantMethod(method) {
		if err := r.routeScoped(threadID, item, goalRoute, true); err != nil {
			return err
		}
		if notificationTurnID(params) != "" {
			return nil
		}
	}

	if notificationTurnID(params) != "" {
		return nil
	}

	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failed != nil {
		return r.failed
	}
	if r.globalEvents+1 > r.limits.MaxGlobalEvents {
		return errors.New("router: global event limit exceeded")
	}
	if r.globalBytes+item.Size > r.limits.MaxGlobalBytes {
		return errors.New("router: global byte limit exceeded")
	}
	if err := r.reserveTotal(item.Size); err != nil {
		return err
	}
	r.globalEvents++
	r.globalBytes += item.Size
	r.globalQueue <- item
	return nil
}

// FailAll wakes all blocked consumers after a transport failure.
func (r *MessageRouter) FailAll(err error) {
	if err == nil {
		err = errors.New("router: failed")
	}

	r.mu.Lock()
	if r.failed != nil {
		r.mu.Unlock()
		return
	}
	r.failed = err

	waiters := r.responseWaiters
	loginQueues := routeQueuesSnapshot(r.loginRoutes)
	turnQueues := routeQueuesSnapshot(r.turnRoutes)
	goalQueues := routeQueuesSnapshot(r.goalRoutes)
	r.responseWaiters = make(map[string]chan Response)
	r.mu.Unlock()

	for _, waiter := range waiters {
		waiter <- Response{Err: err}
	}
	for _, queue := range loginQueues {
		if queue != nil {
			select {
			case queue <- routeItem{Err: err}:
			default:
			}
		}
	}
	for _, queue := range turnQueues {
		if queue != nil {
			select {
			case queue <- routeItem{Err: err}:
			default:
			}
		}
	}
	for _, queue := range goalQueues {
		if queue != nil {
			select {
			case queue <- routeItem{Err: err}:
			default:
			}
		}
	}
	select {
	case r.globalQueue <- routeItem{Err: err}:
	default:
	}
}

func (r *MessageRouter) registerRoute(id string, kind routeKind) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failed != nil {
		return r.failed
	}
	routes, closed := r.routeMaps(kind)
	delete(closed, id)
	state := routes[id]
	if state == nil {
		state = &routeState{}
		routes[id] = state
	}
	if state.queue != nil {
		return nil
	}
	state.queue = make(chan routeItem, r.limits.MaxRouteEvents)
	for _, item := range state.pending {
		state.queue <- item
	}
	state.pending = nil
	return nil
}

func (r *MessageRouter) unregisterRoute(id string, kind routeKind) {
	r.mu.Lock()
	defer r.mu.Unlock()
	routes, closed := r.routeMaps(kind)
	if state := routes[id]; state != nil {
		r.totalEvents -= state.events
		r.totalBytes -= state.bytes
	}
	delete(routes, id)
	closed[id] = struct{}{}
}

func (r *MessageRouter) nextRoute(id string, routeKind routeKind, kind string, ctx context.Context) (json.RawMessage, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	r.mu.Lock()
	routes, _ := r.routeMaps(routeKind)
	state := routes[id]
	failed := r.failed
	r.mu.Unlock()
	if state == nil || state.queue == nil {
		return nil, fmt.Errorf("router: %s %q is not registered", kind, id)
	}
	if item, ok := r.tryNextRoute(id, state, routes); ok {
		return item.Raw, item.Err
	}
	if failed != nil {
		return nil, failed
	}
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	case item := <-state.queue:
		r.releaseRouteItem(id, state, routes, item.Size)
		return item.Raw, item.Err
	}
}

func (r *MessageRouter) routeScoped(id string, item routeItem, kind routeKind, onlyIfRegistered bool) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.failed != nil {
		return r.failed
	}
	routes, closed := r.routeMaps(kind)
	if _, dismissed := closed[id]; dismissed {
		return nil
	}
	state := routes[id]
	if onlyIfRegistered && state == nil {
		return nil
	}
	if state == nil {
		state = &routeState{}
		routes[id] = state
	}
	if state.events+1 > r.limits.MaxRouteEvents {
		return fmt.Errorf("router: route event limit exceeded for %q", id)
	}
	if state.bytes+item.Size > r.limits.MaxRouteBytes {
		return fmt.Errorf("router: route byte limit exceeded for %q", id)
	}
	if err := r.reserveTotal(item.Size); err != nil {
		return err
	}
	state.events++
	state.bytes += item.Size
	if state.queue == nil {
		state.pending = append(state.pending, item)
		return nil
	}
	state.queue <- item
	return nil
}

func (r *MessageRouter) reserveTotal(size int) error {
	if r.totalEvents+1 > r.limits.MaxTotalEvents {
		return errors.New("router: total event limit exceeded")
	}
	if r.totalBytes+size > r.limits.MaxTotalBytes {
		return errors.New("router: total byte limit exceeded")
	}
	r.totalEvents++
	r.totalBytes += size
	return nil
}

func cloneRaw(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	return append(json.RawMessage(nil), raw...)
}

func notificationTurnID(raw json.RawMessage) string {
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

func notificationLoginID(method string, raw json.RawMessage) string {
	if method != "account/login/completed" {
		return ""
	}
	var payload struct {
		LoginID string `json:"loginId"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return payload.LoginID
}

func notificationThreadID(raw json.RawMessage) string {
	var payload struct {
		ThreadID string `json:"threadId"`
	}
	if err := json.Unmarshal(raw, &payload); err != nil {
		return ""
	}
	return payload.ThreadID
}

func notificationEnvelope(method string, params json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(map[string]any{
		"method": method,
		"params": json.RawMessage(cloneRaw(params)),
	})
}

func goalRelevantMethod(method string) bool {
	switch method {
	case "thread/goal/updated", "thread/goal/cleared", "turn/started", "turn/completed", "turn/failed":
		return true
	default:
		return false
	}
}

func firstContext(ctxs ...context.Context) context.Context {
	for _, ctx := range ctxs {
		if ctx != nil {
			return ctx
		}
	}
	return context.Background()
}

func (r *MessageRouter) routeMaps(kind routeKind) (map[string]*routeState, map[string]struct{}) {
	switch kind {
	case loginRoute:
		return r.loginRoutes, r.loginClosed
	case turnRoute:
		return r.turnRoutes, r.turnClosed
	case goalRoute:
		return r.goalRoutes, r.goalClosed
	default:
		return nil, nil
	}
}

func routeQueuesSnapshot(routes map[string]*routeState) []chan routeItem {
	queues := make([]chan routeItem, 0, len(routes))
	for _, state := range routes {
		queues = append(queues, state.queue)
	}
	return queues
}

func (r *MessageRouter) tryNextGlobal() (routeItem, bool) {
	select {
	case item := <-r.globalQueue:
		r.releaseGlobal(item.Size)
		return item, true
	default:
		return routeItem{}, false
	}
}

func (r *MessageRouter) tryNextRoute(id string, state *routeState, routes map[string]*routeState) (routeItem, bool) {
	select {
	case item := <-state.queue:
		r.releaseRouteItem(id, state, routes, item.Size)
		return item, true
	default:
		return routeItem{}, false
	}
}

func (r *MessageRouter) releaseGlobal(size int) {
	r.mu.Lock()
	r.globalEvents--
	r.globalBytes -= size
	r.totalEvents--
	r.totalBytes -= size
	r.mu.Unlock()
}

func (r *MessageRouter) releaseRouteItem(id string, state *routeState, routes map[string]*routeState, size int) {
	r.mu.Lock()
	if current := routes[id]; current == state {
		state.events--
		state.bytes -= size
		r.totalEvents--
		r.totalBytes -= size
	}
	r.mu.Unlock()
}
