# Runtime Installation

`NewClient` does not download a runtime implicitly. It resolves an executable using the current
options and environment, then starts the app-server immediately.

## Resolution Order

For the current managed client path, the runtime is resolved in this order:

1. `WithCodexPath` / `CodexPathOverride`
2. `CODEX_RUNTIME_PATH`
3. verified managed runtime cache for the requested version
4. `PATH`, only when `WithAllowPATH(true)` is set

`WithEnv` replaces the environment passed to the child process. If you do not set it, the SDK
inherits the current process environment.

## Managing A Runtime Cache

Use `cmd/codex-sdk-runtime` to install and locate pinned runtime archives.

```bash
GOWORK=off go run ./cmd/codex-sdk-runtime version 0.144.4
```

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

The offline archive command works only on the matching platform because the checked-in release
artifacts are target-specific.

## Supported Targets

See [supported-platforms.md](supported-platforms.md) for the current GOOS/GOARCH matrix.

## CLI Environment

When you need to override the child process environment, pass `WithEnv(map[string]string{...})`.
The SDK copies the provided map and also injects its originator marker, `OPENAI_BASE_URL`, and
`CODEX_API_KEY` when those values are set.

## Practical Guidance

- Use `WithCodexPath` when you already know the executable path.
- Use `WithRuntimeCacheRoot` and `WithRuntimeVersion` when you want the SDK-managed cache.
- Use `WithAllowPATH(true)` only when PATH lookup is acceptable for the current deployment.
