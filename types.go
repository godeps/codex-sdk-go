package codex

import (
	"encoding/json"
	"time"
)

// Usage describes token usage for a turn. It mirrors the TS SDK's Usage
// (input/cached_input/cache_write_input/output/reasoning_output) plus the
// app-server's total_tokens.
//
// WIRE CASING: the real codex app-server emits camelCase (inputTokens,
// cachedInputTokens, ...), while the TS/exec JSONL surface and this SDK's own
// tests historically used snake_case (input_tokens, ...). UnmarshalJSON accepts
// BOTH so the field decodes correctly against the real CLI (camelCase) without
// breaking existing snake_case producers. Verified against real codex-cli
// 0.153.4 app-server frames (all camelCase).
type Usage struct {
	InputTokens           int `json:"input_tokens"`
	CachedInputTokens     int `json:"cached_input_tokens"`
	CacheWriteInputTokens int `json:"cache_write_input_tokens"`
	OutputTokens          int `json:"output_tokens"`
	ReasoningOutputTokens int `json:"reasoning_output_tokens"`
	TotalTokens           int `json:"total_tokens"`
}

// UnmarshalJSON accepts both camelCase (real app-server wire) and snake_case
// (exec JSONL / legacy) token-usage field names.
func (u *Usage) UnmarshalJSON(data []byte) error {
	var aux struct {
		InputTokens           *int `json:"input_tokens"`
		CachedInputTokens     *int `json:"cached_input_tokens"`
		CacheWriteInputTokens *int `json:"cache_write_input_tokens"`
		OutputTokens          *int `json:"output_tokens"`
		ReasoningOutputTokens *int `json:"reasoning_output_tokens"`
		TotalTokens           *int `json:"total_tokens"`

		InputTokensC           *int `json:"inputTokens"`
		CachedInputTokensC     *int `json:"cachedInputTokens"`
		CacheWriteInputTokensC *int `json:"cacheWriteInputTokens"`
		OutputTokensC          *int `json:"outputTokens"`
		ReasoningOutputTokensC *int `json:"reasoningOutputTokens"`
		TotalTokensC           *int `json:"totalTokens"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	pick := func(snake, camel *int) int {
		if snake != nil {
			return *snake
		}
		if camel != nil {
			return *camel
		}
		return 0
	}
	u.InputTokens = pick(aux.InputTokens, aux.InputTokensC)
	u.CachedInputTokens = pick(aux.CachedInputTokens, aux.CachedInputTokensC)
	u.CacheWriteInputTokens = pick(aux.CacheWriteInputTokens, aux.CacheWriteInputTokensC)
	u.OutputTokens = pick(aux.OutputTokens, aux.OutputTokensC)
	u.ReasoningOutputTokens = pick(aux.ReasoningOutputTokens, aux.ReasoningOutputTokensC)
	u.TotalTokens = pick(aux.TotalTokens, aux.TotalTokensC)
	return nil
}

// ThreadError indicates a failure in the stream.
type ThreadError struct {
	Message string `json:"message"`
	// CodexErrorInfo carries the machine-readable error classification the
	// real app-server emits (e.g. "unauthorized"). Observed on real codex-cli
	// 0.153.4 error frames: {"error":{"message":...,"codexErrorInfo":"unauthorized",...}}.
	CodexErrorInfo string `json:"codexErrorInfo,omitempty"`
}

// ThreadEvent represents a top-level JSONL event.
type ThreadEvent struct {
	Type     string       `json:"type"`
	Method   string       `json:"method,omitempty"`
	ThreadID string       `json:"thread_id,omitempty"`
	TurnID   string       `json:"turn_id,omitempty"`
	Usage    *Usage       `json:"usage,omitempty"`
	Error    *ThreadError `json:"error,omitempty"`
	Item     ThreadItem   `json:"item,omitempty"`
	Message  string       `json:"message,omitempty"`
	Turn     *TurnState   `json:"turn,omitempty"`
	Goal     *Goal        `json:"goal,omitempty"`
	Account  *Account     `json:"account,omitempty"`
	// ItemID identifies the item a streaming delta/progress event belongs to
	// (item.*.delta / item.mcp_tool_call.progress events).
	ItemID string `json:"item_id,omitempty"`
	// Delta carries the incremental text of a streaming delta event
	// (agent message / reasoning / command output / file change / plan).
	// The app-server delta family is the wire equivalent of the TS SDK's
	// item.updated events (finer-grained: token/output level).
	Delta string `json:"delta,omitempty"`
	// SummaryIndex is set on item.reasoning_summary_part.added events.
	SummaryIndex int             `json:"summary_index,omitempty"`
	Raw          json.RawMessage `json:"-"`
}

// CompactionResult records acknowledgement and observed completion of one
// provider-native thread compaction request.
type CompactionResult struct {
	ThreadID            string
	TurnID              string
	RequestAccepted     bool
	CompletionConfirmed bool
}

func (e *ThreadEvent) UnmarshalJSON(data []byte) error {
	var aux struct {
		Type     string          `json:"type"`
		ThreadID string          `json:"thread_id,omitempty"`
		Usage    *Usage          `json:"usage,omitempty"`
		Error    *ThreadError    `json:"error,omitempty"`
		Item     json.RawMessage `json:"item,omitempty"`
		Message  string          `json:"message,omitempty"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	e.Type = aux.Type
	e.ThreadID = aux.ThreadID
	e.Usage = aux.Usage
	e.Error = aux.Error
	e.Message = aux.Message
	e.Raw = append(json.RawMessage(nil), data...)
	if len(aux.Item) != 0 {
		item, err := parseThreadItem(aux.Item)
		if err != nil {
			return err
		}
		e.Item = item
	}
	return nil
}

// ThreadItem is a union of item payloads emitted by Codex.
type ThreadItem interface {
	ItemType() string
}

// TurnStatus is the runtime status of one turn.
type TurnStatus string

const (
	TurnStatusInProgress  TurnStatus = "in_progress"
	TurnStatusCompleted   TurnStatus = "completed"
	TurnStatusFailed      TurnStatus = "failed"
	TurnStatusInterrupted TurnStatus = "interrupted"
)

// TurnState describes one started or completed turn.
type TurnState struct {
	ID          string
	Status      TurnStatus
	Error       *ThreadError
	StartedAt   *time.Time
	CompletedAt *time.Time
	Duration    time.Duration
}

// TurnResult is the collected result returned by the pull-stream API.
type TurnResult struct {
	ID            string
	Status        TurnStatus
	Error         *ThreadError
	StartedAt     *time.Time
	CompletedAt   *time.Time
	Duration      time.Duration
	FinalResponse string
	Items         []ThreadItem
	Usage         *Usage
}

// ThreadStatus is the current state of a persisted thread.
type ThreadStatus string

const (
	ThreadStatusNotLoaded   ThreadStatus = "notLoaded"
	ThreadStatusIdle        ThreadStatus = "idle"
	ThreadStatusActive      ThreadStatus = "active"
	ThreadStatusSystemError ThreadStatus = "systemError"
)

// GoalStatus is the stored status of a thread goal.
type GoalStatus string

const (
	GoalStatusActive        GoalStatus = "active"
	GoalStatusPaused        GoalStatus = "paused"
	GoalStatusBlocked       GoalStatus = "blocked"
	GoalStatusUsageLimited  GoalStatus = "usageLimited"
	GoalStatusBudgetLimited GoalStatus = "budgetLimited"
	GoalStatusComplete      GoalStatus = "complete"
)

// Goal is the stored logical goal attached to a thread.
type Goal struct {
	Objective   string
	Status      GoalStatus
	TokenBudget int
	TokensUsed  int
	UpdatedAt   *time.Time
}

// Account is the current runtime authentication state.
type Account struct {
	Type string
	Raw  map[string]any
}

// AccountState is the result of reading the current runtime account state.
type AccountState struct {
	Account            *Account
	RequiresOpenAIAuth bool
}

// LoginResult is the terminal result of an interactive login attempt.
type LoginResult struct {
	LoginID string
	Account *Account
}

// ModelInfo is one entry returned by model/list.
type ModelInfo struct {
	ID                        string
	Model                     string
	DisplayName               string
	Description               string
	Hidden                    bool
	IsDefault                 bool
	DefaultServiceTier        string
	SupportedReasoningEfforts []string
	Raw                       map[string]any
}

// ModelPage is one page returned by model/list.
type ModelPage struct {
	Data       []ModelInfo
	NextCursor string
}

// ThreadRecord is a persisted thread snapshot returned by thread/list and thread/read.
type ThreadRecord struct {
	ID            string
	Name          string
	Path          string
	CWD           string
	Archived      bool
	Ephemeral     bool
	Status        ThreadStatus
	CurrentTurnID string
	Goal          *Goal
	Turns         []TurnRecord
	Raw           map[string]any
}

// TurnRecord is the persisted representation of one turn within a thread snapshot.
type TurnRecord struct {
	ID          string
	Status      TurnStatus
	Error       *ThreadError
	StartedAt   *time.Time
	CompletedAt *time.Time
	Duration    time.Duration
	Items       []ThreadItem
}

// ThreadPage is one page returned by thread/list.
type ThreadPage struct {
	Data            []ThreadRecord
	NextCursor      string
	BackwardsCursor string
}

// UnknownItem preserves forward-compatible item payloads emitted by newer Codex CLIs.
type UnknownItem struct {
	Type string          `json:"type"`
	Raw  json.RawMessage `json:"raw"`
}

func (i *UnknownItem) ItemType() string { return i.Type }

// AgentMessageItem is a response from the agent.
type AgentMessageItem struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Text  string `json:"text"`
	Phase string `json:"phase,omitempty"`
}

func (i *AgentMessageItem) ItemType() string { return i.Type }

// ReasoningItem captures the agent's reasoning summary.
type ReasoningItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}

func (i *ReasoningItem) ItemType() string { return i.Type }

// CommandExecutionStatus is the status of a command execution.
type CommandExecutionStatus string

const (
	CommandExecutionInProgress CommandExecutionStatus = "in_progress"
	CommandExecutionCompleted  CommandExecutionStatus = "completed"
	CommandExecutionFailed     CommandExecutionStatus = "failed"
	// Real app-server wire casing (codex-cli 0.153.4, protocol schema):
	// status enums are camelCase ("inProgress"), unlike the exec-JSONL
	// snake_case ("in_progress"). Both are accepted when decoding.
	CommandExecutionInProgressCamel CommandExecutionStatus = "inProgress"
)

// IsInProgress reports progress under either wire casing.
func (s CommandExecutionStatus) IsInProgress() bool {
	return s == CommandExecutionInProgress || s == CommandExecutionInProgressCamel
}

// CommandExecutionItem is a command executed by the agent.
//
// WIRE CASING: the real app-server emits camelCase field names and enums
// (aggregatedOutput/exitCode/inProgress); the exec JSONL surface uses
// snake_case (aggregated_output/exit_code/in_progress). UnmarshalJSON
// accepts both so the item decodes correctly against either producer.
type CommandExecutionItem struct {
	ID               string                 `json:"id"`
	Type             string                 `json:"type"`
	Command          string                 `json:"command"`
	AggregatedOutput string                 `json:"aggregated_output"`
	ExitCode         *int                   `json:"exit_code,omitempty"`
	Status           CommandExecutionStatus `json:"status"`
}

// UnmarshalJSON accepts both camelCase (real app-server) and snake_case
// (exec JSONL / legacy) field names.
func (i *CommandExecutionItem) UnmarshalJSON(data []byte) error {
	var aux struct {
		ID      string `json:"id"`
		Type    string `json:"type"`
		Command string `json:"command"`
		Status  string `json:"status"`

		AggregatedOutput  *string `json:"aggregated_output"`
		AggregatedOutputC *string `json:"aggregatedOutput"`
		ExitCode          *int    `json:"exit_code"`
		ExitCodeC         *int64  `json:"exitCode"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	i.ID = aux.ID
	i.Type = aux.Type
	i.Command = aux.Command
	i.Status = CommandExecutionStatus(aux.Status)
	if aux.AggregatedOutput != nil {
		i.AggregatedOutput = *aux.AggregatedOutput
	} else if aux.AggregatedOutputC != nil {
		i.AggregatedOutput = *aux.AggregatedOutputC
	}
	switch {
	case aux.ExitCode != nil:
		i.ExitCode = aux.ExitCode
	case aux.ExitCodeC != nil:
		v := int(*aux.ExitCodeC)
		i.ExitCode = &v
	}
	return nil
}

func (i *CommandExecutionItem) ItemType() string { return i.Type }

// PatchChangeKind indicates a file change type.
type PatchChangeKind string

const (
	PatchChangeAdd    PatchChangeKind = "add"
	PatchChangeDelete PatchChangeKind = "delete"
	PatchChangeUpdate PatchChangeKind = "update"
)

// FileUpdateChange describes a file change.
type FileUpdateChange struct {
	Path string          `json:"path"`
	Kind PatchChangeKind `json:"kind"`
}

// PatchApplyStatus is the status of a file change patch.
type PatchApplyStatus string

const (
	PatchApplyCompleted PatchApplyStatus = "completed"
	PatchApplyFailed    PatchApplyStatus = "failed"
	// Real app-server wire also emits "inProgress"/"declined" (protocol
	// schema PatchApplyStatus); exec JSONL only surfaces terminal states.
	PatchApplyInProgress PatchApplyStatus = "inProgress"
	PatchApplyDeclined   PatchApplyStatus = "declined"
)

// FileChangeItem describes a set of file changes.
type FileChangeItem struct {
	ID      string             `json:"id"`
	Type    string             `json:"type"`
	Changes []FileUpdateChange `json:"changes"`
	Status  PatchApplyStatus   `json:"status"`
}

func (i *FileChangeItem) ItemType() string { return i.Type }

// McpToolCallStatus is the status of an MCP tool call.
type McpToolCallStatus string

const (
	McpToolCallInProgress McpToolCallStatus = "in_progress"
	McpToolCallCompleted  McpToolCallStatus = "completed"
	McpToolCallFailed     McpToolCallStatus = "failed"
	// Real app-server wire casing (protocol schema): "inProgress"/"declined".
	McpToolCallInProgressCamel McpToolCallStatus = "inProgress"
	McpToolCallDeclined        McpToolCallStatus = "declined"
)

// IsInProgress reports progress under either wire casing.
func (s McpToolCallStatus) IsInProgress() bool {
	return s == McpToolCallInProgress || s == McpToolCallInProgressCamel
}

// McpToolCallResult holds a successful MCP tool response.
//
// WIRE CASING: the real app-server emits "structuredContent" (protocol
// schema); the exec JSONL surface uses "structured_content". Both accepted.
type McpToolCallResult struct {
	Content           []map[string]any `json:"content,omitempty"`
	StructuredContent any              `json:"structured_content,omitempty"`
}

// UnmarshalJSON accepts both camelCase (real app-server) and snake_case
// (exec JSONL / legacy) result field names.
func (r *McpToolCallResult) UnmarshalJSON(data []byte) error {
	var aux struct {
		Content            []map[string]any `json:"content"`
		StructuredContent  any              `json:"structured_content"`
		StructuredContentC any              `json:"structuredContent"`
	}
	if err := json.Unmarshal(data, &aux); err != nil {
		return err
	}
	r.Content = aux.Content
	if aux.StructuredContent != nil {
		r.StructuredContent = aux.StructuredContent
	} else {
		r.StructuredContent = aux.StructuredContentC
	}
	return nil
}

// McpToolCallError is an MCP tool error payload.
type McpToolCallError struct {
	Message string `json:"message"`
}

// McpToolCallItem represents a call to an MCP tool.
type McpToolCallItem struct {
	ID        string             `json:"id"`
	Type      string             `json:"type"`
	Server    string             `json:"server"`
	Tool      string             `json:"tool"`
	Arguments any                `json:"arguments"`
	Result    *McpToolCallResult `json:"result,omitempty"`
	Error     *McpToolCallError  `json:"error,omitempty"`
	Status    McpToolCallStatus  `json:"status"`
}

func (i *McpToolCallItem) ItemType() string { return i.Type }

// WebSearchItem captures a web search request.
type WebSearchItem struct {
	ID    string `json:"id"`
	Type  string `json:"type"`
	Query string `json:"query"`
}

func (i *WebSearchItem) ItemType() string { return i.Type }

// ErrorItem is a non-fatal error surfaced as an item.
type ErrorItem struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Message string `json:"message"`
}

func (i *ErrorItem) ItemType() string { return i.Type }

// TodoItem is a single entry in the agent's todo list.
type TodoItem struct {
	Text      string `json:"text"`
	Completed bool   `json:"completed"`
}

// TodoListItem tracks the agent's todo list.
type TodoListItem struct {
	ID    string     `json:"id"`
	Type  string     `json:"type"`
	Items []TodoItem `json:"items"`
}

func (i *TodoListItem) ItemType() string { return i.Type }

// UserMessageContentBlock is one content entry of a user message item.
// Real app-server frame: {"type":"text","text":"...","text_elements":[]}.
type UserMessageContentBlock struct {
	Type         string            `json:"type"`
	Text         string            `json:"text,omitempty"`
	TextElements []json.RawMessage `json:"text_elements,omitempty"`
}

// UserMessageItem is the user's input echoed back by the app-server as a
// thread item (observed on real codex-cli 0.153.4 item/started and
// item/completed frames). Not present in the TS SDK's ThreadItem union —
// app-server surface superset.
type UserMessageItem struct {
	ID       string                    `json:"id"`
	Type     string                    `json:"type"`
	ClientID *string                   `json:"clientId,omitempty"`
	Content  []UserMessageContentBlock `json:"content"`
}

func (i *UserMessageItem) ItemType() string { return i.Type }

// PlanItem carries the agent's plan text (app-server "plan" item).
type PlanItem struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Text string `json:"text"`
}

func (i *PlanItem) ItemType() string { return i.Type }

// WebSearchItem or other types parsed by item type string.
func parseThreadItem(raw json.RawMessage) (ThreadItem, error) {
	var base struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(raw, &base); err != nil {
		return nil, err
	}
	switch base.Type {
	case "agent_message", "agentMessage":
		var item AgentMessageItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "reasoning":
		var item ReasoningItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "command_execution", "commandExecution":
		var item CommandExecutionItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "file_change", "fileChange":
		var item FileChangeItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "mcp_tool_call", "mcpToolCall":
		var item McpToolCallItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "web_search", "webSearch":
		var item WebSearchItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "todo_list":
		var item TodoListItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "error":
		var item ErrorItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "userMessage", "user_message":
		var item UserMessageItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	case "plan":
		var item PlanItem
		if err := json.Unmarshal(raw, &item); err != nil {
			return nil, err
		}
		return &item, nil
	default:
		return &UnknownItem{
			Type: base.Type,
			Raw:  append(json.RawMessage(nil), raw...),
		}, nil
	}
}
