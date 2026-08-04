package codex

import (
	"context"
	"encoding/json"
	"errors"
	"time"
)

// ReadContext loads the thread's persisted state from the app-server.
func (t *Thread) ReadContext(ctx context.Context, includeTurns bool) (*ThreadRecord, error) {
	if ctx == nil {
		return nil, errors.New("codex: nil context")
	}
	if err := t.ensurePrepared(ctx); err != nil {
		return nil, err
	}
	return t.rootClient().ReadThread(ctx, t.ID(), includeTurns)
}

// SetNameContext updates the persisted thread name.
func (t *Thread) SetNameContext(ctx context.Context, name string) error {
	if ctx == nil {
		return errors.New("codex: nil context")
	}
	if err := t.ensurePrepared(ctx); err != nil {
		return err
	}
	return t.rootClient().SetThreadName(ctx, t.ID(), name)
}

// CompactContext requests app-server compaction for the current thread.
func (t *Thread) CompactContext(ctx context.Context) error {
	if ctx == nil {
		return errors.New("codex: nil context")
	}
	if err := t.ensurePrepared(ctx); err != nil {
		return err
	}
	return t.rootClient().CompactThread(ctx, t.ID())
}

// ArchiveContext archives the current thread.
func (t *Thread) ArchiveContext(ctx context.Context) error {
	if err := t.ensurePrepared(ctx); err != nil {
		return err
	}
	return t.rootClient().ArchiveThread(ctx, t.ID())
}

// UnarchiveContext restores the current archived thread.
func (t *Thread) UnarchiveContext(ctx context.Context) error {
	if err := t.ensurePrepared(ctx); err != nil {
		return err
	}
	return t.rootClient().UnarchiveThread(ctx, t.ID())
}

// ForkContext creates a new thread from this thread.
func (t *Thread) ForkContext(ctx context.Context, options ThreadOptions) (*Thread, error) {
	if err := t.ensurePrepared(ctx); err != nil {
		return nil, err
	}
	return t.rootClient().ForkThread(ctx, t.ID(), options)
}

func decodeModelInfo(raw map[string]any) ModelInfo {
	model := ModelInfo{
		ID:                 stringField(raw, "id"),
		Model:              stringField(raw, "model"),
		DisplayName:        stringField(raw, "displayName"),
		Description:        stringField(raw, "description"),
		Hidden:             boolField(raw, "hidden"),
		IsDefault:          boolField(raw, "isDefault"),
		DefaultServiceTier: stringField(raw, "defaultServiceTier"),
		Raw:                cloneMap(raw),
	}
	if efforts, ok := raw["supportedReasoningEfforts"].([]any); ok {
		for _, effort := range efforts {
			if text, ok := effort.(string); ok {
				model.SupportedReasoningEfforts = append(model.SupportedReasoningEfforts, text)
			}
		}
	}
	return model
}

func decodeThreadRecord(raw map[string]any) ThreadRecord {
	record := ThreadRecord{
		ID:            stringField(raw, "id"),
		Name:          stringField(raw, "name"),
		Path:          stringField(raw, "path"),
		CWD:           stringField(raw, "cwd"),
		Archived:      boolField(raw, "archived"),
		Ephemeral:     boolField(raw, "ephemeral"),
		CurrentTurnID: stringField(raw, "currentTurnId"),
		Status:        decodeThreadStatus(raw["status"]),
		Goal:          decodeGoalFromAny(raw["goal"]),
		Raw:           cloneMap(raw),
	}
	if turns, ok := raw["turns"].([]any); ok {
		for _, turn := range turns {
			turnMap, ok := turn.(map[string]any)
			if !ok {
				continue
			}
			record.Turns = append(record.Turns, decodeTurnRecord(turnMap))
		}
	}
	return record
}

func decodeTurnRecord(raw map[string]any) TurnRecord {
	record := TurnRecord{
		ID:          stringField(raw, "id"),
		Status:      TurnStatus(stringField(raw, "status")),
		Error:       decodeThreadErrorFromAny(raw["error"]),
		StartedAt:   secondsField(raw, "startedAt"),
		CompletedAt: secondsField(raw, "completedAt"),
		Duration:    durationField(raw, "durationMs"),
	}
	if items, ok := raw["items"].([]any); ok {
		for _, item := range items {
			itemRaw, err := json.Marshal(item)
			if err != nil {
				continue
			}
			parsed, err := parseThreadItem(itemRaw)
			if err != nil {
				continue
			}
			record.Items = append(record.Items, parsed)
		}
	}
	return record
}

func decodeAccount(raw map[string]any) *Account {
	if raw == nil {
		return nil
	}
	return &Account{
		Type: stringField(raw, "type"),
		Raw:  cloneMap(raw),
	}
}

func decodeGoalFromAny(value any) *Goal {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return &Goal{
		Objective:   stringField(raw, "objective"),
		Status:      GoalStatus(stringField(raw, "status")),
		TokenBudget: intField(raw, "tokenBudget"),
		TokensUsed:  intField(raw, "tokensUsed"),
		UpdatedAt:   secondsField(raw, "updatedAt"),
	}
}

func decodeThreadStatus(value any) ThreadStatus {
	raw, ok := value.(map[string]any)
	if !ok {
		return ""
	}
	return ThreadStatus(stringField(raw, "type"))
}

func decodeThreadErrorFromAny(value any) *ThreadError {
	raw, ok := value.(map[string]any)
	if !ok {
		return nil
	}
	return &ThreadError{Message: stringField(raw, "message")}
}

func resolveApprovalSettings(preset *ApprovalPreset, legacy ApprovalMode, allowLegacy bool) (string, string) {
	if preset != nil {
		switch *preset {
		case ApprovalPresetDenyAll:
			return string(ApprovalNever), ""
		case ApprovalPresetAutoReview:
			return string(ApprovalOnRequest), "auto_review"
		}
	}
	if allowLegacy && legacy != "" {
		return string(legacy), ""
	}
	return "", ""
}

func cloneMap(source map[string]any) map[string]any {
	if len(source) == 0 {
		return nil
	}
	out := make(map[string]any, len(source))
	for key, value := range source {
		out[key] = value
	}
	return out
}

func stringField(raw map[string]any, key string) string {
	value, _ := raw[key].(string)
	return value
}

func boolField(raw map[string]any, key string) bool {
	value, _ := raw[key].(bool)
	return value
}

func intField(raw map[string]any, key string) int {
	switch value := raw[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	case int64:
		return int(value)
	default:
		return 0
	}
}

func secondsField(raw map[string]any, key string) *time.Time {
	switch value := raw[key].(type) {
	case float64:
		return unixSecondsTime(int64(value))
	case int64:
		return unixSecondsTime(value)
	case int:
		return unixSecondsTime(int64(value))
	default:
		return nil
	}
}

func durationField(raw map[string]any, key string) time.Duration {
	switch value := raw[key].(type) {
	case float64:
		return time.Duration(value) * time.Millisecond
	case int64:
		return time.Duration(value) * time.Millisecond
	case int:
		return time.Duration(value) * time.Millisecond
	default:
		return 0
	}
}
