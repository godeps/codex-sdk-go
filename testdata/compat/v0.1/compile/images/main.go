package main

import codex "github.com/godeps/codex-sdk-go"

func main() {
	client := codex.NewCodex(codex.CodexOptions{})
	thread := client.StartThread(codex.ThreadOptions{})
	_, _ = thread.Run(
		codex.ItemsInput(
			codex.UserInput{Type: codex.UserInputText, Text: "Describe the screenshots"},
			codex.UserInput{Type: codex.UserInputLocalImage, Path: "./one.png"},
			codex.UserInput{Type: codex.UserInputLocalImage, Path: "./two.jpg"},
		),
		codex.TurnOptions{},
	)
}
