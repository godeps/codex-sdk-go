package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godeps/codex-sdk-go/internal/runtimebin"
)

func TestCollectTargetJobsSortsAndSkipsMissingURLs(t *testing.T) {
	t.Parallel()

	jobs := workflowJobsResponse{
		Jobs: []workflowJob{
			{Name: "linux", URL: "https://example.com/linux"},
			{Name: "empty"},
			{Name: "darwin", URL: "https://example.com/darwin"},
		},
	}
	got := collectTargetJobs(jobs)
	if len(got) != 2 {
		t.Fatalf("len(got) = %d, want 2", len(got))
	}
	if got[0].Name != "darwin" || got[1].Name != "linux" {
		t.Fatalf("unexpected job order: %#v", got)
	}
}

func TestFindOneRequiresSingleMatch(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "record.json"), []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := findTargetFile(root, "x86_64-unknown-linux-musl", "record.json"); err == nil {
		t.Fatal("expected missing target match to fail")
	}
}

func TestBuildBundleIncludesManifestSignature(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	for _, path := range []string{
		filepath.Join(repoRoot(), "cmd", "codex-sdk-gen", "main.go"),
		filepath.Join(repoRoot(), "cmd", "codex-sdk-gen", "protocolgen.go"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("missing generator source fixture: %v", err)
		}
	}

	ci := ciEvidence{
		WorkflowName:   "ci",
		WorkflowRunURL: "https://example.com/ci",
		WorkflowRunID:  "1",
		Commit:         "abc123",
		GoVersion:      "go version go1.25.5 linux/amd64",
		Commands:       []string{"go test ./..."},
		StaticAnalyzer: "staticcheck",
	}
	coverage := coverageEvidence{}
	lock := lockFile{RuntimeVersion: runtimebin.DefaultRuntimeVersion, SchemaSHA256: "schema-hash"}
	manifest := runtimebin.Manifest{
		SchemaVersion:  runtimebin.ManifestSchemaVersion,
		SDKVersion:     runtimebin.DefaultSDKVersion,
		RuntimeVersion: runtimebin.DefaultRuntimeVersion,
		UpstreamRepo:   runtimebin.DefaultUpstreamRepo,
		UpstreamTag:    "rust-v0.144.4",
		Targets:        []runtimebin.ManifestTarget{},
	}
	envelope := runtimebin.SignatureEnvelope{KeyID: "runtime-manifest-v1", Algorithm: runtimebin.SignatureAlgorithm}

	cfg := releaseEvidenceConfig{
		Tag:                 "v0.2.0",
		ReleaseWorkflowURL:  "https://example.com/release",
		RuntimeTestsRunID:   "2",
		RuntimeTestsRunURL:  "https://example.com/runtime-tests",
		StressRunID:         "4",
		StressRunURL:        "https://example.com/stress",
		StressEvidenceDir:   filepath.Join(root, "stress"),
		RuntimeNativeRunID:  "3",
		RuntimeNativeRunURL: "https://example.com/runtime-native",
		RuntimeBundleDir:    root,
	}
	if err := os.MkdirAll(cfg.StressEvidenceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeJSONFile(t, filepath.Join(cfg.StressEvidenceDir, "stress-summary.json"), stressSummary{
		DurationSeconds: 1800,
		Iterations:      1,
		Commands:        []string{"go test -race ./..."},
	})
	for _, target := range []string{
		"FuzzClientRequestUnmarshal",
		"FuzzServerRequestUnmarshal",
		"FuzzServerNotificationUnmarshal",
		"FuzzThreadItemUnmarshal",
		"FuzzRequestIDUnmarshal",
		"FuzzThreadListCwdFilterUnmarshal",
		"FuzzAuthModeUnmarshal",
	} {
		writeJSONFile(t, filepath.Join(cfg.StressEvidenceDir, target+".json"), stressSummary{
			FuzzTarget: target,
			FuzzTime:   "10m",
			Package:    "./protocol",
		})
	}

	for _, target := range runtimebin.SupportedTargets() {
		dir := filepath.Join(root, target.Triple)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		archivePath := filepath.Join(dir, runtimebin.ArchiveName(runtimebin.DefaultRuntimeVersion, target))
		recordPath := filepath.Join(dir, "record.json")
		evidencePath := filepath.Join(dir, "evidence.json")
		if err := os.WriteFile(archivePath, []byte(target.Triple), 0o644); err != nil {
			t.Fatal(err)
		}
		sha, size, err := fileSHA256AndSize(archivePath)
		if err != nil {
			t.Fatal(err)
		}
		record := runtimebin.ManifestTarget{
			GOOS:          target.GOOS,
			GOARCH:        target.GOARCH,
			Triple:        target.Triple,
			Executable:    target.Executable,
			UpstreamAsset: target.UpstreamAsset,
			ArchiveName:   archivePath[strings.LastIndex(archivePath, string(os.PathSeparator))+1:],
			ArchiveSHA256: sha,
			ArchiveSize:   size,
			Files:         []runtimebin.ManifestFile{{Path: "provenance.json", SHA256: "abc", Size: 3, Mode: 0o644, Role: "provenance"}},
		}
		recordBytes, _ := json.Marshal(record)
		if err := os.WriteFile(recordPath, recordBytes, 0o644); err != nil {
			t.Fatal(err)
		}
		smokeBytes, _ := json.Marshal(runtimebin.NativeSmokeResult{
			TargetTriple:         target.Triple,
			RuntimeGOOS:          target.GOOS,
			RuntimeGOARCH:        target.GOARCH,
			ArchiveSHA256:        sha,
			ManifestSHA256:       "manifest",
			VersionOutput:        "codex 0.144.4",
			InitializeSucceeded:  true,
			CloseSucceeded:       true,
			ModelListSucceeded:   true,
			ThreadStartSucceeded: true,
			TurnStartSucceeded:   true,
			StreamSucceeded:      true,
		})
		if err := os.WriteFile(evidencePath, smokeBytes, 0o644); err != nil {
			t.Fatal(err)
		}
		manifest.Targets = append(manifest.Targets, record)
	}

	bundle, err := buildBundle(cfg, ci, coverage, workflowJobsResponse{}, lock, manifest, envelope)
	if err != nil {
		t.Fatal(err)
	}
	if bundle.ManifestSignature.KeyID != "runtime-manifest-v1" {
		t.Fatalf("manifest signature key = %q", bundle.ManifestSignature.KeyID)
	}
	if len(bundle.CompletedMatrix) != len(runtimebin.SupportedTargets()) {
		t.Fatalf("completed matrix len = %d", len(bundle.CompletedMatrix))
	}
	if bundle.Stress.Workflow.RunID != "4" || len(bundle.Stress.Summaries) != 8 {
		t.Fatalf("unexpected stress evidence: %#v", bundle.Stress)
	}
}

func writeJSONFile(t *testing.T, path string, value any) {
	t.Helper()
	bytes, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, bytes, 0o644); err != nil {
		t.Fatal(err)
	}
}
