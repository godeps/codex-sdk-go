package router

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestRouterReplaysPendingTurnNotifications(t *testing.T) {
	r := New(DefaultLimits())
	raw := json.RawMessage(`{"turn":{"id":"turn-1"},"kind":"started"}`)
	if err := r.RouteNotification("turn/started", raw); err != nil {
		t.Fatalf("RouteNotification() error = %v", err)
	}
	if err := r.RegisterTurn("turn-1"); err != nil {
		t.Fatalf("RegisterTurn() error = %v", err)
	}
	got, err := r.NextTurn("turn-1")
	if err != nil {
		t.Fatalf("NextTurn() error = %v", err)
	}
	if string(got) != `{"method":"turn/started","params":{"turn":{"id":"turn-1"},"kind":"started"}}` {
		t.Fatalf("NextTurn() = %s, want full envelope", got)
	}
}

func TestRouterReplaysCompactionNotificationByThread(t *testing.T) {
	r := New(DefaultLimits())
	raw := json.RawMessage(`{"threadId":"thread-1","turnId":"compact-turn-1"}`)
	if err := r.RouteNotification("thread/compacted", raw); err != nil {
		t.Fatalf("RouteNotification() error = %v", err)
	}
	if err := r.RegisterCompaction("thread-1"); err != nil {
		t.Fatalf("RegisterCompaction() error = %v", err)
	}
	got, err := r.NextCompaction("thread-1")
	if err != nil {
		t.Fatalf("NextCompaction() error = %v", err)
	}
	if string(got) != `{"method":"thread/compacted","params":{"threadId":"thread-1","turnId":"compact-turn-1"}}` {
		t.Fatalf("NextCompaction() = %s, want full envelope", got)
	}
}

func TestRouterLoginReplayAndUnregisterDropsLateEvents(t *testing.T) {
	r := New(DefaultLimits())
	raw := json.RawMessage(`{"loginId":"login-1","status":"ok"}`)
	if err := r.RouteNotification("account/login/completed", raw); err != nil {
		t.Fatalf("RouteNotification() error = %v", err)
	}
	if err := r.RegisterLogin("login-1"); err != nil {
		t.Fatalf("RegisterLogin() error = %v", err)
	}
	got, err := r.NextLogin("login-1")
	if err != nil {
		t.Fatalf("NextLogin() error = %v", err)
	}
	if string(got) != `{"method":"account/login/completed","params":{"loginId":"login-1","status":"ok"}}` {
		t.Fatalf("NextLogin() = %s, want full envelope", got)
	}
	r.UnregisterLogin("login-1")
	if err := r.RouteNotification("account/login/completed", raw); err != nil {
		t.Fatalf("late RouteNotification() error = %v", err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Millisecond)
	defer cancel()
	if _, err := r.NextLogin("login-1", ctx); err == nil {
		t.Fatal("NextLogin() error = nil, want unregistered failure")
	}
}

func TestRouterContextCancelDoesNotConsumeQueuedEvent(t *testing.T) {
	r := New(DefaultLimits())
	if err := r.RegisterTurn("turn-1"); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.NextTurn("turn-1", ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("NextTurn() error = %v, want context canceled", err)
	}
	raw := json.RawMessage(`{"turn":{"id":"turn-1"},"kind":"after-cancel"}`)
	if err := r.RouteNotification("turn/updated", raw); err != nil {
		t.Fatalf("RouteNotification() error = %v", err)
	}
	got, err := r.NextTurn("turn-1")
	if err != nil {
		t.Fatalf("NextTurn() after cancel error = %v", err)
	}
	if string(got) != `{"method":"turn/updated","params":{"turn":{"id":"turn-1"},"kind":"after-cancel"}}` {
		t.Fatalf("NextTurn() after cancel = %s, want full envelope", got)
	}
}

func TestRouterGlobalContextCancel(t *testing.T) {
	r := New(DefaultLimits())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.NextGlobal(ctx); !errors.Is(err, context.Canceled) {
		t.Fatalf("NextGlobal() error = %v, want context canceled", err)
	}
}

func TestRouterFailAllWakesResponseWaiterAndRoutes(t *testing.T) {
	r := New(DefaultLimits())
	waiter, err := r.CreateResponseWaiter("req-1")
	if err != nil {
		t.Fatalf("CreateResponseWaiter() error = %v", err)
	}
	if err := r.RegisterTurn("turn-1"); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterLogin("login-1"); err != nil {
		t.Fatal(err)
	}
	turnErrCh := make(chan error, 1)
	loginErrCh := make(chan error, 1)
	globalErrCh := make(chan error, 1)
	go func() {
		_, err := r.NextTurn("turn-1")
		turnErrCh <- err
	}()
	go func() {
		_, err := r.NextLogin("login-1")
		loginErrCh <- err
	}()
	go func() {
		_, err := r.NextGlobal()
		globalErrCh <- err
	}()
	want := errors.New("boom")
	r.FailAll(want)
	got := <-waiter
	if !errors.Is(got.Err, want) {
		t.Fatalf("waiter error = %v, want %v", got.Err, want)
	}
	if err := <-turnErrCh; !errors.Is(err, want) {
		t.Fatalf("blocked NextTurn() error = %v, want %v", err, want)
	}
	if err := <-loginErrCh; !errors.Is(err, want) {
		t.Fatalf("blocked NextLogin() error = %v, want %v", err, want)
	}
	if err := <-globalErrCh; !errors.Is(err, want) {
		t.Fatalf("blocked NextGlobal() error = %v, want %v", err, want)
	}
}

func TestRouterEnforcesRouteEventAndByteLimits(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxRouteEvents = 1
	limits.MaxRouteBytes = 8
	r := New(limits)
	if err := r.RouteNotification("turn/started", json.RawMessage(`{"turn":{"id":"turn-1"}}`)); err == nil {
		t.Fatal("RouteNotification() error = nil, want route byte limit failure")
	}

	limits = DefaultLimits()
	limits.MaxRouteEvents = 1
	limits.MaxRouteBytes = 1024
	r = New(limits)
	if err := r.RouteNotification("turn/started", json.RawMessage(`{"turn":{"id":"turn-1"}}`)); err != nil {
		t.Fatalf("first RouteNotification() error = %v", err)
	}
	if err := r.RouteNotification("turn/updated", json.RawMessage(`{"turn":{"id":"turn-1"}}`)); err == nil {
		t.Fatal("second RouteNotification() error = nil, want route event limit failure")
	}
}

func TestRouterEnforcesGlobalAndTotalLimits(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxGlobalEvents = 1
	limits.MaxGlobalBytes = 1024
	limits.MaxTotalEvents = 1
	limits.MaxTotalBytes = 1024
	r := New(limits)
	raw := json.RawMessage(`{"threadId":"thread-1"}`)
	if err := r.RouteNotification("thread/goal/updated", raw); err != nil {
		t.Fatalf("first RouteNotification() error = %v", err)
	}
	if err := r.RouteNotification("thread/goal/cleared", raw); err == nil {
		t.Fatal("second RouteNotification() error = nil, want total/global limit failure")
	}
}

func TestRouterUnregisterReleasesTotalBudget(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxTotalEvents = 1
	limits.MaxTotalBytes = 1024
	r := New(limits)
	raw := json.RawMessage(`{"turn":{"id":"turn-1"}}`)
	if err := r.RouteNotification("turn/started", raw); err != nil {
		t.Fatal(err)
	}
	r.UnregisterTurn("turn-1")
	if err := r.RouteNotification("thread/goal/updated", json.RawMessage(`{"threadId":"thread-1"}`)); err != nil {
		t.Fatalf("RouteNotification() after unregister error = %v", err)
	}
}

func TestRouterRoutesGoalEnvelopeWithoutSwallowingTurnRoute(t *testing.T) {
	r := New(DefaultLimits())
	if err := r.RegisterGoal("thread-1"); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterTurn("turn-1"); err != nil {
		t.Fatal(err)
	}
	raw := json.RawMessage(`{"threadId":"thread-1","turn":{"id":"turn-1"}}`)
	if err := r.RouteNotification("turn/started", raw); err != nil {
		t.Fatal(err)
	}
	goalEnvelope, err := r.NextGoal("thread-1")
	if err != nil {
		t.Fatalf("NextGoal() error = %v", err)
	}
	turnEnvelope, err := r.NextTurn("turn-1")
	if err != nil {
		t.Fatalf("NextTurn() error = %v", err)
	}
	want := `{"method":"turn/started","params":{"threadId":"thread-1","turn":{"id":"turn-1"}}}`
	if string(goalEnvelope) != want {
		t.Fatalf("NextGoal() = %s, want %s", goalEnvelope, want)
	}
	if string(turnEnvelope) != want {
		t.Fatalf("NextTurn() = %s, want %s", turnEnvelope, want)
	}
}

func TestRouterGoalIsolationAcrossThreads(t *testing.T) {
	r := New(DefaultLimits())
	for i := 0; i < 20; i++ {
		threadID := fmt.Sprintf("thread-%d", i)
		if err := r.RegisterGoal(threadID); err != nil {
			t.Fatalf("RegisterGoal(%s) error = %v", threadID, err)
		}
	}
	for i := 0; i < 20; i++ {
		threadID := fmt.Sprintf("thread-%d", i)
		raw := json.RawMessage(fmt.Sprintf(`{"threadId":"%s","goal":{"status":"active"}}`, threadID))
		if err := r.RouteNotification("thread/goal/updated", raw); err != nil {
			t.Fatalf("RouteNotification(%s) error = %v", threadID, err)
		}
	}
	for i := 0; i < 20; i++ {
		threadID := fmt.Sprintf("thread-%d", i)
		envelope, err := r.NextGoal(threadID)
		if err != nil {
			t.Fatalf("NextGoal(%s) error = %v", threadID, err)
		}
		if !strings.Contains(string(envelope), `"threadId":"`+threadID+`"`) {
			t.Fatalf("NextGoal(%s) = %s", threadID, envelope)
		}
	}
}

func TestRouterTurnIsolationAcrossTwentyConcurrentTurns(t *testing.T) {
	r := New(DefaultLimits())
	const turns = 20
	for i := 0; i < turns; i++ {
		turnID := fmt.Sprintf("turn-%d", i)
		raw := json.RawMessage(fmt.Sprintf(`{"threadId":"thread-%d","turn":{"id":"%s"},"seq":%d}`, i, turnID, i))
		if i%2 == 0 {
			if err := r.RouteNotification("turn/started", raw); err != nil {
				t.Fatalf("early RouteNotification(%s) error = %v", turnID, err)
			}
		}
		if err := r.RegisterTurn(turnID); err != nil {
			t.Fatalf("RegisterTurn(%s) error = %v", turnID, err)
		}
		if i%2 != 0 {
			if err := r.RouteNotification("turn/started", raw); err != nil {
				t.Fatalf("late RouteNotification(%s) error = %v", turnID, err)
			}
		}
	}

	var wg sync.WaitGroup
	for i := 0; i < turns; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			turnID := fmt.Sprintf("turn-%d", i)
			envelope, err := r.NextTurn(turnID)
			if err != nil {
				t.Errorf("NextTurn(%s) error = %v", turnID, err)
				return
			}
			if !strings.Contains(string(envelope), `"turn":{"id":"`+turnID+`"}`) {
				t.Errorf("NextTurn(%s) = %s", turnID, envelope)
			}
			if !strings.Contains(string(envelope), `"threadId":"thread-`+strconv.Itoa(i)+`"`) {
				t.Errorf("NextTurn(%s) missing thread id in %s", turnID, envelope)
			}
		}(i)
	}
	wg.Wait()
}

func TestRouterLoginIsolationAcrossConcurrentWaiters(t *testing.T) {
	r := New(DefaultLimits())
	const logins = 20
	for i := 0; i < logins; i++ {
		loginID := fmt.Sprintf("login-%d", i)
		raw := json.RawMessage(fmt.Sprintf(`{"loginId":"%s","status":"ok","seq":%d}`, loginID, i))
		if i%2 == 0 {
			if err := r.RouteNotification("account/login/completed", raw); err != nil {
				t.Fatalf("early login notification %s error = %v", loginID, err)
			}
		}
		if err := r.RegisterLogin(loginID); err != nil {
			t.Fatalf("RegisterLogin(%s) error = %v", loginID, err)
		}
		if i%2 != 0 {
			if err := r.RouteNotification("account/login/completed", raw); err != nil {
				t.Fatalf("late login notification %s error = %v", loginID, err)
			}
		}
	}
	for i := 0; i < logins; i++ {
		loginID := fmt.Sprintf("login-%d", i)
		envelope, err := r.NextLogin(loginID)
		if err != nil {
			t.Fatalf("NextLogin(%s) error = %v", loginID, err)
		}
		if !strings.Contains(string(envelope), `"loginId":"`+loginID+`"`) {
			t.Fatalf("NextLogin(%s) = %s", loginID, envelope)
		}
	}
}

func TestRouterLateResponsesAndUnknownWaitersAreIgnored(t *testing.T) {
	r := New(DefaultLimits())
	r.RouteResponse("missing", json.RawMessage(`{"ok":true}`), nil)
	waiter, err := r.CreateResponseWaiter("req-1")
	if err != nil {
		t.Fatal(err)
	}
	r.RouteResponse("req-1", json.RawMessage(`{"ok":true}`), nil)
	got := <-waiter
	if string(got.Result) != `{"ok":true}` || got.Err != nil {
		t.Fatalf("RouteResponse() = %#v", got)
	}
	r.RouteResponse("req-1", json.RawMessage(`{"late":true}`), nil)
}

func TestRouterRouteOverflowIsolatedButTotalFailureIsTerminal(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxRouteEvents = 1
	limits.MaxRouteBytes = 1024
	r := New(limits)
	if err := r.RegisterTurn("turn-1"); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterTurn("turn-2"); err != nil {
		t.Fatal(err)
	}
	raw1 := json.RawMessage(`{"turn":{"id":"turn-1"}}`)
	raw2 := json.RawMessage(`{"turn":{"id":"turn-2"}}`)
	if err := r.RouteNotification("turn/started", raw1); err != nil {
		t.Fatalf("first RouteNotification(turn-1) error = %v", err)
	}
	if err := r.RouteNotification("turn/updated", raw1); err == nil {
		t.Fatal("second RouteNotification(turn-1) error = nil, want route overflow")
	}
	if err := r.RouteNotification("turn/started", raw2); err != nil {
		t.Fatalf("RouteNotification(turn-2) after isolated overflow error = %v", err)
	}
	if _, err := r.NextTurn("turn-2"); err != nil {
		t.Fatalf("NextTurn(turn-2) error = %v", err)
	}
}

func TestDefaultLimitsIfZeroKeepsExplicitFields(t *testing.T) {
	limits := DefaultLimitsIfZero(Limits{
		MaxRouteEvents: 7,
		MaxTotalBytes:  17,
	})
	if limits.MaxRouteEvents != 7 {
		t.Fatalf("MaxRouteEvents = %d, want 7", limits.MaxRouteEvents)
	}
	if limits.MaxTotalBytes != 17 {
		t.Fatalf("MaxTotalBytes = %d, want 17", limits.MaxTotalBytes)
	}
	if limits.MaxGlobalEvents != DefaultLimits().MaxGlobalEvents {
		t.Fatalf("MaxGlobalEvents = %d, want default %d", limits.MaxGlobalEvents, DefaultLimits().MaxGlobalEvents)
	}
}

func TestRouterCreateResponseWaiterErrors(t *testing.T) {
	limits := DefaultLimits()
	limits.MaxResponseWaiters = 1
	r := New(limits)
	if _, err := r.CreateResponseWaiter("req-1"); err != nil {
		t.Fatal(err)
	}
	if _, err := r.CreateResponseWaiter("req-1"); err == nil {
		t.Fatal("duplicate waiter error = nil")
	}
	if _, err := r.CreateResponseWaiter("req-2"); err == nil {
		t.Fatal("limit error = nil")
	}
}

func TestRouterDefaultLimitsMatchAcceptanceSpec(t *testing.T) {
	limits := DefaultLimits()
	if limits.MaxResponseWaiters != 1024 {
		t.Fatalf("MaxResponseWaiters = %d, want 1024", limits.MaxResponseWaiters)
	}
	if limits.MaxRouteEvents != 256 || limits.MaxRouteBytes != 8*mib {
		t.Fatalf("route limits = (%d,%d), want (256,%d)", limits.MaxRouteEvents, limits.MaxRouteBytes, 8*mib)
	}
	if limits.MaxGlobalEvents != 1024 || limits.MaxGlobalBytes != 32*mib {
		t.Fatalf("global limits = (%d,%d), want (1024,%d)", limits.MaxGlobalEvents, limits.MaxGlobalBytes, 32*mib)
	}
	if limits.MaxTotalEvents != 4096 || limits.MaxTotalBytes != 64*mib {
		t.Fatalf("total limits = (%d,%d), want (4096,%d)", limits.MaxTotalEvents, limits.MaxTotalBytes, 64*mib)
	}
}

func TestRouterFailAllBroadcastsToAllConsumersExactlyOnce(t *testing.T) {
	r := New(DefaultLimits())
	waiter, err := r.CreateResponseWaiter("req-1")
	if err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterTurn("turn-1"); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterLogin("login-1"); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterGoal("thread-1"); err != nil {
		t.Fatal(err)
	}
	turnErr := make(chan error, 1)
	loginErr := make(chan error, 1)
	goalErr := make(chan error, 1)
	globalErr := make(chan error, 1)
	go func() { _, err := r.NextTurn("turn-1"); turnErr <- err }()
	go func() { _, err := r.NextLogin("login-1"); loginErr <- err }()
	go func() { _, err := r.NextGoal("thread-1"); goalErr <- err }()
	go func() { _, err := r.NextGlobal(); globalErr <- err }()

	want := errors.New("transport failed")
	r.FailAll(want)
	if got := <-waiter; !errors.Is(got.Err, want) {
		t.Fatalf("waiter error = %v, want %v", got.Err, want)
	}
	for _, tc := range []struct {
		name string
		ch   <-chan error
	}{
		{name: "turn", ch: turnErr},
		{name: "login", ch: loginErr},
		{name: "goal", ch: goalErr},
		{name: "global", ch: globalErr},
	} {
		if err := <-tc.ch; !errors.Is(err, want) {
			t.Fatalf("%s blocked consumer error = %v, want %v", tc.name, err, want)
		}
	}
	if _, err := r.NextTurn("turn-1"); !errors.Is(err, want) {
		t.Fatalf("NextTurn() stable error = %v, want %v", err, want)
	}
	if _, err := r.NextLogin("login-1"); !errors.Is(err, want) {
		t.Fatalf("NextLogin() stable error = %v, want %v", err, want)
	}
	if _, err := r.NextGoal("thread-1"); !errors.Is(err, want) {
		t.Fatalf("NextGoal() stable error = %v, want %v", err, want)
	}
	if _, err := r.NextGlobal(); !errors.Is(err, want) {
		t.Fatalf("NextGlobal() stable error = %v, want %v", err, want)
	}
}

func TestRouterFailAllConcurrentWithUnregisterTurnDoesNotRace(t *testing.T) {
	const iterations = 200
	for i := 0; i < iterations; i++ {
		r := New(DefaultLimits())
		turnID := fmt.Sprintf("turn-%d", i)
		if err := r.RegisterTurn(turnID); err != nil {
			t.Fatalf("RegisterTurn(%s) error = %v", turnID, err)
		}
		if err := r.RouteNotification("turn/started", json.RawMessage(fmt.Sprintf(`{"turn":{"id":"%s"}}`, turnID))); err != nil {
			t.Fatalf("RouteNotification(%s) error = %v", turnID, err)
		}

		var started atomic.Int32
		var wg sync.WaitGroup
		wg.Add(2)
		go func() {
			defer wg.Done()
			started.Add(1)
			for started.Load() < 2 {
			}
			r.FailAll(errors.New("boom"))
		}()
		go func() {
			defer wg.Done()
			started.Add(1)
			for started.Load() < 2 {
			}
			r.UnregisterTurn(turnID)
		}()
		wg.Wait()
	}
}

func TestRouterNextGlobalReturnsStableFailureAfterFailAll(t *testing.T) {
	r := New(DefaultLimits())
	r.FailAll(nil)
	if _, err := r.NextGlobal(); err == nil || err.Error() != "router: failed" {
		t.Fatalf("NextGlobal() error = %v, want router failed", err)
	}
}

func TestGoalHelpers(t *testing.T) {
	if !goalRelevantMethod("turn/completed") || goalRelevantMethod("warning") {
		t.Fatal("goalRelevantMethod() mismatch")
	}
	raw := json.RawMessage(`{"threadId":"thread-1"}`)
	if got := notificationThreadID(raw); got != "thread-1" {
		t.Fatalf("notificationThreadID() = %q", got)
	}
	envelope, err := notificationEnvelope("thread/goal/updated", raw)
	if err != nil {
		t.Fatalf("notificationEnvelope() error = %v", err)
	}
	if string(envelope) != `{"method":"thread/goal/updated","params":{"threadId":"thread-1"}}` {
		t.Fatalf("notificationEnvelope() = %s", envelope)
	}
	if got := notificationThreadID(json.RawMessage(`nope`)); got != "" {
		t.Fatalf("notificationThreadID(invalid) = %q", got)
	}
	if got := notificationTurnID(json.RawMessage(`{"turnId":"turn-1"}`)); got != "turn-1" {
		t.Fatalf("notificationTurnID(turnId) = %q", got)
	}
	if got := notificationTurnID(json.RawMessage(`{"turn":{"id":"turn-2"}}`)); got != "turn-2" {
		t.Fatalf("notificationTurnID(turn.id) = %q", got)
	}
	if got := notificationTurnID(json.RawMessage(`oops`)); got != "" {
		t.Fatalf("notificationTurnID(invalid) = %q", got)
	}
	if got := notificationLoginID("wrong/method", json.RawMessage(`{"loginId":"x"}`)); got != "" {
		t.Fatalf("notificationLoginID(wrong method) = %q", got)
	}
	if got := notificationLoginID("account/login/completed", json.RawMessage(`oops`)); got != "" {
		t.Fatalf("notificationLoginID(invalid) = %q", got)
	}
}

func TestRouterFailedRegistrationAndNextGoalCancel(t *testing.T) {
	r := New(DefaultLimits())
	r.FailAll(errors.New("stop"))
	if err := r.RegisterTurn("turn-1"); err == nil {
		t.Fatal("RegisterTurn() error = nil after FailAll")
	}
	if err := r.RegisterGoal("thread-1"); err == nil {
		t.Fatal("RegisterGoal() error = nil after FailAll")
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := r.NextGoal("thread-1", ctx); !errors.Is(err, context.Canceled) && err == nil {
		t.Fatalf("NextGoal() error = %v", err)
	}
}

func TestRouterRegisterRouteIdempotentAndDismissedLateEvent(t *testing.T) {
	r := New(DefaultLimits())
	if err := r.RegisterGoal("thread-1"); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterGoal("thread-1"); err != nil {
		t.Fatalf("second RegisterGoal() error = %v", err)
	}
	r.UnregisterGoal("thread-1")
	if err := r.RouteNotification("thread/goal/updated", json.RawMessage(`{"threadId":"thread-1"}`)); err != nil {
		t.Fatalf("late dismissed goal notification error = %v", err)
	}
}

func TestNormalizeLimitsAllZeroBranches(t *testing.T) {
	got := normalizeLimits(Limits{})
	want := DefaultLimits()
	if got != want {
		t.Fatalf("normalizeLimits({}) = %#v, want %#v", got, want)
	}
}
