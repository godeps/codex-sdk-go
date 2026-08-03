package codex

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/godeps/codex-sdk-go/internal/runtimebin"
)

func TestResolveClientOptionsAcceptsCodexOptionsAndHelpers(t *testing.T) {
	t.Parallel()

	options, err := resolveClientOptions(
		CodexOptions{BaseURL: "https://example.test"},
		WithAPIKey("sk-test"),
		WithRuntimeVersion("0.144.4"),
		WithAllowPATH(true),
	)
	if err != nil {
		t.Fatalf("resolveClientOptions: %v", err)
	}
	if options.BaseURL != "https://example.test" || options.APIKey != "sk-test" || options.RuntimeVersion != "0.144.4" || !options.AllowPATH {
		t.Fatalf("resolved options = %#v", options)
	}
}

func TestResolveManagedExecutablePrecedenceAndBrokenHigherPriority(t *testing.T) {
	target, err := runtimebin.CurrentTarget()
	if err != nil {
		t.Fatalf("CurrentTarget: %v", err)
	}
	version := runtimebin.DefaultRuntimeVersion
	cacheRoot := t.TempDir()
	managed := writeManagedRuntime(t, cacheRoot, version, target)
	envBinary := writeExecutable(t, "env-codex")
	pathBinary := writeExecutable(t, "codex")

	t.Run("explicit overrides env and managed", func(t *testing.T) {
		resolved, err := resolveManagedExecutable(CodexOptions{
			CodexPathOverride: managed,
			Env:               map[string]string{"CODEX_RUNTIME_PATH": envBinary},
			RuntimeCacheRoot:  cacheRoot,
			RuntimeVersion:    version,
			AllowPATH:         true,
		})
		if err != nil {
			t.Fatalf("resolveManagedExecutable: %v", err)
		}
		if resolved.path != managed {
			t.Fatalf("resolved path = %q, want %q", resolved.path, managed)
		}
	})

	t.Run("broken explicit wins and fails", func(t *testing.T) {
		_, err := resolveManagedExecutable(CodexOptions{
			CodexPathOverride: filepath.Join(t.TempDir(), "missing-codex"),
			Env:               map[string]string{"CODEX_RUNTIME_PATH": envBinary},
			RuntimeCacheRoot:  cacheRoot,
			RuntimeVersion:    version,
		})
		if err == nil {
			t.Fatal("expected broken explicit path to fail")
		}
	})

	t.Run("env overrides managed", func(t *testing.T) {
		resolved, err := resolveManagedExecutable(CodexOptions{
			Env:              map[string]string{"CODEX_RUNTIME_PATH": envBinary},
			RuntimeCacheRoot: cacheRoot,
			RuntimeVersion:   version,
		})
		if err != nil {
			t.Fatalf("resolveManagedExecutable: %v", err)
		}
		if resolved.path != envBinary {
			t.Fatalf("resolved path = %q, want %q", resolved.path, envBinary)
		}
	})

	t.Run("broken env wins and fails", func(t *testing.T) {
		_, err := resolveManagedExecutable(CodexOptions{
			Env:              map[string]string{"CODEX_RUNTIME_PATH": filepath.Join(t.TempDir(), "missing-codex")},
			RuntimeCacheRoot: cacheRoot,
			RuntimeVersion:   version,
		})
		if err == nil {
			t.Fatal("expected broken CODEX_RUNTIME_PATH to fail")
		}
	})

	t.Run("managed overrides path", func(t *testing.T) {
		t.Setenv(pathEnvKey(), filepath.Dir(pathBinary))
		resolved, err := resolveManagedExecutable(CodexOptions{
			RuntimeCacheRoot: cacheRoot,
			RuntimeVersion:   version,
			AllowPATH:        true,
		})
		if err != nil {
			t.Fatalf("resolveManagedExecutable: %v", err)
		}
		if resolved.path != managed {
			t.Fatalf("resolved path = %q, want %q", resolved.path, managed)
		}
	})

	t.Run("path only when allowed", func(t *testing.T) {
		emptyCache := t.TempDir()
		t.Setenv(pathEnvKey(), filepath.Dir(pathBinary))
		resolved, err := resolveManagedExecutable(CodexOptions{
			RuntimeCacheRoot: emptyCache,
			RuntimeVersion:   version,
			AllowPATH:        true,
		})
		if err != nil {
			t.Fatalf("resolveManagedExecutable: %v", err)
		}
		if resolved.path != pathBinary {
			t.Fatalf("resolved path = %q, want %q", resolved.path, pathBinary)
		}
		if _, err := resolveManagedExecutable(CodexOptions{
			RuntimeCacheRoot: emptyCache,
			RuntimeVersion:   version,
			AllowPATH:        false,
		}); err == nil {
			t.Fatal("expected PATH disallowed resolution to fail")
		}
	})
}

func writeManagedRuntime(t *testing.T, cacheRoot string, version string, target runtimebin.TargetSpec) string {
	t.Helper()

	normalized, err := runtimebin.NormalizeRuntimeVersion(version)
	if err != nil {
		t.Fatalf("NormalizeRuntimeVersion: %v", err)
	}
	root := filepath.Join(cacheRoot, "targets", target.Triple, normalized, "bin")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	path := filepath.Join(root, target.Executable)
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func writeExecutable(t *testing.T, name string) string {
	t.Helper()

	dir := t.TempDir()
	path := filepath.Join(dir, name)
	if runtime.GOOS == "windows" && filepath.Ext(path) == "" {
		path += ".exe"
	}
	if err := os.WriteFile(path, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}
	return path
}

func pathEnvKey() string {
	if runtime.GOOS == "windows" {
		return "Path"
	}
	return "PATH"
}
