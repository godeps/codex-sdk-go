package runtimebin

import (
	"context"
	"crypto/tls"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestTargetAndTrustHelpers(t *testing.T) {
	t.Parallel()

	targets := SupportedTargets()
	if len(targets) != 6 {
		t.Fatalf("SupportedTargets() len = %d", len(targets))
	}
	targets[0].Triple = "mutated"
	if SupportedTargets()[0].Triple == "mutated" {
		t.Fatal("SupportedTargets() did not return a copy")
	}
	shuffled := []TargetSpec{targets[5], targets[0], targets[3]}
	SortTargets(shuffled)
	if !slices.IsSortedFunc(shuffled, func(a, b TargetSpec) int {
		return strings.Compare(a.Triple, b.Triple)
	}) {
		t.Fatalf("SortTargets() did not sort by triple: %#v", shuffled)
	}
	if ArchiveName("0.144.4", TargetSpec{GOOS: "linux", GOARCH: "amd64"}) != "codex-sdk-go-runtime_0.144.4_linux_amd64.tar.gz" {
		t.Fatal("ArchiveName() mismatch")
	}
	if joinRuntimePath("/tmp/root", "bin") != "/tmp/root/bin" {
		t.Fatal("joinRuntimePath() slash mismatch")
	}
	if joinRuntimePath(`C:\tmp\root`, "bin") != `C:\tmp\root\bin` {
		t.Fatal("joinRuntimePath() windows mismatch")
	}
	if pathSeparator("/tmp") != '/' || pathSeparator(`C:\tmp`) != '\\' {
		t.Fatal("pathSeparator() mismatch")
	}
	if binaryBaseName("codex.exe") != "codex" || binaryBaseName("codex") != "codex" {
		t.Fatal("binaryBaseName() mismatch")
	}
	current, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	if targetOrCurrent(nil).Triple != current.Triple {
		t.Fatal("targetOrCurrent(nil) mismatch")
	}
	if targetOrCurrent(&targets[1]).Triple != targets[1].Triple {
		t.Fatal("targetOrCurrent(explicit) mismatch")
	}
	roots := DefaultTrustRoots()
	if len(roots) != 1 || roots[0].KeyID != "runtime-manifest-v3" {
		t.Fatalf("DefaultTrustRoots() = %#v", roots)
	}
}

func TestManifestValidationAndLoadErrors(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	valid := Manifest{
		SchemaVersion:  ManifestSchemaVersion,
		SDKVersion:     "v0.2.0",
		RuntimeVersion: DefaultRuntimeVersion,
		UpstreamRepo:   DefaultUpstreamRepo,
		UpstreamTag:    "rust-v0.144.4",
		Targets: []ManifestTarget{{
			GOOS:                  target.GOOS,
			GOARCH:                target.GOARCH,
			Triple:                target.Triple,
			Executable:            target.Executable,
			UpstreamAsset:         target.UpstreamAsset,
			UpstreamArchiveSHA256: strings.Repeat("a", 64),
			UpstreamArchiveSize:   1,
			ArchiveName:           "archive.tar.gz",
			ArchiveSHA256:         strings.Repeat("b", 64),
			ArchiveSize:           2,
			Files: []ManifestFile{{
				Path:   "bin/codex",
				SHA256: strings.Repeat("c", 64),
				Size:   3,
				Mode:   0o755,
				Role:   "executable",
			}},
		}},
	}
	if err := valid.Validate(); err == nil {
		t.Fatal("expected single-target manifest to fail target count gate")
	}

	root := t.TempDir()
	badManifestPath := filepath.Join(root, "manifest.json")
	if err := os.WriteFile(badManifestPath, []byte("{not-json"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := LoadManifest(badManifestPath); err == nil {
		t.Fatal("expected invalid manifest JSON to fail")
	}
	badSigPath := filepath.Join(root, "manifest.json.sig")
	if err := os.WriteFile(badSigPath, []byte(`{"keyId":"","algorithm":"rsa","signature":"??"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSignatureEnvelope(badSigPath); err == nil {
		t.Fatal("expected invalid signature envelope to fail")
	}
	if _, err := LookupTarget("plan9", "amd64"); err == nil {
		t.Fatal("expected unsupported goos/goarch to fail")
	}
	if _, err := LookupTargetByTriple("bad-triple"); err == nil {
		t.Fatal("expected unsupported triple to fail")
	}
}

func TestResolverHelpers(t *testing.T) {
	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveRuntime(ResolveOptions{
		ExplicitBinary: "/missing/codex",
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         &target,
	}); err == nil {
		t.Fatal("expected missing explicit path to fail")
	}

	pathDir := t.TempDir()
	binary := filepath.Join(pathDir, "codex")
	if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	resolved, err := ResolveRuntime(ResolveOptions{
		Env:            map[string]string{"PATH": pathDir},
		RuntimeVersion: DefaultRuntimeVersion,
		AllowPATH:      true,
		Target:         &target,
	})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Source != "path" {
		t.Fatalf("ResolveRuntime() source = %q", resolved.Source)
	}

	t.Setenv("OS", "Windows_NT")
	env := map[string]string{"PATH": "A", "Path": "B"}
	if PathEnvKey(env) != "Path" {
		t.Fatalf("PathEnvKey() = %q", PathEnvKey(env))
	}
}

func TestInstallerHelperFunctions(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := managedInstallRoot(t.TempDir(), "", target); err == nil {
		t.Fatal("expected empty managed version to fail")
	}
	root, err := managedInstallRoot(t.TempDir(), DefaultRuntimeVersion, target)
	if err != nil || !strings.Contains(root, target.Triple) {
		t.Fatalf("managedInstallRoot() = %q, %v", root, err)
	}
	if InstallRoot(t.TempDir(), DefaultRuntimeVersion, target) == "" {
		t.Fatal("InstallRoot() empty")
	}
	if got, ok := safeArchivePath("../evil"); ok || got != "" {
		t.Fatal("safeArchivePath() accepted traversal")
	}
	if got, ok := safeArchivePath("bin/codex"); !ok || got != "bin/codex" {
		t.Fatalf("safeArchivePath() = %q, %v", got, ok)
	}
	if err := enforceSameHostPolicy("https://a.test/x", "https://b.test/y"); err == nil {
		t.Fatal("expected host mismatch to fail")
	}
	if err := enforceSameHostPolicy("http://a.test/x"); err == nil {
		t.Fatal("expected non-https to fail")
	}
	if !strings.Contains(sanitizeURL("https://user:pass@example.com/x?token=secret#frag"), "https://example.com/x") {
		t.Fatal("sanitizeURL() mismatch")
	}
	requestErr := &url.Error{Op: "Get", URL: "https://user:pass@example.com/x?token=secret", Err: errors.New("boom")}
	if strings.Contains(sanitizeRequestError(requestErr).Error(), "secret") {
		t.Fatal("sanitizeRequestError() leaked secret")
	}

	lockPath := filepath.Join(t.TempDir(), "stale.lock")
	if err := os.WriteFile(lockPath, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	old := time.Now().Add(-time.Hour)
	if err := os.Chtimes(lockPath, old, old); err != nil {
		t.Fatal(err)
	}
	release, err := acquireLock(context.Background(), lockPath, time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	release()
}

func TestDownloadURLAndResolveInstallInputsOnline(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	record, err := release.Manifest.Target(target.Triple)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/codex-sdk-go-runtime-manifest.json", func(w http.ResponseWriter, _ *http.Request) {
		bytes, _ := os.ReadFile(release.ManifestPath)
		_, _ = w.Write(bytes)
	})
	mux.HandleFunc("/codex-sdk-go-runtime-manifest.json.sig", func(w http.ResponseWriter, _ *http.Request) {
		bytes, _ := os.ReadFile(release.SignaturePath)
		_, _ = w.Write(bytes)
	})
	mux.HandleFunc("/"+record.ArchiveName, func(w http.ResponseWriter, _ *http.Request) {
		bytes, _ := os.ReadFile(release.ArchivePaths[target.Triple])
		_, _ = w.Write(bytes)
	})
	server := httptest.NewTLSServer(mux)
	defer server.Close()

	client := server.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	manifest, manifestBytes, envelope, archivePath, cleanup, err := resolveInstallInputs(context.Background(), InstallOptions{
		RuntimeVersion:  DefaultRuntimeVersion,
		Target:          &target,
		BaseURL:         server.URL,
		HTTPClient:      client,
		MaxArchiveBytes: DefaultMaxArchiveBytes,
	}, target)
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if len(manifestBytes) == 0 || archivePath == "" || envelope.KeyID == "" || manifest.RuntimeVersion != DefaultRuntimeVersion {
		t.Fatalf("resolveInstallInputs() incomplete result: %#v %#v %q", manifest, envelope, archivePath)
	}

	statusServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	defer statusServer.Close()
	statusClient := statusServer.Client()
	statusClient.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	if _, err := downloadURL(context.Background(), statusClient, statusServer.URL, 32); err == nil {
		t.Fatal("expected status error")
	}
}

func TestRepackHelperFunctions(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	upstreamDir := buildFakePackage(t, root, target)
	upstreamArchive := filepath.Join(root, "upstream.tar.gz")
	if _, err := PackageArchive(PackageOptions{PackageDir: upstreamDir, OutputArchive: upstreamArchive, RuntimeVersion: DefaultRuntimeVersion, Target: target}); err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/"+target.UpstreamAsset, func(w http.ResponseWriter, _ *http.Request) {
		bytes, _ := os.ReadFile(upstreamArchive)
		_, _ = w.Write(bytes)
	})
	server := httptest.NewTLSServer(mux)
	defer server.Close()
	client := server.Client()
	client.Transport.(*http.Transport).TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec
	downloaded, cleanup, err := DownloadUpstreamArchive(context.Background(), RepackOptions{
		RuntimeVersion:  DefaultRuntimeVersion,
		Target:          target,
		UpstreamBaseURL: server.URL,
		HTTPClient:      client,
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()
	if _, err := os.Stat(downloaded); err != nil {
		t.Fatal(err)
	}

	if _, ok := expectedUpstreamPaths(target)["codex-resources/zsh/bin/zsh"]; !ok {
		t.Fatal("expectedUpstreamPaths() missing zsh for linux")
	}
	if allVerified([]SignatureRecord{{Status: "verified"}, {Status: "verified"}}) != true || allVerified([]SignatureRecord{{Status: "failed"}}) != false {
		t.Fatal("allVerified() mismatch")
	}
	if codeModeHostExecutable(target) != "codex-code-mode-host" || rgExecutable(target) != "rg" {
		t.Fatal("linux executable helpers mismatch")
	}

	stageDir := t.TempDir()
	licensePath := filepath.Join(root, "LICENSE.custom")
	noticePath := filepath.Join(root, "NOTICE.custom")
	sbomPath := filepath.Join(root, "sbom.custom.json")
	provenancePath := filepath.Join(root, "prov.custom.json")
	signPath := filepath.Join(root, "sign.custom.json")
	for _, file := range []string{licensePath, noticePath, sbomPath, provenancePath, signPath} {
		if err := os.WriteFile(file, []byte(filepath.Base(file)), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := writeReleaseEvidence(stageDir, RepackOptions{
		SDKVersion:     DefaultSDKVersion,
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
		LicensePath:    licensePath,
		NoticePath:     noticePath,
		SBOMPath:       sbomPath,
		ProvenancePath: provenancePath,
		NativeSignPath: signPath,
	}, strings.Repeat("d", 64), 123); err != nil {
		t.Fatal(err)
	}
	for _, name := range []string{licenseFileName, noticeFileName, sbomFileName, provenanceFileName, nativeSignaturesFileName} {
		if _, err := os.Stat(filepath.Join(stageDir, name)); err != nil {
			t.Fatalf("expected release evidence file %s: %v", name, err)
		}
	}
	if err := copyFile(filepath.Join(stageDir, "copied.txt"), licensePath, 0o644); err != nil {
		t.Fatal(err)
	}
	if hashBytes([]byte("abc")) == "" {
		t.Fatal("hashBytes() empty")
	}
	if err := WriteEvidence(filepath.Join(stageDir, "evidence.json"), map[string]string{"ok": "1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := CollectNativeSignatureEvidence(stageDir, target); err != nil {
		t.Fatal(err)
	}
}
