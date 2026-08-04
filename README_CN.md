# Codex SDK（Go）

Go 版 Codex SDK 通过一套基于 `context.Context` 的 `Client` API 提供长期连接和并发调用。
v0.1 兼容外观层在 v0.2.x 迁移窗口结束后退役，不再属于 v0.3 公共 API。

## 快速开始

```go
package main

import (
	"context"
	"fmt"
	"time"

	codex "github.com/godeps/codex-sdk-go"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
	defer cancel()

	client, err := codex.NewClient(ctx)
	if err != nil {
		panic(err)
	}
	defer client.Close()

	thread, err := client.StartThread(ctx, codex.ThreadOptions{
		WorkingDirectory: ".",
		SandboxMode:      codex.SandboxDangerFullAccess,
		ApprovalPolicy:   codex.ApprovalNever,
	})
	if err != nil {
		panic(err)
	}

	turn, err := thread.RunContext(ctx, codex.TextInput("概述这个仓库"), codex.TurnOptions{})
	if err != nil {
		panic(err)
	}

	fmt.Println(turn.FinalResponse)
}
```

`NewClient` 会立即启动 app-server。需要固定 runtime、API key 或配置时，可再传入
`codex.WithCodexPath`、`codex.WithRuntimeCacheRoot`、`codex.WithRuntimeVersion`、
`codex.WithAllowPATH`、`codex.WithBaseURL` 或 `codex.WithAPIKey`。

## API 一览

### 根客户端

- `NewClient(ctx, opts...)` 会立即启动并初始化 app-server。
- `Client.Metadata`、`Close`、`CloseContext`、`Wait` 和 `WaitContext` 显式暴露生命周期。
- `Client.ListModels`、`StartThread`、`ResumeThread`、`ReadThread`、`ListThreads`、
  `ForkThread`、`ArchiveThread`、`UnarchiveThread`、`SetThreadName`、`CompactThread`
  都使用 `context.Context`。
- `Client.LoginAPIKey`、`LoginChatGPT`、`LoginDeviceCode`、`Account`、`Logout`、
  `GetGoal`、`SetGoal`、`ClearGoal`、`PauseGoal`、`StartGoal` 同样是 context-first。

### 线程、turn 与 goal

- `Thread.RunContext` 直接启动并收集一次 turn。
- `Thread.StartTurnContext` 返回 `TurnHandle`。
- `TurnHandle.Stream`、`RunContext`、`Steer`、`Interrupt` 用来控制一个进行中的 turn。
- `Thread.StartGoalContext` 返回 `GoalHandle`。
- `GoalHandle.StreamContext`、`RunContext`、`CancelContext`、`Close` 用来控制一个逻辑 goal。

`TurnResult` 包含 turn ID、状态、错误、时间戳、耗时、最终回答、items 和 usage。

### 输入

使用 `TextInput` 传递普通提示词，使用 `ItemsInput` 传递结构化输入。当前提供的辅助
构造器：

- `DataURLImageInput(url string)`
- `LocalImageInput(path string)`
- `SkillInput(name, path string)`
- `MentionInput(name, path string)`

`ItemsInput` 会保持顺序；空值、非法值或不支持的条目会直接返回错误。

### 选项

`NewClient` 既可以直接接收 `CodexOptions`，也可以接收辅助 `Option`。当前可用选项：

- `WithCodexPath`
- `WithBaseURL`
- `WithAPIKey`
- `WithConfig`
- `WithEnv`
- `WithRuntimeCacheRoot`
- `WithRuntimeVersion`
- `WithAllowPATH`

`ThreadOptions` 和 `TurnOptions` 暴露了当前 app-server 的主要字段。

常见枚举值：

- `ApprovalMode`：`never`、`on-request`、`on-failure`、`untrusted`
- `ApprovalPreset`：`deny_all`、`auto_review`
- `SandboxMode`：`read-only`、`workspace-write`、`danger-full-access`
- `ModelReasoningEffort`：`minimal`、`low`、`medium`、`high`、`xhigh`
- `WebSearchMode`：`disabled`、`cached`、`live`
- `ReasoningSummary`：`none`、`auto`、`brief`、`detailed`

`ApprovalPresetDenyAll` 映射到 `ApprovalNever`。`ApprovalPresetAutoReview` 映射到
`ApprovalOnRequest` 并使用 `auto_review` reviewer。

## 流式、Steer 与 Interrupt

`TurnHandle.Stream()` 返回 pull-style 的 `TurnStream`。每次 `Next(ctx)` 都会返回一个
`ThreadEvent`。`Close()` 是幂等的，并会注销 route。`Close` 之后继续 `Next` 会返回
`ErrStreamClosed`。

`TurnHandle.Steer` 用来向正在运行的 turn 注入额外输入，`TurnHandle.Interrupt` 用来
停止它。

## 登录、账号与 Goal

上下文优先的登录与账号 API：

- `Client.LoginAPIKey(ctx, apiKey)`
- `Client.LoginChatGPT(ctx)`
- `Client.LoginDeviceCode(ctx)`
- `Client.Account(ctx, refreshToken)`
- `Client.Logout(ctx)`

交互式登录句柄提供 `WaitContext` 和 `CancelContext`。

goal API 作用于已持久化的线程：

- `Thread.GetGoalContext(ctx)`
- `Thread.SetGoalContext(ctx, update)`
- `Thread.ClearGoalContext(ctx)`
- `Thread.PauseGoalContext(ctx)`
- `Thread.StartGoalContext(ctx, objective)`

`GoalHandle` 是逻辑 goal 流；用 `RunContext` 收集最终结果，用 `CancelContext`
执行尽力而为的暂停与中断清理。

## 运行时安装与解析

SDK 客户端和 runtime 安装器解决的是不同问题：

- `NewClient` 解析 Codex 可执行文件的顺序是 `WithCodexPath`、`CODEX_RUNTIME_PATH`、
  已验证的 managed cache，再到仅在 `WithAllowPATH(true)` 时允许的 `PATH`。
- `WithEnv` 会替换传给 Codex CLI 进程的环境；如果不设置，它会继承当前进程环境。
- `codex-sdk-runtime` 管理固定版本的 runtime 包和缓存。其解析顺序是显式路径、
  `CODEX_RUNTIME_PATH`、已验证的 managed cache，最后才是在策略允许时使用 `PATH`。

支持的平台：

| GOOS | GOARCH | target triple | executable |
|---|---|---|---|
| darwin | amd64 | x86_64-apple-darwin | codex |
| darwin | arm64 | aarch64-apple-darwin | codex |
| linux | amd64 | x86_64-unknown-linux-musl | codex |
| linux | arm64 | aarch64-unknown-linux-musl | codex |
| windows | amd64 | x86_64-pc-windows-msvc | codex.exe |
| windows | arm64 | aarch64-pc-windows-msvc | codex.exe |

当前平台可以使用仓库内的离线测试包执行安装：

```bash
archive="runtime/testdata/release/codex-sdk-go-runtime_0.144.4_$(go env GOOS)_$(go env GOARCH).tar.gz"
cache="$(mktemp -d)"
GOWORK=off go run ./cmd/codex-sdk-runtime install \
  --cache-root "$cache" \
  --manifest runtime/manifest.json \
  --manifest-signature runtime/manifest.json.sig \
  --archive "$archive"
GOWORK=off go run ./cmd/codex-sdk-runtime path \
  --cache-root "$cache" \
  --runtime-version 0.144.4
```

版本映射命令：

```bash
GOWORK=off go run ./cmd/codex-sdk-runtime version 0.144.4
```

## 协议生成与锁定

`reference.lock.json` 锁定了上游 Codex checkout、Python 根目录、runtime 包与版本、
schema 路径、schema SHA-256，以及六个平台目标。生成器不会自动切换到别的 checkout。

- `GOWORK=off go run ./cmd/codex-sdk-gen verify` 用来检查 `schema/` 和
  `protocol/manifest.json` 是否与锁一致。
- `GOWORK=off go run ./cmd/codex-sdk-gen refresh` 会从锁定的 reference checkout 重新
  刷新快照。

如果锁定的 reference 仓库不可用，生成器会直接失败，而不是自动切换到别的版本。

## 迁移与示例

从 v0.1 或 v0.2 兼容 API 升级的调用方，需要在采用 v0.3 前完成迁移。参见
[docs/migration-v0.1-to-v0.2.md](docs/migration-v0.1-to-v0.2.md) 和
[docs/api-reference.md](docs/api-reference.md)。

可执行示例位于：

- `example/common`
- `example/stream`
- `example/steer`
- `example/image`
- `example/structured-output`

## 贡献

见 [CONTRIBUTING.md](CONTRIBUTING.md)。

## 许可证

MIT。见 [LICENSE](LICENSE)。
