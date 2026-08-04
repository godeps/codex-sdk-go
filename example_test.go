package codex_test

import (
	"fmt"

	codex "github.com/godeps/codex-sdk-go"
)

func ExampleTextInput() {
	input := codex.TextInput("Summarize repository status")

	fmt.Println(input.Text)
	fmt.Println(len(input.Items))

	// Output:
	// Summarize repository status
	// 0
}

func ExampleItemsInput() {
	input := codex.ItemsInput(
		codex.SkillInput("golang-documentation", "/workspace/.codex/skills/golang-documentation"),
		codex.MentionInput("README", "README.md"),
	)

	fmt.Println(len(input.Items))
	fmt.Println(input.Items[0].Type)
	fmt.Println(input.Items[1].Path)

	// Output:
	// 2
	// skill
	// README.md
}

func ExampleThreadOptions() {
	options := codex.ThreadOptions{
		WorkingDirectory: "/workspace/project",
		SandboxMode:      codex.SandboxWorkspaceWrite,
		ApprovalPolicy:   codex.ApprovalOnRequest,
	}

	fmt.Println(options.WorkingDirectory)
	fmt.Println(options.SandboxMode)
	fmt.Println(options.ApprovalPolicy)

	// Output:
	// /workspace/project
	// workspace-write
	// on-request
}
