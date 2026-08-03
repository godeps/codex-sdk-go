package main

import codex "github.com/godeps/codex-sdk-go"

func main() {
	client := codex.NewCodex(codex.CodexOptions{})
	thread := client.StartThread(codex.ThreadOptions{})
	_, _ = thread.Run(codex.TextInput("Diagnose the failure"), codex.TurnOptions{})
}
