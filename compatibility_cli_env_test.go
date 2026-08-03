package codex

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"
)

func TestCompatibilityCLIArgvTranscriptMatchesV01Golden(t *testing.T) {
	t.Parallel()

	networkAccess := true
	args, err := buildCommandArgs(CodexExecArgs{
		ThreadID: "thread-v01",
		Images:   []string{"one.png", "two.jpg"},
		Config: map[string]any{
			"approval_policy":        "never",
			"feature_enabled":        true,
			"model_reasoning_effort": "low",
			"nested": map[string]any{
				"answer": 42,
				"array":  []any{"x", true, 3.5},
			},
			"sandbox_workspace_write": map[string]any{
				"network_access": false,
			},
			"web_search": "cached",
		},
		Model:                 "gpt-5",
		SandboxMode:           SandboxDangerFullAccess,
		WorkingDirectory:      "/tmp/project",
		AdditionalDirectories: []string{"/extra/one", "/extra/two"},
		SkipGitRepoCheck:      true,
		OutputSchemaFile:      "/tmp/schema.json",
		ModelReasoningEffort:  ReasoningHigh,
		NetworkAccessEnabled:  &networkAccess,
		WebSearchMode:         WebSearchLive,
		ApprovalPolicy:        ApprovalOnRequest,
	})
	if err != nil {
		t.Fatalf("buildCommandArgs: %v", err)
	}

	got, err := json.MarshalIndent(args, "", "  ")
	if err != nil {
		t.Fatalf("marshal args: %v", err)
	}
	want := readCompatFile(t, filepath.Join("transcripts", "argv-thread-options.json"))
	if string(got)+"\n" != want {
		t.Fatalf("argv transcript mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestCompatibilityNilEnvInheritsParentAndSetsOriginator(t *testing.T) {
	t.Setenv("COMPAT_INHERITED", "present")
	env := envMap(buildEnv(nil, "https://example.test", "sk-test"))

	if env["COMPAT_INHERITED"] != "present" {
		t.Fatalf("expected inherited env, got %q", env["COMPAT_INHERITED"])
	}
	if env[internalOriginatorEnv] != goSDKOriginator {
		t.Fatalf("expected originator override %q, got %q", goSDKOriginator, env[internalOriginatorEnv])
	}
	if env["OPENAI_BASE_URL"] != "https://example.test" {
		t.Fatalf("expected OPENAI_BASE_URL override, got %q", env["OPENAI_BASE_URL"])
	}
	if env["CODEX_API_KEY"] != "sk-test" {
		t.Fatalf("expected CODEX_API_KEY override, got %q", env["CODEX_API_KEY"])
	}
}

func TestCompatibilityOverrideEnvReplacesParentEnvironment(t *testing.T) {
	t.Setenv("COMPAT_SHOULD_NOT_LEAK", "1")
	got := envMap(buildEnv(map[string]string{
		"PATH": "/custom/bin",
		"Path": `C:\custom\bin`,
	}, "", ""))
	want := map[string]string{
		"PATH":                "/custom/bin",
		"Path":                `C:\custom\bin`,
		internalOriginatorEnv: goSDKOriginator,
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("unexpected override env:\n got: %#v\nwant: %#v", got, want)
	}
}

func TestCompatibilityFindCodexPathPrefersPATHBinary(t *testing.T) {
	temp := t.TempDir()
	path := filepath.Join(temp, "codex")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("write fake codex: %v", err)
	}
	t.Setenv("PATH", temp)

	got := findCodexPath()
	if got != path {
		t.Fatalf("expected PATH binary %q, got %q", path, got)
	}
}

func TestCompatibilityFindCodexPathFallsBackToBundledVendorPath(t *testing.T) {
	t.Setenv("PATH", "")
	got := findCodexPath()
	if !strings.Contains(got, string(filepath.Separator)+"vendor"+string(filepath.Separator)) {
		t.Fatalf("expected vendor fallback path, got %q", got)
	}
	if filepath.Base(got) != codexBinaryNameForRuntime() {
		t.Fatalf("unexpected binary name in %q", got)
	}
}

func envMap(entries []string) map[string]string {
	out := make(map[string]string, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			out[parts[0]] = parts[1]
		}
	}
	return out
}

func codexBinaryNameForRuntime() string {
	if runtime.GOOS == "windows" {
		return "codex.exe"
	}
	return "codex"
}
