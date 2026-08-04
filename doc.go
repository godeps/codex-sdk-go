// Package codex provides a context-first Go client for the Codex app-server.
//
// Use one long-lived Client per process, then create or resume threads and run
// turns with explicit context.Context values. The public surface centers on
// Client, Thread, TurnHandle, and GoalHandle rather than a per-turn process
// wrapper.
//
// For end-to-end runnable programs, see the example directories in the
// repository root. For focused executable snippets that compile under go test,
// see example_test.go.
package codex
