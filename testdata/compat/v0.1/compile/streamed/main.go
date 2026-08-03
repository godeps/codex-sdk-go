package main

import codex "github.com/godeps/codex-sdk-go"

func main() {
	client := codex.NewCodex(codex.CodexOptions{})
	thread := client.StartThread(codex.ThreadOptions{})
	streamed, _ := thread.RunStreamed(codex.TextInput("Stream progress"), codex.TurnOptions{})
	if streamed == nil {
		return
	}
	for range streamed.Events {
	}
	_ = <-streamed.Done
}
