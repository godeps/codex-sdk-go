package main

import codex "github.com/godeps/codex-sdk-go"

func main() {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"summary": map[string]any{"type": "string"},
		},
	}
	client := codex.NewCodex(codex.CodexOptions{})
	thread := client.StartThread(codex.ThreadOptions{})
	_, _ = thread.Run(codex.TextInput("Summarize"), codex.TurnOptions{OutputSchema: schema})
}
