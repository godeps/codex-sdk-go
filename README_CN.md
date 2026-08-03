# Codex SDK（Go）

在 Go 工作流和应用中嵌入 Codex 代理。

该 SDK 封装了 `codex` CLI，并通过 stdin/stdout 交换 JSONL 事件。

## 特性

- 从 Go 启动或恢复 Codex 线程。
- 流式处理 JSONL 事件或等待最终响应。
- 提供 JSON schema 以获得结构化响应。
- 在文本提示中附带本地图片。
- 配置沙箱、联网搜索和模型设置。
- 透传额外的 Codex CLI `--config` 覆盖项。
- 对新版 Codex CLI 的未知流式 item 类型保持兼容。

## 要求

- Go 1.25+（模块声明 `go 1.25.5`）。
- `codex` CLI 可在 `PATH` 中找到，或与模块一起打包（见“Codex CLI 解析顺序”）。

## 安装

```bash
go get github.com/godeps/codex-sdk-go
```

## 快速开始

```go
package main

import (
	"fmt"

	"github.com/godeps/codex-sdk-go"
)

func main() {
	client := codex.NewCodex(codex.CodexOptions{})
	thread := client.StartThread(codex.ThreadOptions{})

	turn, err := thread.Run(codex.TextInput("诊断测试失败并给出修复方案"), codex.TurnOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println(turn.FinalResponse)
	fmt.Println(turn.Items)
}
```

在同一个 `Thread` 实例上反复调用 `Run()` 以继续对话。

```go
nextTurn, err := thread.Run(codex.TextInput("实现修复"), codex.TurnOptions{})
```

## 流式响应

`Run()` 会缓冲事件直到该轮结束。若需处理中间进度，请使用 `RunStreamed()`。

```go
streamed, err := thread.RunStreamed(
	codex.TextInput("诊断测试失败并给出修复方案"),
	codex.TurnOptions{},
)
if err != nil {
	panic(err)
}

for event := range streamed.Events {
	switch event.Type {
	case "item.completed":
		fmt.Println("item", event.Item)
	case "turn.completed":
		fmt.Println("usage", event.Usage)
	}
}

if err := <-streamed.Done; err != nil {
	panic(err)
}
```

## 结构化输出

Codex 可生成符合指定 schema 的 JSON 响应。schema 必须是 JSON 对象。

```go
schema := map[string]any{
	"type": "object",
	"properties": map[string]any{
		"summary": map[string]any{"type": "string"},
		"status":  map[string]any{"type": "string", "enum": []any{"ok", "action_required"}},
	},
	"required":             []any{"summary", "status"},
	"additionalProperties": false,
}

turn, err := thread.Run(
	codex.TextInput("汇总仓库状态"),
	codex.TurnOptions{OutputSchema: schema},
)
if err != nil {
	panic(err)
}
fmt.Println(turn.FinalResponse)
```

## 附加图片

需要在文本之外附带图片时，使用结构化输入条目。

```go
turn, err := thread.Run(
	codex.ItemsInput(
		codex.UserInput{Type: codex.UserInputText, Text: "描述这些截图"},
		codex.UserInput{Type: codex.UserInputLocalImage, Path: "./ui.png"},
		codex.UserInput{Type: codex.UserInputLocalImage, Path: "./diagram.jpg"},
	),
	codex.TurnOptions{},
)
```

## 线程管理

线程会持久化到 `~/.codex/sessions`。可使用 `ResumeThread()` 进行恢复。

```go
threadID := os.Getenv("CODEX_THREAD_ID")
thread := codex.ResumeThread(threadID, codex.ThreadOptions{})
_, err := thread.Run(codex.TextInput("实现修复"), codex.TurnOptions{})
```

在首次 turn 启动后可获取当前线程 ID：

```go
id := thread.ID()
```

## 配置

### CodexOptions

```go
client := codex.NewCodex(codex.CodexOptions{
	CodexPathOverride: "/path/to/codex",
	BaseURL:           "https://api.openai.com",
	APIKey:            "your-api-key",
	Config: map[string]any{
		"show_raw_agent_reasoning": true,
		"sandbox_workspace_write": map[string]any{
			"network_access": true,
		},
	},
	Env: map[string]string{
		"PATH": "/usr/local/bin",
	},
})
```

说明：

- `Config` 接受嵌套对象。SDK 会在每次调用 CLI 时将其扁平化为重复的 `--config dotted.path=TOML-value` 参数。
- `ThreadOptions` 中的显式设置会追加在 `Config` 之后，因此命中同一配置键时会覆盖全局默认值。
- `Env` 会完全覆盖传给 CLI 的环境变量（SDK 不会继承父进程环境）。
- `BaseURL` 与 `APIKey` 会映射为 CLI 进程的 `OPENAI_BASE_URL` 与 `CODEX_API_KEY`。

### ThreadOptions

```go
networkAccess := true
thread := client.StartThread(codex.ThreadOptions{
	Model:                 "your-model",
	SandboxMode:           codex.SandboxWorkspaceWrite,
	WorkingDirectory:      "/path/to/project",
	SkipGitRepoCheck:      true,
	ModelReasoningEffort:  codex.ReasoningMedium,
	NetworkAccessEnabled:  &networkAccess,
	WebSearchMode:         codex.WebSearchLive,
	ApprovalPolicy:        codex.ApprovalOnRequest,
	AdditionalDirectories: []string{"/path/to/extra"},
})
```

支持的取值：

- `SandboxMode`: `read-only`, `workspace-write`, `danger-full-access`
- `ModelReasoningEffort`: `minimal`, `low`, `medium`, `high`, `xhigh`
- `WebSearchMode`: `disabled`, `cached`, `live`
- `ApprovalPolicy`: `never`, `on-request`, `on-failure`, `untrusted`

若你更喜欢布尔开关的方式，可设置 `WebSearchEnabled` 而不是 `WebSearchMode`。

## 前向兼容

如果较新的 Codex CLI 发出了本 SDK 尚未建模的 item 类型，stream 不会中断，而是暴露为 `*codex.UnknownItem`。其中 `Type` 保留 item 类型字符串，`Raw` 保留原始 item JSON，便于宿主自行处理。

## 错误处理与用量

- 当 turn 失败或 CLI 退出错误时，`Run()` 会返回错误。
- `Turn.Usage` 在可用时包含 `input_tokens`、`cached_input_tokens` 与 `output_tokens`。

## Codex CLI 解析顺序

SDK 按如下顺序解析 CLI：

1. 若提供则使用 `CodexPathOverride`。
2. 使用 `PATH` 中的 `codex`。
3. 使用模块内 `vendor/<target-triple>/codex/` 下的打包二进制。

## 许可

参见 `LICENSE`。
