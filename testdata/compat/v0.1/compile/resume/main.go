package main

import codex "github.com/godeps/codex-sdk-go"

func main() {
	client := codex.NewCodex(codex.CodexOptions{})
	thread := client.ResumeThread("thread-123", codex.ThreadOptions{})
	_, _ = thread.Run(codex.TextInput("Continue"), codex.TurnOptions{})
	_ = thread.ID()
}
