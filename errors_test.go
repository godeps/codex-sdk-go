package codex

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

func TestMapRPCErrorMapsTypedClassesAndPreservesRPCFields(t *testing.T) {
	t.Parallel()

	source := json.RawMessage(`{"errorInfo":{"kind":"server_overloaded"}}`)
	err := MapRPCError(-32001, "busy", source)

	var busy *ServerBusyError
	if !errors.As(err, &busy) {
		t.Fatalf("MapRPCError() = %T, want ServerBusyError", err)
	}

	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatal("errors.As(..., *RPCError) = false")
	}
	if rpcErr.Code != -32001 || rpcErr.Message != "busy" {
		t.Fatalf("RPCError fields = %#v", rpcErr)
	}
	if string(rpcErr.Data) != string(source) {
		t.Fatalf("RPCError data = %s, want %s", rpcErr.Data, source)
	}

	source[2] = 'X'
	if string(rpcErr.Data) != `{"errorInfo":{"kind":"server_overloaded"}}` {
		t.Fatalf("RPCError data aliasing detected: %s", rpcErr.Data)
	}
	if !errors.Is(err, rpcErr) {
		t.Fatal("errors.Is(err, rpcErr) = false")
	}
}

func TestMapRPCErrorMapsServerRangeWithoutOverloadToServerRPCError(t *testing.T) {
	t.Parallel()

	err := MapRPCError(-32042, "plain server error", json.RawMessage(`{"detail":"no overload"}`))

	var serverErr *ServerRPCError
	if !errors.As(err, &serverErr) {
		t.Fatalf("MapRPCError() = %T, want ServerRPCError", err)
	}

	var rpcErr *RPCError
	if !errors.As(err, &rpcErr) {
		t.Fatal("errors.As(..., *RPCError) = false")
	}
	if rpcErr.Code != -32042 || rpcErr.Message != "plain server error" {
		t.Fatalf("RPCError fields = %#v", rpcErr)
	}
}

func TestMapRPCErrorMapsRetryLimitBeforeAndWithOverload(t *testing.T) {
	t.Parallel()

	for _, tc := range []struct {
		name string
		data string
	}{
		{name: "message only", data: `{"detail":"plain"}`},
		{name: "with overload", data: `{"codexErrorInfo":"server_overloaded"}`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			err := MapRPCError(-32002, "too many failed attempts", json.RawMessage(tc.data))

			var limit *RetryLimitExceededError
			if !errors.As(err, &limit) {
				t.Fatalf("MapRPCError() = %T, want RetryLimitExceededError", err)
			}
			var busy *ServerBusyError
			if errors.As(err, &busy) {
				t.Fatal("retry-limit error should remain its own typed class")
			}
		})
	}
}

func TestMapRPCErrorMapsJSONRPCStandardCodes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		code   int
		target any
	}{
		{name: "parse", code: -32700, target: new(*ParseError)},
		{name: "invalid request", code: -32600, target: new(*InvalidRequestError)},
		{name: "method not found", code: -32601, target: new(*MethodNotFoundError)},
		{name: "invalid params", code: -32602, target: new(*InvalidParamsError)},
		{name: "internal", code: -32603, target: new(*InternalRPCError)},
		{name: "outside server range", code: 42, target: new(*RPCError)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := MapRPCError(tt.code, "msg", json.RawMessage(`{"x":1}`))
			if !errors.As(err, tt.target) {
				t.Fatalf("MapRPCError() = %T, want target %T", err, tt.target)
			}
		})
	}
}

func TestIsRetryableMatchesPinnedPythonBehavior(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  error
		want bool
	}{
		{name: "server busy typed", err: &ServerBusyError{RPCError: &RPCError{Code: -32000}}, want: true},
		{name: "retry limit typed", err: &RetryLimitExceededError{RPCError: &RPCError{Code: -32000}}, want: true},
		{name: "raw rpc overload outside range", err: &RPCError{Code: 42, Data: json.RawMessage(`{"errorInfo":{"kind":"server_overloaded"}}`)}, want: true},
		{name: "server range without overload", err: &ServerRPCError{RPCError: &RPCError{Code: -32050, Data: json.RawMessage(`{"kind":"other"}`)}}, want: false},
		{name: "invalid params", err: &InvalidParamsError{RPCError: &RPCError{Code: -32602}}, want: false},
		{name: "non rpc", err: errors.New("boom"), want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsRetryable(tt.err); got != tt.want {
				t.Fatalf("IsRetryable(%T) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

func TestRetryOnOverloadRetriesOnlyRetryableErrors(t *testing.T) {
	t.Parallel()

	options := RetryOptions{
		MaxAttempts:  4,
		InitialDelay: time.Second,
		MaxDelay:     2 * time.Second,
		JitterRatio:  0,
	}

	t.Run("retries overloaded and returns value", func(t *testing.T) {
		attempts := 0
		var waits []time.Duration

		value, err := retryOnOverload(context.Background(), options, func(context.Context) (string, error) {
			attempts++
			if attempts < 3 {
				return "", MapRPCError(-32000, "busy", json.RawMessage(`"server_overloaded"`))
			}
			return "ok", nil
		}, func(_ context.Context, delay time.Duration) error {
			waits = append(waits, delay)
			return nil
		}, func() float64 { return 0.5 })
		if err != nil {
			t.Fatalf("retryOnOverload() error = %v", err)
		}
		if value != "ok" {
			t.Fatalf("retryOnOverload() value = %q, want ok", value)
		}
		if attempts != 3 {
			t.Fatalf("attempts = %d, want 3", attempts)
		}
		if len(waits) != 2 || waits[0] != time.Second || waits[1] != 2*time.Second {
			t.Fatalf("waits = %#v", waits)
		}
	})

	t.Run("retry limit remains retryable", func(t *testing.T) {
		attempts := 0
		var waits []time.Duration

		_, err := retryOnOverload(context.Background(), options, func(context.Context) (string, error) {
			attempts++
			return "", MapRPCError(-32000, "retry limit reached", json.RawMessage(`{"detail":"x"}`))
		}, func(_ context.Context, delay time.Duration) error {
			waits = append(waits, delay)
			return nil
		}, func() float64 { return 0.5 })

		var limit *RetryLimitExceededError
		if !errors.As(err, &limit) {
			t.Fatalf("retryOnOverload() error = %T, want RetryLimitExceededError", err)
		}
		if attempts != options.MaxAttempts {
			t.Fatalf("attempts = %d, want %d", attempts, options.MaxAttempts)
		}
		if len(waits) != options.MaxAttempts-1 {
			t.Fatalf("wait count = %d, want %d", len(waits), options.MaxAttempts-1)
		}
	})

	t.Run("does not retry non retryable errors", func(t *testing.T) {
		attempts := 0
		want := &InvalidParamsError{RPCError: &RPCError{Code: -32602, Message: "bad input"}}

		_, err := retryOnOverload(context.Background(), options, func(context.Context) (string, error) {
			attempts++
			return "", want
		}, func(context.Context, time.Duration) error {
			t.Fatal("wait should not be called for non-retryable error")
			return nil
		}, func() float64 { return 0.5 })
		if err != want {
			t.Fatalf("retryOnOverload() error identity changed: got %#v want %#v", err, want)
		}
		if attempts != 1 {
			t.Fatalf("attempts = %d, want 1", attempts)
		}
	})
}

func TestRetryOnOverloadUsesExponentialBackoffCapAndJitterBounds(t *testing.T) {
	t.Parallel()

	options := RetryOptions{
		MaxAttempts:  4,
		InitialDelay: time.Second,
		MaxDelay:     1500 * time.Millisecond,
		JitterRatio:  0.2,
	}

	t.Run("lower bound", func(t *testing.T) {
		var waits []time.Duration
		attempts := 0

		_, err := retryOnOverload(context.Background(), options, func(context.Context) (string, error) {
			attempts++
			return "", &ServerBusyError{RPCError: &RPCError{Code: -32000}}
		}, func(_ context.Context, delay time.Duration) error {
			waits = append(waits, delay)
			return nil
		}, func() float64 { return 0 })
		if err == nil {
			t.Fatal("retryOnOverload() error = nil, want terminal overload")
		}
		want := []time.Duration{800 * time.Millisecond, 1200 * time.Millisecond, 1200 * time.Millisecond}
		if len(waits) != len(want) {
			t.Fatalf("wait count = %d, want %d", len(waits), len(want))
		}
		for i := range want {
			if waits[i] != want[i] {
				t.Fatalf("wait[%d] = %v, want %v", i, waits[i], want[i])
			}
		}
	})

	t.Run("upper bound", func(t *testing.T) {
		var waits []time.Duration

		_, err := retryOnOverload(context.Background(), options, func(context.Context) (string, error) {
			return "", &ServerBusyError{RPCError: &RPCError{Code: -32000}}
		}, func(_ context.Context, delay time.Duration) error {
			waits = append(waits, delay)
			return nil
		}, func() float64 { return 1 })
		if err == nil {
			t.Fatal("retryOnOverload() error = nil, want terminal overload")
		}
		want := []time.Duration{1200 * time.Millisecond, 1800 * time.Millisecond, 1800 * time.Millisecond}
		if len(waits) != len(want) {
			t.Fatalf("wait count = %d, want %d", len(waits), len(want))
		}
		for i := range want {
			if waits[i] != want[i] {
				t.Fatalf("wait[%d] = %v, want %v", i, waits[i], want[i])
			}
		}
	})
}

func TestRetryOnOverloadReturnsContextCancellationFromSleep(t *testing.T) {
	t.Parallel()

	ctx, cancel := context.WithCancel(context.Background())
	options := DefaultRetryOptions()

	_, err := retryOnOverload(ctx, options, func(context.Context) (struct{}, error) {
		return struct{}{}, &ServerBusyError{RPCError: &RPCError{Code: -32000}}
	}, func(context.Context, time.Duration) error {
		cancel()
		return ctx.Err()
	}, func() float64 { return 0.5 })
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("retryOnOverload() error = %v, want context canceled", err)
	}
}

func TestRetryOnOverloadPreservesTerminalErrorIdentityAfterMaxAttempts(t *testing.T) {
	t.Parallel()

	options := RetryOptions{
		MaxAttempts:  3,
		InitialDelay: time.Second,
		MaxDelay:     time.Second,
		JitterRatio:  0,
	}
	want := &ServerBusyError{RPCError: &RPCError{Code: -32000, Message: "busy"}}
	attempts := 0

	_, err := retryOnOverload(context.Background(), options, func(context.Context) (string, error) {
		attempts++
		return "", want
	}, func(context.Context, time.Duration) error { return nil }, func() float64 { return 0.5 })

	if err != want {
		t.Fatalf("retryOnOverload() error identity changed: got %#v want %#v", err, want)
	}
	if attempts != options.MaxAttempts {
		t.Fatalf("attempts = %d, want %d", attempts, options.MaxAttempts)
	}
}

func TestRetryOnOverloadRejectsInvalidDependencies(t *testing.T) {
	t.Parallel()

	options := DefaultRetryOptions()
	noop := func(context.Context) (struct{}, error) { return struct{}{}, nil }

	tests := []struct {
		name   string
		ctx    context.Context
		op     func(context.Context) (struct{}, error)
		wait   retryWait
		random retryRand
		want   string
	}{
		{name: "nil context", op: noop, wait: realRetryWait, random: func() float64 { return 0.5 }, want: "nil context"},
		{name: "nil op", ctx: context.Background(), wait: realRetryWait, random: func() float64 { return 0.5 }, want: "nil retry operation"},
		{name: "nil wait", ctx: context.Background(), op: noop, random: func() float64 { return 0.5 }, want: "nil retry wait function"},
		{name: "nil random", ctx: context.Background(), op: noop, wait: realRetryWait, want: "nil retry random source"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := retryOnOverload(tt.ctx, options, tt.op, tt.wait, tt.random)
			if err == nil || err.Error() != "codex: "+tt.want {
				t.Fatalf("retryOnOverload() error = %v, want %q", err, "codex: "+tt.want)
			}
		})
	}
}
