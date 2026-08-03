package main

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io"
	"math"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/godeps/codex-sdk-go/internal/runtimebin"
)

func TestMockResponsesServerRecordsRequests(t *testing.T) {
	server, err := newMockResponsesServer()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = server.close() }()

	modelsResp, err := http.Get(server.url + "/v1/models") //nolint:noctx
	if err != nil {
		t.Fatal(err)
	}
	defer modelsResp.Body.Close()
	modelsBody, err := io.ReadAll(modelsResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(modelsBody), `"mock-model"`) {
		t.Fatalf("unexpected models payload: %s", modelsBody)
	}

	responsesResp, err := http.Post(server.url+"/v1/responses", "application/json", strings.NewReader(`{"input":"hi"}`)) //nolint:noctx
	if err != nil {
		t.Fatal(err)
	}
	defer responsesResp.Body.Close()
	streamBody, err := io.ReadAll(responsesResp.Body)
	if err != nil {
		t.Fatal(err)
	}
	bodyText := string(streamBody)
	for _, needle := range []string{"response.created", "response.completed", `"total_tokens":5`} {
		if !strings.Contains(bodyText, needle) {
			t.Fatalf("stream payload missing %q: %s", needle, bodyText)
		}
	}
	if got := server.requestCount(); got != 2 {
		t.Fatalf("requestCount() = %d, want 2", got)
	}
}

func TestAccumulateTurnEventCapturesUsageAndFinalMessage(t *testing.T) {
	var result runtimebin.NativeSmokeResult
	done, err := accumulateTurnEvent(json.RawMessage(`{
	  "method":"thread/tokenUsage/updated",
	  "params":{
	    "threadId":"thread-1",
	    "turnId":"turn-1",
	    "tokenUsage":{
	      "last":{"input_tokens":2,"cached_input_tokens":1,"output_tokens":3},
	      "total":{"totalTokens":5,"reasoningOutputTokens":0}
	    }
	  }
	}`), &result)
	if err != nil {
		t.Fatal(err)
	}
	if done {
		t.Fatal("usage update should not terminate the stream")
	}

	done, err = accumulateTurnEvent(json.RawMessage(`{
	  "method":"item/completed",
	  "params":{
	    "threadId":"thread-1",
	    "turnId":"turn-1",
	    "item":{"id":"msg-1","type":"agentMessage","text":"runtime smoke ok","phase":{"value":"final_answer"}}
	  }
	}`), &result)
	if err != nil {
		t.Fatal(err)
	}
	if done {
		t.Fatal("item completion should not terminate the stream")
	}

	done, err = accumulateTurnEvent(json.RawMessage(`{
	  "method":"turn/completed",
	  "params":{"threadId":"thread-1","turn":{"id":"turn-1","status":"completed"}}
	}`), &result)
	if err != nil {
		t.Fatal(err)
	}
	if !done {
		t.Fatal("turn/completed should terminate the stream")
	}
	if result.FinalResponse != "runtime smoke ok" {
		t.Fatalf("FinalResponse = %q", result.FinalResponse)
	}
	if result.TurnStatus != "completed" {
		t.Fatalf("TurnStatus = %q", result.TurnStatus)
	}
	if result.Usage == nil || result.Usage.InputTokens != 2 || result.Usage.CachedInputTokens != 1 || result.Usage.OutputTokens != 3 || result.Usage.TotalTokens != 5 {
		t.Fatalf("Usage = %#v", result.Usage)
	}
}

func TestCLIInstallVerifyPathAndRemove(t *testing.T) {
	target := currentTestTarget(t)
	cacheRoot := t.TempDir()
	release := buildSignedRelease(t)
	recordPath := release.recordPaths[target.Triple]

	if _, err := runtimebin.Install(context.Background(), runtimebin.InstallOptions{
		CacheRoot:      cacheRoot,
		RuntimeVersion: runtimebin.DefaultRuntimeVersion,
		Target:         &target,
		ManifestPath:   release.manifestPath,
		SignaturePath:  release.signaturePath,
		ArchivePath:    release.archivePaths[target.Triple],
		TrustRoots:     release.trustRoots,
	}); err != nil {
		t.Fatalf("runtimebin.Install: %v", err)
	}

	if _, err := captureStdout(t, func() error {
		return run(context.Background(), []string{
			"verify",
			"--cache-root", cacheRoot,
			"--target", target.Triple,
			"--runtime-version", runtimebin.DefaultRuntimeVersion,
			"--manifest", release.manifestPath,
		})
	}); err != nil {
		t.Fatalf("verify: %v", err)
	}

	pathOutput, err := captureStdout(t, func() error {
		return run(context.Background(), []string{
			"path",
			"--cache-root", cacheRoot,
			"--target", target.Triple,
			"--runtime-version", runtimebin.DefaultRuntimeVersion,
		})
	})
	if err != nil {
		t.Fatalf("path: %v", err)
	}
	if !strings.Contains(pathOutput, filepath.Join("bin", target.Executable)) {
		t.Fatalf("path output = %q", pathOutput)
	}

	if _, err := captureStdout(t, func() error {
		return run(context.Background(), []string{
			"remove",
			"--cache-root", cacheRoot,
			"--target", target.Triple,
			"--runtime-version", runtimebin.DefaultRuntimeVersion,
		})
	}); err != nil {
		t.Fatalf("remove: %v", err)
	}

	recordBytes, err := os.ReadFile(recordPath)
	if err != nil {
		t.Fatal(err)
	}
	var record runtimebin.ManifestTarget
	if err := json.Unmarshal(recordBytes, &record); err != nil {
		t.Fatal(err)
	}
	if _, err := runtimebin.VerifyInstalled(cacheRoot, runtimebin.DefaultRuntimeVersion, target, record); err == nil {
		t.Fatal("VerifyInstalled unexpectedly succeeded after remove")
	}
}

func TestCLICommandValidation(t *testing.T) {
	if err := run(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "usage: codex-sdk-runtime") {
		t.Fatalf("run(nil) = %v", err)
	}
	if err := run(context.Background(), []string{"unknown"}); err == nil || !strings.Contains(err.Error(), "usage: codex-sdk-runtime") {
		t.Fatalf("run(unknown) = %v", err)
	}
	if err := runPackage(nil); err == nil || !strings.Contains(err.Error(), "package requires") {
		t.Fatalf("runPackage(nil) = %v", err)
	}
	if err := runVersion(nil); err == nil || !strings.Contains(err.Error(), "one argument") {
		t.Fatalf("runVersion(nil) = %v", err)
	}
	if _, err := lookupTargetFlag("bad-triple"); err == nil {
		t.Fatal("lookupTargetFlag(bad-triple) unexpectedly succeeded")
	}
	if _, err := decodeBase64("%%%"); err == nil {
		t.Fatal("decodeBase64(%%%) unexpectedly succeeded")
	}
	if _, err := captureStdout(t, func() error {
		return run(context.Background(), []string{
			"install",
			"--cache-root", t.TempDir(),
			"--manifest", filepath.Join("does", "not", "exist.json"),
			"--manifest-signature", filepath.Join("does", "not", "exist.sig"),
			"--archive", filepath.Join("does", "not", "exist.tar.gz"),
		})
	}); err == nil {
		t.Fatal("install with missing files unexpectedly succeeded")
	}

	root := t.TempDir()
	target := currentTestTarget(t)
	packageDir := filepath.Join(root, "package")
	buildTestPackage(t, packageDir, target)
	archivePath := filepath.Join(root, "package.tar.gz")
	if _, err := captureStdout(t, func() error {
		return run(context.Background(), []string{
			"package",
			"--package-dir", packageDir,
			"--archive", archivePath,
			"--target", target.Triple,
		})
	}); err != nil {
		t.Fatalf("package: %v", err)
	}
	if _, err := os.Stat(archivePath); err != nil {
		t.Fatalf("package archive missing: %v", err)
	}
	if err := runNativeSmoke(context.Background(), nil); err == nil || !strings.Contains(err.Error(), "native-smoke requires") {
		t.Fatalf("runNativeSmoke(nil) = %v", err)
	}
	if err := runVerify([]string{"--manifest", filepath.Join("does", "not", "exist.json")}); err == nil {
		t.Fatal("runVerify() unexpectedly succeeded with a missing manifest")
	}
}

func TestNumericValueAndMessagePhaseHelpers(t *testing.T) {
	cases := []struct {
		value any
		want  int64
		ok    bool
	}{
		{value: int(1), want: 1, ok: true},
		{value: int32(2), want: 2, ok: true},
		{value: int64(3), want: 3, ok: true},
		{value: float64(4), want: 4, ok: true},
		{value: json.Number("5"), want: 5, ok: true},
		{value: math.Pi, want: 3, ok: true},
		{value: "bad", want: 0, ok: false},
	}
	for _, tc := range cases {
		got, ok := numericValue(tc.value)
		if got != tc.want || ok != tc.ok {
			t.Fatalf("numericValue(%#v) = (%d, %v), want (%d, %v)", tc.value, got, ok, tc.want, tc.ok)
		}
	}
	if got := messagePhase(json.RawMessage(`"final_answer"`)); got != "final_answer" {
		t.Fatalf("messagePhase(string) = %q", got)
	}
	if got := messagePhase(json.RawMessage(`{"type":"final_answer"}`)); got != "final_answer" {
		t.Fatalf("messagePhase(object) = %q", got)
	}
	if got := messagePhase(json.RawMessage(`{"unexpected":true}`)); got != "" {
		t.Fatalf("messagePhase(unexpected) = %q", got)
	}
}

func TestMainHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		os.Args = []string{"codex-sdk-runtime", "version", "0.144.4"}
		main()
		return
	}

	cmd := exec.Command(os.Args[0], "-test.run", "^TestMainHelperProcess$")
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("helper process failed: %v\n%s", err, output)
	}
	if !strings.Contains(string(output), "rust-v0.144.4") {
		t.Fatalf("helper process output = %q", output)
	}
}

func captureStdout(t *testing.T, fn func() error) (string, error) {
	t.Helper()

	original := os.Stdout
	reader, writer, err := os.Pipe()
	if err != nil {
		t.Fatal(err)
	}
	os.Stdout = writer
	runErr := fn()
	_ = writer.Close()
	os.Stdout = original
	bytes, readErr := io.ReadAll(reader)
	_ = reader.Close()
	if readErr != nil {
		t.Fatal(readErr)
	}
	return string(bytes), runErr
}

type signedRelease struct {
	manifestPath  string
	signaturePath string
	archivePaths  map[string]string
	recordPaths   map[string]string
	trustRoots    []runtimebin.TrustRoot
}

func buildSignedRelease(t *testing.T) signedRelease {
	t.Helper()

	root := t.TempDir()
	recordPaths := make([]string, 0, len(runtimebin.SupportedTargets()))
	archives := make(map[string]string, len(runtimebin.SupportedTargets()))
	recordIndex := make(map[string]string, len(runtimebin.SupportedTargets()))
	for _, target := range runtimebin.SupportedTargets() {
		packageDir := filepath.Join(root, target.Triple)
		buildTestPackage(t, packageDir, target)
		archivePath := filepath.Join(root, runtimebin.ArchiveName(runtimebin.DefaultRuntimeVersion, target))
		recordPath := filepath.Join(root, target.Triple+".json")
		record, err := runtimebin.PackageArchive(runtimebin.PackageOptions{
			PackageDir:     packageDir,
			OutputArchive:  archivePath,
			RuntimeVersion: runtimebin.DefaultRuntimeVersion,
			Target:         target,
		})
		if err != nil {
			t.Fatal(err)
		}
		bytes, err := json.Marshal(record)
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(recordPath, bytes, 0o644); err != nil {
			t.Fatal(err)
		}
		recordPaths = append(recordPaths, recordPath)
		archives[target.Triple] = archivePath
		recordIndex[target.Triple] = recordPath
	}

	manifestPath := filepath.Join(root, "manifest.json")
	args := []string{"manifest", "--output", manifestPath}
	args = append(args, "--sdk-version", runtimebin.DefaultSDKVersion, "--runtime-version", runtimebin.DefaultRuntimeVersion)
	for _, recordPath := range recordPaths {
		args = append(args, "--record", recordPath)
	}
	if _, err := captureStdout(t, func() error { return run(context.Background(), args) }); err != nil {
		t.Fatalf("manifest: %v", err)
	}

	seed := bytes.Repeat([]byte{11}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	signaturePath := filepath.Join(root, "manifest.json.sig")
	if _, err := captureStdout(t, func() error {
		return run(context.Background(), []string{
			"sign",
			"--manifest", manifestPath,
			"--output", signaturePath,
			"--key-id", "runtime-manifest-v3",
			"--private-key-base64", base64.StdEncoding.EncodeToString(privateKey),
		})
	}); err != nil {
		t.Fatalf("sign: %v", err)
	}

	defaultRoots := runtimebin.DefaultTrustRoots()
	defaultRoots[0] = runtimebin.TrustRoot{
		KeyID:     "runtime-manifest-v3",
		PublicKey: privateKey.Public().(ed25519.PublicKey),
	}

	return signedRelease{
		manifestPath:  manifestPath,
		signaturePath: signaturePath,
		archivePaths:  archives,
		recordPaths:   recordIndex,
		trustRoots:    defaultRoots,
	}
}
