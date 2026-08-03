package codex

import (
	"reflect"
	"testing"
)

func TestNormalizeAppServerInput(t *testing.T) {
	input := ItemsInput(
		UserInput{Type: UserInputText, Text: "hello"},
		DataURLImageInput("data:image/png;base64,AA=="),
		LocalImageInput("/tmp/image.png"),
		SkillInput("review", "/skills/review"),
		MentionInput("spec", "/docs/spec.md"),
	)
	got, err := normalizeAppServerInput(input)
	if err != nil {
		t.Fatal(err)
	}
	want := []map[string]any{
		{"type": "text", "text": "hello"},
		{"type": "image", "url": "data:image/png;base64,AA=="},
		{"type": "localImage", "path": "/tmp/image.png"},
		{"type": "skill", "name": "review", "path": "/skills/review"},
		{"type": "mention", "name": "spec", "path": "/docs/spec.md"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("normalizeAppServerInput() = %#v, want %#v", got, want)
	}
}

func TestNormalizeAppServerInputRejectsInvalid(t *testing.T) {
	tests := []struct {
		name  string
		input Input
	}{
		{name: "empty"},
		{name: "invalid image", input: ItemsInput(DataURLImageInput("https://example.com/image.png"))},
		{name: "empty local image", input: ItemsInput(LocalImageInput(""))},
		{name: "empty skill", input: ItemsInput(SkillInput("", ""))},
		{name: "unknown", input: ItemsInput(UserInput{Type: "future"})},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if _, err := normalizeAppServerInput(tt.input); err == nil {
				t.Fatal("normalizeAppServerInput() error = nil")
			}
		})
	}
}
