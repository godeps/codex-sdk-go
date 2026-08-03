package main

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/godeps/codex-sdk-go/internal/runtimebin"
)

func TestRunVersion(t *testing.T) {
	if err := run(t.Context(), []string{"version", "0.144.4a1.post2"}); err != nil {
		t.Fatalf("run version: %v", err)
	}
}

func TestRunManifestAndSign(t *testing.T) {
	root := t.TempDir()
	recordPaths := make([]string, 0, len(runtimebin.SupportedTargets()))
	for _, target := range runtimebin.SupportedTargets() {
		packageDir := filepath.Join(root, target.Triple)
		mustMkdirAll(t, filepath.Join(packageDir, "bin"))
		mustMkdirAll(t, filepath.Join(packageDir, "codex-path"))
		mustMkdirAll(t, filepath.Join(packageDir, "codex-resources"))
		mustWriteFile(t, filepath.Join(packageDir, "codex-package.json"), []byte("{}\n"), 0o644)
		mustWriteFile(t, filepath.Join(packageDir, "bin", target.Executable), []byte("codex\n"), 0o755)
		hostName := "codex-code-mode-host"
		rgName := "rg"
		if target.GOOS == "windows" {
			hostName += ".exe"
			rgName += ".exe"
		}
		mustWriteFile(t, filepath.Join(packageDir, "bin", hostName), []byte("host\n"), 0o755)
		mustWriteFile(t, filepath.Join(packageDir, "codex-path", rgName), []byte("rg\n"), 0o755)
		if target.GOOS == "linux" {
			mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "bwrap"), []byte("bwrap\n"), 0o755)
		}
		if target.GOOS == "windows" {
			mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "codex-command-runner.exe"), []byte("runner\n"), 0o755)
			mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "codex-windows-sandbox-setup.exe"), []byte("sandbox\n"), 0o755)
		}

		recordPath := filepath.Join(root, target.Triple+".json")
		archivePath := filepath.Join(root, target.Triple+".tar.gz")
		record, err := runtimebin.PackageArchive(runtimebin.PackageOptions{PackageDir: packageDir, OutputArchive: archivePath, RuntimeVersion: runtimebin.DefaultRuntimeVersion, Target: target})
		if err != nil {
			t.Fatal(err)
		}
		recordBytes, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(recordPath, recordBytes, 0o644); err != nil {
			t.Fatal(err)
		}
		recordPaths = append(recordPaths, recordPath)
	}

	manifestPath := filepath.Join(root, "manifest.json")
	args := []string{"manifest", "--output", manifestPath}
	for _, path := range recordPaths {
		args = append(args, "--record", path)
	}
	if err := run(t.Context(), args); err != nil {
		t.Fatalf("run manifest: %v", err)
	}

	seed := make([]byte, ed25519.SeedSize)
	for i := range seed {
		seed[i] = 9
	}
	privateKey := ed25519.NewKeyFromSeed(seed)
	signaturePath := filepath.Join(root, "manifest.json.sig")
	t.Setenv("CODEX_RUNTIME_MANIFEST_PRIVATE_KEY_BASE64", base64.StdEncoding.EncodeToString(privateKey))
	if err := run(t.Context(), []string{
		"sign",
		"--manifest", manifestPath,
		"--output", signaturePath,
		"--key-id", "test",
	}); err != nil {
		t.Fatalf("run sign: %v", err)
	}

	manifest, manifestBytes, err := runtimebin.LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.Targets) != len(runtimebin.SupportedTargets()) {
		t.Fatalf("manifest target count = %d", len(manifest.Targets))
	}
	envelope, err := runtimebin.LoadSignatureEnvelope(signaturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := runtimebin.VerifyManifestSignature(manifestBytes, envelope, []runtimebin.TrustRoot{{
		KeyID:     "test",
		PublicKey: privateKey.Public().(ed25519.PublicKey),
	}}); err != nil {
		t.Fatalf("verify manifest signature: %v", err)
	}
}

func TestRunRepackAndNativeSmoke(t *testing.T) {
	root := t.TempDir()
	target := currentTestTarget(t)
	upstreamDir := filepath.Join(root, "upstream")
	buildTestPackage(t, upstreamDir, target)
	upstreamArchive := filepath.Join(root, "upstream.tar.gz")
	if _, err := runtimebin.PackageArchive(runtimebin.PackageOptions{
		PackageDir:     upstreamDir,
		OutputArchive:  upstreamArchive,
		RuntimeVersion: runtimebin.DefaultRuntimeVersion,
		Target:         target,
	}); err != nil {
		t.Fatal(err)
	}

	stageDir := filepath.Join(root, "stage")
	recordPath := filepath.Join(root, "record.json")
	archivePath := filepath.Join(root, "repacked.tar.gz")
	if err := run(t.Context(), []string{
		"repack",
		"--target", target.Triple,
		"--upstream-archive", upstreamArchive,
		"--stage-dir", stageDir,
		"--archive", archivePath,
		"--record", recordPath,
	}); err != nil {
		t.Fatalf("run repack: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stageDir, "NOTICE")); err != nil {
		t.Fatalf("missing staged NOTICE: %v", err)
	}
	evidencePath := filepath.Join(root, "evidence.json")
	if err := run(t.Context(), []string{
		"native-smoke",
		"--runtime-path", filepath.Join(stageDir, "bin", target.Executable),
		"--root-dir", stageDir,
		"--target", target.Triple,
		"--record", recordPath,
		"--evidence", evidencePath,
	}); err != nil {
		t.Fatalf("run native-smoke: %v", err)
	}
	bytes, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(bytes), `"versionOutput"`) {
		t.Fatalf("unexpected evidence payload: %s", bytes)
	}
}

func TestRealLinuxArchiveIntegrationFromEnv(t *testing.T) {
	archivePath := os.Getenv("CODEX_RUNTIME_REAL_LINUX_ARCHIVE")
	if archivePath == "" {
		t.Skip("CODEX_RUNTIME_REAL_LINUX_ARCHIVE not set")
	}
	if runtime.GOOS != "linux" || runtime.GOARCH != "amd64" {
		t.Skip("real runtime integration requires linux/amd64 host")
	}

	root := t.TempDir()
	target, err := runtimebin.LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	recordPath := filepath.Join(root, "record.json")
	archiveOut := filepath.Join(root, "repacked.tar.gz")
	stageDir := filepath.Join(root, "stage")
	if err := run(t.Context(), []string{
		"repack",
		"--target", target.Triple,
		"--runtime-version", runtimebin.DefaultRuntimeVersion,
		"--sdk-version", runtimebin.DefaultSDKVersion,
		"--upstream-archive", archivePath,
		"--stage-dir", stageDir,
		"--archive", archiveOut,
		"--record", recordPath,
	}); err != nil {
		t.Fatalf("real repack: %v", err)
	}
	evidencePath := filepath.Join(root, "evidence.json")
	if err := run(t.Context(), []string{
		"native-smoke",
		"--runtime-path", filepath.Join(stageDir, "bin", target.Executable),
		"--root-dir", stageDir,
		"--target", target.Triple,
		"--record", recordPath,
		"--evidence", evidencePath,
	}); err != nil {
		t.Fatalf("real native-smoke: %v", err)
	}
	evidence, err := os.ReadFile(evidencePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(evidence), `"initializeSucceeded": true`) || !strings.Contains(string(evidence), `"streamSucceeded": true`) {
		t.Fatalf("unexpected real evidence payload: %s", evidence)
	}
}

func currentTestTarget(t *testing.T) runtimebin.TargetSpec {
	t.Helper()
	target, err := runtimebin.CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	return target
}

func buildTestPackage(t *testing.T, packageDir string, target runtimebin.TargetSpec) {
	t.Helper()
	mustMkdirAll(t, filepath.Join(packageDir, "bin"))
	mustMkdirAll(t, filepath.Join(packageDir, "codex-path"))
	mustMkdirAll(t, filepath.Join(packageDir, "codex-resources"))
	mustWriteFile(t, filepath.Join(packageDir, "codex-package.json"), []byte("{}\n"), 0o644)
	script := `#!/usr/bin/env python3
import json
import sys
import time

if len(sys.argv) > 1 and sys.argv[1] == "--version":
    print("codex-cli 0.144.4")
    raise SystemExit(0)

if len(sys.argv) > 2 and sys.argv[1] == "app-server" and sys.argv[2] == "--listen":
    for line in sys.stdin:
        line = line.strip()
        if not line:
            continue
        message = json.loads(line)
        req_id = message.get("id")
        method = message.get("method")
        if method == "initialize":
            print(json.dumps({"id": req_id, "result": {"protocolVersion": "2026-08-03", "userAgent": "fake", "serverInfo": {"name": "fake-app-server", "version": "1.0.0"}}}))
            sys.stdout.flush()
        elif method == "initialized":
            continue
        elif method == "model/list":
            print(json.dumps({"id": req_id, "result": {"data": [{"id": "mock-model", "model": "mock-model"}]}}))
            sys.stdout.flush()
        elif method == "thread/start":
            print(json.dumps({"id": req_id, "result": {"thread": {"id": "thread-1", "status": "idle"}}}))
            sys.stdout.flush()
        elif method == "turn/start":
            print(json.dumps({"id": req_id, "result": {"turn": {"id": "turn-1", "status": "in_progress"}}}))
            sys.stdout.flush()
            time.sleep(0.1)
            print(json.dumps({"method": "turn/started", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "in_progress"}}}))
            print(json.dumps({"method": "item/completed", "params": {"threadId": "thread-1", "turnId": "turn-1", "item": {"id": "msg-1", "type": "agent_message", "text": "runtime smoke ok", "phase": "final_answer"}}}))
            print(json.dumps({"method": "thread/tokenUsage/updated", "params": {"threadId": "thread-1", "turnId": "turn-1", "tokenUsage": {"last": {"input_tokens": 1, "cached_input_tokens": 0, "output_tokens": 1}}}}))
            print(json.dumps({"method": "turn/completed", "params": {"threadId": "thread-1", "turn": {"id": "turn-1", "status": "completed"}}}))
            sys.stdout.flush()
        else:
            print(json.dumps({"id": req_id, "result": {}}))
            sys.stdout.flush()
    raise SystemExit(0)

raise SystemExit(2)
`
	mustWriteFile(t, filepath.Join(packageDir, "bin", target.Executable), []byte(script), 0o755)
	hostName := "codex-code-mode-host"
	rgName := "rg"
	if target.GOOS == "windows" {
		hostName += ".exe"
		rgName += ".exe"
	}
	mustWriteFile(t, filepath.Join(packageDir, "bin", hostName), []byte("host\n"), 0o755)
	mustWriteFile(t, filepath.Join(packageDir, "codex-path", rgName), []byte("rg\n"), 0o755)
	switch target.GOOS {
	case "linux":
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "bwrap"), []byte("bwrap\n"), 0o755)
		mustMkdirAll(t, filepath.Join(packageDir, "codex-resources", "zsh", "bin"))
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "zsh", "bin", "zsh"), []byte("zsh\n"), 0o755)
	case "darwin":
		mustMkdirAll(t, filepath.Join(packageDir, "codex-resources", "zsh", "bin"))
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "zsh", "bin", "zsh"), []byte("zsh\n"), 0o755)
	case "windows":
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "codex-command-runner.exe"), []byte("runner\n"), 0o755)
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "codex-windows-sandbox-setup.exe"), []byte("sandbox\n"), 0o755)
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteFile(t *testing.T, path string, contents []byte, mode os.FileMode) {
	t.Helper()
	if err := os.WriteFile(path, contents, mode); err != nil {
		t.Fatal(err)
	}
}
