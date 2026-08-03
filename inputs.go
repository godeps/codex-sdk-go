package codex

import (
	"errors"
	"strings"
)

// DataURLImageInput creates a structured image input from a data URL.
func DataURLImageInput(url string) UserInput {
	return UserInput{Type: UserInputImage, URL: url}
}

// LocalImageInput creates a structured local-image input.
func LocalImageInput(path string) UserInput {
	return UserInput{Type: UserInputLocalImage, Path: path}
}

// SkillInput creates a structured skill reference.
func SkillInput(name, path string) UserInput {
	return UserInput{Type: UserInputSkill, Name: name, Path: path}
}

// MentionInput creates a structured resource mention.
func MentionInput(name, path string) UserInput {
	return UserInput{Type: UserInputMention, Name: name, Path: path}
}

func normalizeAppServerInput(input Input) ([]map[string]any, error) {
	if len(input.Items) == 0 {
		if input.Text == "" {
			return nil, errors.New("codex: input is empty")
		}
		return []map[string]any{{"type": "text", "text": input.Text}}, nil
	}

	wire := make([]map[string]any, 0, len(input.Items))
	for _, item := range input.Items {
		switch item.Type {
		case UserInputText:
			if item.Text == "" {
				return nil, errors.New("codex: text input is empty")
			}
			wire = append(wire, map[string]any{"type": "text", "text": item.Text})
		case UserInputImage:
			if !strings.HasPrefix(item.URL, "data:") {
				return nil, errors.New("codex: image input must be a data url")
			}
			wire = append(wire, map[string]any{"type": "image", "url": item.URL})
		case UserInputLocalImage:
			if item.Path == "" {
				return nil, errors.New("codex: local image path is empty")
			}
			wire = append(wire, map[string]any{"type": "localImage", "path": item.Path})
		case UserInputSkill:
			if item.Name == "" || item.Path == "" {
				return nil, errors.New("codex: skill input requires name and path")
			}
			wire = append(wire, map[string]any{"type": "skill", "name": item.Name, "path": item.Path})
		case UserInputMention:
			if item.Name == "" || item.Path == "" {
				return nil, errors.New("codex: mention input requires name and path")
			}
			wire = append(wire, map[string]any{"type": "mention", "name": item.Name, "path": item.Path})
		default:
			return nil, errors.New("codex: unsupported input type")
		}
	}
	return wire, nil
}
