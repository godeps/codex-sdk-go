package codex

import "context"

// ApprovalMode controls the approval policy for tool executions.
type ApprovalMode string

const (
	ApprovalNever     ApprovalMode = "never"
	ApprovalOnRequest ApprovalMode = "on-request"
	ApprovalOnFailure ApprovalMode = "on-failure"
	ApprovalUntrusted ApprovalMode = "untrusted"
)

// ApprovalPreset is the high-level approval behavior used by the app-server API.
type ApprovalPreset string

const (
	ApprovalPresetDenyAll    ApprovalPreset = "deny_all"
	ApprovalPresetAutoReview ApprovalPreset = "auto_review"
)

// SandboxMode controls filesystem access for the Codex CLI.
type SandboxMode string

const (
	SandboxReadOnly         SandboxMode = "read-only"
	SandboxWorkspaceWrite   SandboxMode = "workspace-write"
	SandboxDangerFullAccess SandboxMode = "danger-full-access"
)

// ModelReasoningEffort configures the model reasoning effort.
type ModelReasoningEffort string

const (
	ReasoningMinimal ModelReasoningEffort = "minimal"
	ReasoningLow     ModelReasoningEffort = "low"
	ReasoningMedium  ModelReasoningEffort = "medium"
	ReasoningHigh    ModelReasoningEffort = "high"
	ReasoningXHigh   ModelReasoningEffort = "xhigh"
)

// WebSearchMode controls web search behavior.
type WebSearchMode string

const (
	WebSearchDisabled WebSearchMode = "disabled"
	WebSearchCached   WebSearchMode = "cached"
	WebSearchLive     WebSearchMode = "live"
)

// CodexOptions configures the Codex client.
type CodexOptions struct {
	CodexPathOverride string
	BaseURL           string
	APIKey            string
	RuntimeCacheRoot  string
	RuntimeVersion    string
	AllowPATH         bool
	// Config provides additional Codex CLI configuration overrides.
	// The SDK flattens nested objects into repeated --config dotted.path=TOML-value flags.
	Config map[string]any
	// Env overrides the environment passed to the Codex CLI process.
	// When provided, the SDK will not inherit variables from the parent process.
	Env map[string]string
}

// Option configures the context-first Client constructor.
type Option interface {
	applyCodexOption(*CodexOptions) error
}

type optionFunc func(*CodexOptions) error

func (f optionFunc) applyCodexOption(options *CodexOptions) error { return f(options) }

func (o CodexOptions) applyCodexOption(options *CodexOptions) error {
	*options = o
	return nil
}

// Personality is the runtime personality identifier for a thread or turn.
type Personality string

// ReasoningSummary controls how much reasoning summary the runtime should emit.
type ReasoningSummary string

const (
	ReasoningSummaryNone     ReasoningSummary = "none"
	ReasoningSummaryAuto     ReasoningSummary = "auto"
	ReasoningSummaryBrief    ReasoningSummary = "brief"
	ReasoningSummaryDetailed ReasoningSummary = "detailed"
)

// ThreadOptions configures a Codex thread.
type ThreadOptions struct {
	Config                map[string]any
	ApprovalPreset        *ApprovalPreset
	Model                 string
	ModelProvider         string
	SandboxMode           SandboxMode
	WorkingDirectory      string
	SkipGitRepoCheck      bool
	ModelReasoningEffort  ModelReasoningEffort
	NetworkAccessEnabled  *bool
	WebSearchMode         WebSearchMode
	WebSearchEnabled      *bool
	ApprovalPolicy        ApprovalMode
	AdditionalDirectories []string
	BaseInstructions      string
	DeveloperInstructions string
	Ephemeral             *bool
	Personality           Personality
	ServiceName           string
	ServiceTier           string
	SessionStartSource    string
	ThreadSource          map[string]any
}

// TurnOptions configures a single turn.
type TurnOptions struct {
	// OutputSchema is a JSON schema describing the expected agent output.
	OutputSchema any
	// Context controls cancellation for the turn.
	Context          context.Context
	ApprovalPreset   *ApprovalPreset
	ApprovalPolicy   ApprovalMode
	Model            string
	ReasoningEffort  ModelReasoningEffort
	WorkingDirectory string
	Personality      Personality
	SandboxMode      SandboxMode
	ServiceTier      string
	ReasoningSummary ReasoningSummary
}

// UserInputType represents the type of an input entry.
type UserInputType string

const (
	UserInputText       UserInputType = "text"
	UserInputLocalImage UserInputType = "local_image"
	UserInputImage      UserInputType = "image"
	UserInputSkill      UserInputType = "skill"
	UserInputMention    UserInputType = "mention"
)

// UserInput represents a structured input entry.
type UserInput struct {
	Type UserInputType `json:"type"`
	Text string        `json:"text,omitempty"`
	URL  string        `json:"url,omitempty"`
	Name string        `json:"name,omitempty"`
	Path string        `json:"path,omitempty"`
}

// Input describes a turn input. Provide Text or Items.
type Input struct {
	Text  string
	Items []UserInput
}

// TextInput creates an Input with a text prompt.
func TextInput(text string) Input {
	return Input{Text: text}
}

// ItemsInput creates an Input with structured entries.
func ItemsInput(items ...UserInput) Input {
	return Input{Items: items}
}

func (o TurnOptions) ContextOrBackground() context.Context {
	if o.Context != nil {
		return o.Context
	}
	return context.Background()
}

func WithCodexPath(path string) Option {
	return optionFunc(func(options *CodexOptions) error {
		options.CodexPathOverride = path
		return nil
	})
}

func WithBaseURL(baseURL string) Option {
	return optionFunc(func(options *CodexOptions) error {
		options.BaseURL = baseURL
		return nil
	})
}

func WithAPIKey(apiKey string) Option {
	return optionFunc(func(options *CodexOptions) error {
		options.APIKey = apiKey
		return nil
	})
}

func WithConfig(config map[string]any) Option {
	return optionFunc(func(options *CodexOptions) error {
		options.Config = cloneMap(config)
		return nil
	})
}

func WithEnv(env map[string]string) Option {
	return optionFunc(func(options *CodexOptions) error {
		if env == nil {
			options.Env = nil
			return nil
		}
		options.Env = make(map[string]string, len(env))
		for key, value := range env {
			options.Env[key] = value
		}
		return nil
	})
}

func WithRuntimeCacheRoot(root string) Option {
	return optionFunc(func(options *CodexOptions) error {
		options.RuntimeCacheRoot = root
		return nil
	})
}

func WithRuntimeVersion(version string) Option {
	return optionFunc(func(options *CodexOptions) error {
		options.RuntimeVersion = version
		return nil
	})
}

func WithAllowPATH(allow bool) Option {
	return optionFunc(func(options *CodexOptions) error {
		options.AllowPATH = allow
		return nil
	})
}

// ThreadListOptions controls thread/list queries.
type ThreadListOptions struct {
	Archived       *bool
	Cursor         string
	CWD            []string
	Limit          int
	ModelProviders []string
	SearchTerm     string
	SortDirection  string
	SortKey        string
	SourceKinds    []string
	UseStateDBOnly *bool
}
