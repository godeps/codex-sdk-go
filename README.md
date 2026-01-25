# Codex SDK (Go)

Embed the Codex agent in Go workflows and apps.

This SDK wraps the `codex` CLI and exchanges JSONL events over stdin/stdout.

## Features

- Start or resume Codex threads from Go.
- Stream JSONL events or wait for final responses.
- Provide JSON schema for structured responses.
- Attach local images alongside text prompts.
- Configure sandboxing, web search, and model settings.

## Requirements

- Go 1.25+ (module declares `go 1.25.5`).
- A `codex` CLI available on `PATH` or bundled alongside the module (see "Codex CLI resolution").

## Installation

```bash
go get github.com/godeps/codex-sdk-go
```

## Quickstart

```go
package main

import (
	"fmt"

	"github.com/godeps/codex-sdk-go"
)

func main() {
	client := codex.NewCodex(codex.CodexOptions{})
	thread := client.StartThread(codex.ThreadOptions{})

	turn, err := thread.Run(codex.TextInput("Diagnose the test failure and propose a fix"), codex.TurnOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println(turn.FinalResponse)
	fmt.Println(turn.Items)
}
```

Call `Run()` repeatedly on the same `Thread` instance to continue the conversation.

```go
nextTurn, err := thread.Run(codex.TextInput("Implement the fix"), codex.TurnOptions{})
```

## Streaming responses

`Run()` buffers events until the turn finishes. To react to intermediate progress, use `RunStreamed()`.

```go
streamed, err := thread.RunStreamed(
	codex.TextInput("Diagnose the test failure and propose a fix"),
	codex.TurnOptions{},
)
if err != nil {
	panic(err)
}

for event := range streamed.Events {
	switch event.Type {
	case "item.completed":
		fmt.Println("item", event.Item)
	case "turn.completed":
		fmt.Println("usage", event.Usage)
	}
}

if err := <-streamed.Done; err != nil {
	panic(err)
}
```

## Structured output

Codex can produce a JSON response that conforms to a specified schema. The schema must be a JSON object.

```go
schema := map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{"type": "string"},
		"status":  map[string]any{"type": "string", "enum": []any{"ok", "action_required"}},
	},
	"required":             []any{"summary", "status"},
	"additionalProperties": false,
}

turn, err := thread.Run(
	codex.TextInput("Summarize repository status"),
	codex.TurnOptions{OutputSchema: schema},
)
if err != nil {
	panic(err)
}
fmt.Println(turn.FinalResponse)
```

## Attaching images

Provide structured input entries when you need to include images alongside text.

```go
turn, err := thread.Run(
	codex.ItemsInput(
		codex.UserInput{Type: codex.UserInputText, Text: "Describe these screenshots"},
		codex.UserInput{Type: codex.UserInputLocalImage, Path: "./ui.png"},
		codex.UserInput{Type: codex.UserInputLocalImage, Path: "./diagram.jpg"},
	),
	codex.TurnOptions{},
)
```

## Thread management

Threads are persisted in `~/.codex/sessions`. Reconstruct them with `ResumeThread()` when needed.

```go
threadID := os.Getenv("CODEX_THREAD_ID")
thread := codex.ResumeThread(threadID, codex.ThreadOptions{})
_, err := thread.Run(codex.TextInput("Implement the fix"), codex.TurnOptions{})
```

You can access the current thread ID after the first turn starts:

```go
id := thread.ID()
```

## Configuration

### CodexOptions

```go
client := codex.NewCodex(codex.CodexOptions{
	CodexPathOverride: "/path/to/codex",
	BaseURL:           "https://api.openai.com",
	APIKey:            "your-api-key",
	Env: map[string]string{
		"PATH": "/usr/local/bin",
	},
})
```

Notes:

- `Env` fully overrides the environment passed to the CLI (the SDK will not inherit the parent process env).
- `BaseURL` and `APIKey` are mapped to `OPENAI_BASE_URL` and `CODEX_API_KEY` for the CLI process.

### API key configuration

You can provide the API key either via `CodexOptions` or environment variables.

```go
// Option 1: pass explicitly
client := codex.NewCodex(codex.CodexOptions{
	APIKey: "your-api-key",
})

// Option 2: use environment variable
// export CODEX_API_KEY=your-api-key
```

If you need a custom base URL, set `BaseURL` or export `OPENAI_BASE_URL`.

### ThreadOptions

```go
networkAccess := true
thread := client.StartThread(codex.ThreadOptions{
	Model:                 "your-model",
	SandboxMode:           codex.SandboxWorkspaceWrite,
	WorkingDirectory:      "/path/to/project",
	SkipGitRepoCheck:      true,
	ModelReasoningEffort:  codex.ReasoningMedium,
	NetworkAccessEnabled:  &networkAccess,
	WebSearchMode:         codex.WebSearchLive,
	ApprovalPolicy:        codex.ApprovalOnRequest,
	AdditionalDirectories: []string{"/path/to/extra"},
})
```

Supported values:

- `SandboxMode`: `read-only`, `workspace-write`, `danger-full-access`
- `ModelReasoningEffort`: `minimal`, `low`, `medium`, `high`, `xhigh`
- `WebSearchMode`: `disabled`, `cached`, `live`
- `ApprovalPolicy`: `never`, `on-request`, `on-failure`, `untrusted`

If you prefer a boolean toggle for web search, set `WebSearchEnabled` instead of `WebSearchMode`.

## Error handling and usage

- `Run()` returns an error when the turn fails or the CLI exits with an error.
- `Turn.Usage` includes `input_tokens`, `cached_input_tokens`, and `output_tokens` when available.

## Codex CLI resolution

The SDK resolves the CLI in this order:

1. `CodexPathOverride` if provided.
2. `codex` in `PATH`.
3. A bundled binary under `vendor/<target-triple>/codex/` within the module.

## License

See `LICENSE`.
