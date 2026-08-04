package codex

import (
	"context"
	"errors"
	"path/filepath"
	"testing"
)

func TestThreadMethodsDoNotRematerializeStartedThread(t *testing.T) {
	server, captures := writeContextAPIServer(t)
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	thread, err := client.StartThread(context.Background(), ThreadOptions{})
	if err != nil {
		t.Fatalf("StartThread: %v", err)
	}
	if err := thread.SetNameContext(context.Background(), "kept"); err != nil {
		t.Fatalf("SetNameContext: %v", err)
	}
	if _, err := thread.ReadContext(context.Background(), false); err != nil {
		t.Fatalf("ReadContext: %v", err)
	}
	if err := thread.CompactContext(context.Background()); err != nil {
		t.Fatalf("CompactContext: %v", err)
	}
	if _, err := thread.RunContext(context.Background(), TextInput("hello"), TurnOptions{}); err != nil {
		t.Fatalf("RunContext: %v", err)
	}

	if got := countCaptureMatches(t, captures.root, "thread_start_*.json"); got != 1 {
		t.Fatalf("thread/start count = %d, want 1", got)
	}
	if got := countCaptureMatches(t, captures.root, "thread_resume_*.json"); got != 0 {
		t.Fatalf("thread/resume count = %d, want 0", got)
	}
}

func TestThreadMethodsDoNotRematerializeResumedThread(t *testing.T) {
	server, captures := writeContextAPIServer(t)
	client, err := NewClient(context.Background(), WithCodexPath(server))
	if err != nil {
		t.Fatalf("NewClient: %v", err)
	}
	defer func() { _ = client.Close() }()

	thread, err := client.ResumeThread(context.Background(), "thread-ctx-1", ThreadOptions{})
	if err != nil {
		t.Fatalf("ResumeThread: %v", err)
	}
	if _, err := thread.ReadContext(context.Background(), true); err != nil {
		t.Fatalf("ReadContext: %v", err)
	}
	if err := thread.SetNameContext(context.Background(), "resumed"); err != nil {
		t.Fatalf("SetNameContext: %v", err)
	}
	if _, err := thread.ForkContext(context.Background(), ThreadOptions{}); err != nil {
		t.Fatalf("ForkContext: %v", err)
	}
	if _, err := thread.RunContext(context.Background(), TextInput("hello"), TurnOptions{}); err != nil {
		t.Fatalf("RunContext: %v", err)
	}

	if got := countCaptureMatches(t, captures.root, "thread_resume_*.json"); got != 1 {
		t.Fatalf("thread/resume count = %d, want 1", got)
	}
	if got := countCaptureMatches(t, captures.root, "thread_start_*.json"); got != 0 {
		t.Fatalf("thread/start count = %d, want 0", got)
	}
}

func TestZeroValueThreadStillReturnsTransportClosed(t *testing.T) {
	var thread Thread
	if thread.ID() != "" {
		t.Fatalf("ID() = %q, want empty", thread.ID())
	}
	if _, err := thread.ReadContext(context.Background(), false); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("ReadContext() error = %v", err)
	}
	if _, err := thread.StartTurnContext(context.Background(), TextInput("hello"), TurnOptions{}); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("StartTurnContext() error = %v", err)
	}
	if _, err := thread.ForkContext(context.Background(), ThreadOptions{}); !errors.Is(err, ErrTransportClosed) {
		t.Fatalf("ForkContext() error = %v", err)
	}
}

func countCaptureMatches(t *testing.T, root string, pattern string) int {
	t.Helper()

	matches, err := filepath.Glob(filepath.Join(root, pattern))
	if err != nil {
		t.Fatalf("Glob(%q): %v", pattern, err)
	}
	return len(matches)
}
