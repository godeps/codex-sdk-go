package codex

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

var (
	ErrTransportClosed = errors.New("codex: transport closed")
	ErrLimitExceeded   = errors.New("codex: limit exceeded")
	ErrStreamClosed    = errors.New("codex: stream closed")
)

// RPCError is an error returned by the app-server JSON-RPC endpoint.
type RPCError struct {
	Code    int
	Message string
	Data    json.RawMessage
}

func (e *RPCError) Error() string {
	if e == nil {
		return "codex: json-rpc error"
	}
	return fmt.Sprintf("codex: json-rpc error %d: %s", e.Code, e.Message)
}

type ParseError struct{ *RPCError }
type InvalidRequestError struct{ *RPCError }
type MethodNotFoundError struct{ *RPCError }
type InvalidParamsError struct{ *RPCError }
type InternalRPCError struct{ *RPCError }
type ServerRPCError struct{ *RPCError }
type ServerBusyError struct{ *RPCError }
type RetryLimitExceededError struct{ *RPCError }

func (e *ParseError) Unwrap() error              { return e.RPCError }
func (e *InvalidRequestError) Unwrap() error     { return e.RPCError }
func (e *MethodNotFoundError) Unwrap() error     { return e.RPCError }
func (e *InvalidParamsError) Unwrap() error      { return e.RPCError }
func (e *InternalRPCError) Unwrap() error        { return e.RPCError }
func (e *ServerRPCError) Unwrap() error          { return e.RPCError }
func (e *ServerBusyError) Unwrap() error         { return e.RPCError }
func (e *RetryLimitExceededError) Unwrap() error { return e.RPCError }

// MapRPCError maps a JSON-RPC error to the most specific public error type.
func MapRPCError(code int, message string, data json.RawMessage) error {
	rpcErr := &RPCError{Code: code, Message: message, Data: append(json.RawMessage(nil), data...)}
	switch code {
	case -32700:
		return &ParseError{RPCError: rpcErr}
	case -32600:
		return &InvalidRequestError{RPCError: rpcErr}
	case -32601:
		return &MethodNotFoundError{RPCError: rpcErr}
	case -32602:
		return &InvalidParamsError{RPCError: rpcErr}
	case -32603:
		return &InternalRPCError{RPCError: rpcErr}
	}
	if code >= -32099 && code <= -32000 {
		if containsRetryLimit(message) {
			return &RetryLimitExceededError{RPCError: rpcErr}
		}
		if containsServerOverloaded(data) {
			return &ServerBusyError{RPCError: rpcErr}
		}
		return &ServerRPCError{RPCError: rpcErr}
	}
	return rpcErr
}

// IsRetryable reports whether err represents a transient server-overload failure.
func IsRetryable(err error) bool {
	var busy *ServerBusyError
	if errors.As(err, &busy) {
		return true
	}
	var limit *RetryLimitExceededError
	if errors.As(err, &limit) {
		return true
	}
	var rpcErr *RPCError
	return errors.As(err, &rpcErr) && containsServerOverloaded(rpcErr.Data)
}

func containsRetryLimit(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "retry limit") || strings.Contains(message, "too many failed attempts")
}

func containsServerOverloaded(data json.RawMessage) bool {
	if len(data) == 0 || string(data) == "null" {
		return false
	}
	var value any
	if err := json.Unmarshal(data, &value); err != nil {
		return false
	}
	return valueContainsServerOverloaded(value)
}

func valueContainsServerOverloaded(value any) bool {
	switch value := value.(type) {
	case string:
		return strings.EqualFold(value, "server_overloaded")
	case []any:
		for _, item := range value {
			if valueContainsServerOverloaded(item) {
				return true
			}
		}
	case map[string]any:
		for _, item := range value {
			if valueContainsServerOverloaded(item) {
				return true
			}
		}
	}
	return false
}
