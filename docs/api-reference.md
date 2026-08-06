# API Reference

This page is a compact reference for the current exported API. The `go doc -all .` output is the
source of truth for exact signatures and fields.

## Root Client

- `NewClient(ctx, opts ...Option) (*Client, error)`
- `Client.Metadata() *Metadata`
- `Client.Close() error`
- `Client.CloseContext(ctx context.Context) error`
- `Client.Wait() error`
- `Client.WaitContext(ctx context.Context) error`

Thread and runtime management:

- `Client.ListModels(ctx, includeHidden bool)`
- `Client.StartThread(ctx, options ThreadOptions)`
- `Client.ResumeThread(ctx, threadID, options)`
- `Client.ReadThread(ctx, threadID, includeTurns bool)`
- `Client.ListThreads(ctx, options ThreadListOptions)`
- `Client.ForkThread(ctx, threadID, options)`
- `Client.ArchiveThread(ctx, threadID)`
- `Client.UnarchiveThread(ctx, threadID)`
- `Client.SetThreadName(ctx, threadID, name)`
- `Client.CompactThread(ctx, threadID)`
- `Client.CompactThreadAndWait(ctx, threadID)`

Auth and goals:

- `Client.LoginAPIKey(ctx, apiKey)`
- `Client.LoginChatGPT(ctx)`
- `Client.LoginDeviceCode(ctx)`
- `Client.Account(ctx, refreshToken)`
- `Client.Logout(ctx)`
- `Client.GetGoal(ctx, threadID)`
- `Client.SetGoal(ctx, threadID, update)`
- `Client.ClearGoal(ctx, threadID)`
- `Client.PauseGoal(ctx, threadID)`
- `Client.StartGoal(ctx, threadID, objective)`

## Thread And Turn

- `Thread.RunContext(ctx, input, turnOptions)`
- `Thread.StartTurnContext(ctx, input, turnOptions)`
- `Thread.StartGoalContext(ctx, objective)`
- `Thread.ReadContext(ctx, includeTurns)`
- `Thread.SetNameContext(ctx, name)`
- `Thread.CompactContext(ctx)`
- `Thread.CompactContextAndWait(ctx)`
- `Thread.ArchiveContext(ctx)`
- `Thread.UnarchiveContext(ctx)`
- `Thread.ForkContext(ctx, options)`

`TurnHandle`:

- `ID() string`
- `Stream() (*TurnStream, error)`
- `StreamContext(ctx) (*TurnStream, error)`
- `RunContext(ctx) (*TurnResult, error)`
- `Steer(input Input) error`
- `SteerContext(ctx, input) error`
- `Interrupt() error`
- `InterruptContext(ctx) error`

`TurnStream`:

- `Next(ctx) (*ThreadEvent, error)`
- `Close() error`

## Goals

- `GoalHandle.StreamContext(ctx)`
- `GoalHandle.RunContext(ctx)`
- `GoalHandle.CancelContext(ctx)`
- `GoalHandle.Close()`
- `GoalStream.Next(ctx)`
- `GoalStream.Close()`

Goal values:

- `GoalStatusActive`
- `GoalStatusPaused`
- `GoalStatusBlocked`
- `GoalStatusUsageLimited`
- `GoalStatusBudgetLimited`
- `GoalStatusComplete`

## Login Handles

- `ChatGPTLoginHandle.WaitContext(ctx)`
- `ChatGPTLoginHandle.CancelContext(ctx)`
- `DeviceCodeLoginHandle.WaitContext(ctx)`
- `DeviceCodeLoginHandle.CancelContext(ctx)`

## Inputs

- `TextInput(text string) Input`
- `ItemsInput(items ...UserInput) Input`
- `DataURLImageInput(url string) UserInput`
- `LocalImageInput(path string) UserInput`
- `SkillInput(name, path string) UserInput`
- `MentionInput(name, path string) UserInput`

## Options

Client options:

- `WithCodexPath(path string)`
- `WithBaseURL(baseURL string)`
- `WithAPIKey(apiKey string)`
- `WithConfig(config map[string]any)`
- `WithEnv(env map[string]string)`
- `WithRuntimeCacheRoot(root string)`
- `WithRuntimeVersion(version string)`
- `WithAllowPATH(allow bool)`

Thread options:

- `ThreadOptions.Config`
- `ThreadOptions.ApprovalPreset`
- `ThreadOptions.Model`
- `ThreadOptions.ModelProvider`
- `ThreadOptions.SandboxMode`
- `ThreadOptions.WorkingDirectory`
- `ThreadOptions.SkipGitRepoCheck`
- `ThreadOptions.ModelReasoningEffort`
- `ThreadOptions.NetworkAccessEnabled`
- `ThreadOptions.WebSearchMode`
- `ThreadOptions.WebSearchEnabled`
- `ThreadOptions.ApprovalPolicy`
- `ThreadOptions.AdditionalDirectories`
- `ThreadOptions.BaseInstructions`
- `ThreadOptions.DeveloperInstructions`
- `ThreadOptions.Ephemeral`
- `ThreadOptions.Personality`
- `ThreadOptions.ServiceName`
- `ThreadOptions.ServiceTier`
- `ThreadOptions.SessionStartSource`
- `ThreadOptions.ThreadSource`

Turn options:

- `TurnOptions.OutputSchema`
- `TurnOptions.Context`
- `TurnOptions.ApprovalPreset`
- `TurnOptions.ApprovalPolicy`
- `TurnOptions.Model`
- `TurnOptions.ReasoningEffort`
- `TurnOptions.WorkingDirectory`
- `TurnOptions.Personality`
- `TurnOptions.SandboxMode`
- `TurnOptions.ServiceTier`
- `TurnOptions.ReasoningSummary`

## Errors And Retry

- `ErrTransportClosed`
- `ErrLimitExceeded`
- `ErrStreamClosed`
- `MapRPCError(code, message, data)`
- `IsRetryable(err)`
- `RetryOnOverload(ctx, options, op)`
- `DefaultRetryOptions()`
