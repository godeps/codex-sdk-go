package codex

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCompatibilityItemsInputMapsTextToStdinAndImagesToArgs(t *testing.T) {
	t.Parallel()

	captures := newCompatCaptures(t)
	script := writeCompatCodexScript(t, captures, compatScriptEvents{
		Lines: []string{
			`{"type":"thread.started","thread_id":"thread-inputs"}`,
			`{"type":"turn.completed","usage":{"input_tokens":1,"cached_input_tokens":0,"output_tokens":1}}`,
		},
	})

	thread := NewThread(NewCodexExec(script, nil), CodexOptions{}, ThreadOptions{}, "")
	_, err := thread.Run(
		ItemsInput(
			UserInput{Type: UserInputText, Text: "first"},
			UserInput{Type: UserInputLocalImage, Path: "one.png"},
			UserInput{Type: UserInputText, Text: "second"},
			UserInput{Type: UserInputLocalImage, Path: "two.jpg"},
		),
		TurnOptions{},
	)
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if got := strings.TrimSpace(captures.stdin(t)); got != strings.TrimSpace(readCompatFile(t, filepath.Join("transcripts", "stdin-items.txt"))) {
		t.Fatalf("stdin transcript mismatch\n--- got ---\n%s\n--- want ---\n%s", got, readCompatFile(t, filepath.Join("transcripts", "stdin-items.txt")))
	}
	if got := captures.args(t); !reflect.DeepEqual(got, []string{"exec", "--experimental-json", "--image", "one.png", "--image", "two.jpg"}) {
		t.Fatalf("unexpected args: %#v", got)
	}
}

func TestCompatibilityRunCollectsItemsUsageAndFinalResponse(t *testing.T) {
	t.Parallel()

	captures := newCompatCaptures(t)
	script := writeCompatCodexScript(t, captures, compatScriptEvents{
		Lines: []string{
			readCompatFileTrimmed(t, filepath.Join("transcripts", "jsonl-success.jsonl"), 0),
			readCompatFileTrimmed(t, filepath.Join("transcripts", "jsonl-success.jsonl"), 1),
			readCompatFileTrimmed(t, filepath.Join("transcripts", "jsonl-success.jsonl"), 2),
			readCompatFileTrimmed(t, filepath.Join("transcripts", "jsonl-success.jsonl"), 3),
		},
	})

	thread := NewThread(NewCodexExec(script, nil), CodexOptions{}, ThreadOptions{}, "")
	turn, err := thread.Run(TextInput("hello"), TurnOptions{})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	if thread.ID() != "thread-success-1" {
		t.Fatalf("unexpected thread ID: %q", thread.ID())
	}
	if turn.FinalResponse != "final answer" {
		t.Fatalf("unexpected final response: %q", turn.FinalResponse)
	}
	if turn.Usage == nil || *turn.Usage != (Usage{InputTokens: 3, CachedInputTokens: 1, OutputTokens: 5}) {
		t.Fatalf("unexpected usage: %#v", turn.Usage)
	}
	if len(turn.Items) != 2 {
		t.Fatalf("unexpected item count: %d", len(turn.Items))
	}
	if _, ok := turn.Items[0].(*ReasoningItem); !ok {
		t.Fatalf("expected reasoning item, got %T", turn.Items[0])
	}
	msg, ok := turn.Items[1].(*AgentMessageItem)
	if !ok {
		t.Fatalf("expected agent message item, got %T", turn.Items[1])
	}
	if msg.Text != "final answer" {
		t.Fatalf("unexpected final item text: %q", msg.Text)
	}
}

func TestCompatibilityRunReturnsTurnFailedMessage(t *testing.T) {
	t.Parallel()

	script := writeCompatCodexScript(t, newCompatCaptures(t), compatScriptEvents{
		Lines: []string{
			`{"type":"thread.started","thread_id":"thread-failed"}`,
			`{"type":"turn.failed","error":{"message":"turn failed"}}`,
		},
	})

	thread := NewThread(NewCodexExec(script, nil), CodexOptions{}, ThreadOptions{}, "")
	_, err := thread.Run(TextInput("hello"), TurnOptions{})
	if err == nil || err.Error() != "turn failed" {
		t.Fatalf("expected turn failure, got %v", err)
	}
}

func TestCompatibilityRunStreamedPreservesSchemaLifetimeDuringRun(t *testing.T) {
	t.Parallel()

	captures := newCompatCaptures(t)
	script := writeCompatCodexScript(t, captures, compatScriptEvents{
		Lines: []string{
			`{"type":"thread.started","thread_id":"thread-schema"}`,
			`{"type":"turn.completed","usage":{"input_tokens":1,"cached_input_tokens":0,"output_tokens":1}}`,
		},
	})

	thread := NewThread(NewCodexExec(script, nil), CodexOptions{}, ThreadOptions{}, "")
	streamed, err := thread.RunStreamed(TextInput("schema"), TurnOptions{
		OutputSchema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"answer": map[string]any{"type": "string"},
			},
		},
	})
	if err != nil {
		t.Fatalf("RunStreamed: %v", err)
	}
	for range streamed.Events {
	}
	if err := <-streamed.Done; err != nil {
		t.Fatalf("stream done: %v", err)
	}

	if status := strings.TrimSpace(captures.schemaStatus(t)); status != "present" {
		t.Fatalf("expected schema file to exist during run, got %q", status)
	}
	if got := strings.TrimSpace(captures.schemaJSON(t)); got != strings.TrimSpace(readCompatFile(t, filepath.Join("transcripts", "structured-output-schema.json"))) {
		t.Fatalf("unexpected schema payload:\n got: %s\nwant: %s", got, readCompatFile(t, filepath.Join("transcripts", "structured-output-schema.json")))
	}
	path := strings.TrimSpace(captures.schemaPath(t))
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("expected schema path removed after completion, stat err=%v", err)
	}
}

type compatCaptures struct {
	root string
}

func newCompatCaptures(t *testing.T) compatCaptures {
	t.Helper()

	root := t.TempDir()
	return compatCaptures{root: root}
}

func (c compatCaptures) args(t *testing.T) []string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(c.root, "args.json"))
	if err != nil {
		t.Fatalf("read args capture: %v", err)
	}
	var out []string
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("decode args capture: %v", err)
	}
	return out
}

func (c compatCaptures) stdin(t *testing.T) string {
	t.Helper()
	return c.mustReadString(t, "stdin.txt")
}

func (c compatCaptures) schemaStatus(t *testing.T) string {
	t.Helper()
	return c.mustReadString(t, "schema-status.txt")
}

func (c compatCaptures) schemaJSON(t *testing.T) string {
	t.Helper()
	return c.mustReadString(t, "schema.json.txt")
}

func (c compatCaptures) schemaPath(t *testing.T) string {
	t.Helper()
	return c.mustReadString(t, "schema-path.txt")
}

func (c compatCaptures) mustReadString(t *testing.T, name string) string {
	t.Helper()

	data, err := os.ReadFile(filepath.Join(c.root, name))
	if err != nil {
		t.Fatalf("read capture %s: %v", name, err)
	}
	return string(data)
}

type compatScriptEvents struct {
	Lines []string
}

func writeCompatCodexScript(t *testing.T, captures compatCaptures, events compatScriptEvents) string {
	t.Helper()

	lines, err := json.Marshal(events.Lines)
	if err != nil {
		t.Fatalf("marshal lines: %v", err)
	}

	script := "#!/usr/bin/env python3\n" +
		"import json, os, pathlib, sys\n" +
		"capture = pathlib.Path(os.environ['COMPAT_CAPTURE_DIR'])\n" +
		"capture.mkdir(parents=True, exist_ok=True)\n" +
		"capture.joinpath('args.json').write_text(json.dumps(sys.argv[1:]))\n" +
		"capture.joinpath('stdin.txt').write_text(sys.stdin.read())\n" +
		"schema_path = ''\n" +
		"for i, arg in enumerate(sys.argv[1:]):\n" +
		"    if arg == '--output-schema' and i + 2 <= len(sys.argv[1:]):\n" +
		"        schema_path = sys.argv[1:][i + 1]\n" +
		"        break\n" +
		"capture.joinpath('schema-path.txt').write_text(schema_path)\n" +
		"if schema_path and os.path.exists(schema_path):\n" +
		"    capture.joinpath('schema-status.txt').write_text('present')\n" +
		"    capture.joinpath('schema.json.txt').write_text(pathlib.Path(schema_path).read_text())\n" +
		"else:\n" +
		"    capture.joinpath('schema-status.txt').write_text('missing')\n" +
		"    capture.joinpath('schema.json.txt').write_text('')\n" +
		"for line in json.loads(" + strconvQuote(string(lines)) + "):\n" +
		"    print(line)\n"

	path := filepath.Join(t.TempDir(), "fake-codex")
	if runtime.GOOS == "windows" {
		path += ".py"
	}
	if err := writeExecutableFixture(path, []byte(script)); err != nil {
		t.Fatalf("write fake codex script: %v", err)
	}

	wrapper := path
	if runtime.GOOS != "windows" {
		wrapper = filepath.Join(t.TempDir(), "codex")
		if err := writeExecutableFixture(wrapper, []byte("#!/bin/sh\nCOMPAT_CAPTURE_DIR="+shellQuote(captures.root)+" exec "+shellQuote(path)+" \"$@\"\n")); err != nil {
			t.Fatalf("write wrapper: %v", err)
		}
	} else {
		t.Setenv("COMPAT_CAPTURE_DIR", captures.root)
	}
	time.Sleep(20 * time.Millisecond)
	return wrapper
}

func readCompatFileTrimmed(t *testing.T, rel string, index int) string {
	t.Helper()

	lines := strings.Split(strings.TrimSpace(readCompatFile(t, rel)), "\n")
	return lines[index]
}

func shellQuote(path string) string {
	return "'" + strings.ReplaceAll(path, "'", "'\"'\"'") + "'"
}

func strconvQuote(s string) string {
	b, _ := json.Marshal(s)
	return string(b)
}

func writeExecutableFixture(path string, contents []byte) error {
	dir := filepath.Dir(path)
	temp, err := os.CreateTemp(dir, ".tmp-exec-*")
	if err != nil {
		return err
	}
	tempPath := temp.Name()
	if _, err := temp.Write(contents); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := temp.Sync(); err != nil {
		_ = temp.Close()
		_ = os.Remove(tempPath)
		return err
	}
	if err := temp.Close(); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Chmod(tempPath, 0o755); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		_ = os.Remove(tempPath)
		return err
	}
	return nil
}
