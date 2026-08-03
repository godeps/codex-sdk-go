package runtimebin

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDownloadUpstreamArchiveSuccessAndErrors(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	packageDir := buildFakePackage(t, root, target)
	upstreamArchive := filepath.Join(root, "upstream.tar.gz")
	if _, err := PackageArchive(PackageOptions{
		PackageDir:     packageDir,
		OutputArchive:  upstreamArchive,
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}); err != nil {
		t.Fatal(err)
	}
	archiveBytes, err := os.ReadFile(upstreamArchive)
	if err != nil {
		t.Fatal(err)
	}

	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasSuffix(r.URL.Path, target.UpstreamAsset) {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	path, cleanup, err := DownloadUpstreamArchive(context.Background(), RepackOptions{
		RuntimeVersion:  DefaultRuntimeVersion,
		Target:          target,
		UpstreamBaseURL: server.URL,
		HTTPClient:      server.Client(),
	})
	if err != nil {
		t.Fatal(err)
	}
	defer cleanup()

	stageDir := filepath.Join(root, "stage")
	if err := stageUpstreamArchive(path, stageDir, target); err != nil {
		t.Fatalf("stageUpstreamArchive(): %v", err)
	}
	if _, err := os.Stat(filepath.Join(stageDir, "bin", target.Executable)); err != nil {
		t.Fatalf("staged executable missing: %v", err)
	}

	if _, _, err := DownloadUpstreamArchive(context.Background(), RepackOptions{
		RuntimeVersion:  DefaultRuntimeVersion,
		Target:          target,
		UpstreamBaseURL: "http://example.com/runtime",
	}); err == nil {
		t.Fatal("DownloadUpstreamArchive(http) unexpectedly succeeded")
	}
}

func TestRepackRuntimeCreatesEvidenceWithImplicitStageDir(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	root := t.TempDir()
	packageDir := buildFakePackage(t, root, target)
	upstreamArchive := filepath.Join(root, "upstream.tar.gz")
	if _, err := PackageArchive(PackageOptions{
		PackageDir:     packageDir,
		OutputArchive:  upstreamArchive,
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}); err != nil {
		t.Fatal(err)
	}
	record, err := RepackRuntime(context.Background(), RepackOptions{
		RuntimeVersion:  DefaultRuntimeVersion,
		SDKVersion:      DefaultSDKVersion,
		Target:          target,
		UpstreamArchive: upstreamArchive,
		OutputArchive:   filepath.Join(root, "repacked.tar.gz"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.UpstreamArchiveSHA256 == "" || record.ArchiveSHA256 == "" {
		t.Fatalf("unexpected repack record: %#v", record)
	}
}

func TestSignatureEvidenceAndExpectedPathsBranches(t *testing.T) {
	t.Parallel()

	darwinTarget, err := LookupTarget("darwin", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	windowsTarget, err := LookupTarget("windows", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	linuxTarget, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	stageDir := t.TempDir()
	for _, target := range []TargetSpec{darwinTarget, windowsTarget, linuxTarget} {
		packageDir := buildFakePackage(t, stageDir, target)
		evidence, err := CollectNativeSignatureEvidence(packageDir, target)
		if err != nil {
			t.Fatal(err)
		}
		if target.GOOS == "linux" && !strings.Contains(evidence.Reason, "not distributed") {
			t.Fatalf("linux evidence reason = %q", evidence.Reason)
		}
		if target.GOOS == "darwin" && !strings.Contains(evidence.Reason, "darwin host") {
			t.Fatalf("darwin evidence reason = %q", evidence.Reason)
		}
		if target.GOOS == "windows" && !strings.Contains(evidence.Reason, "windows host") {
			t.Fatalf("windows evidence reason = %q", evidence.Reason)
		}
	}

	darwinPaths := expectedUpstreamPaths(darwinTarget)
	if _, ok := darwinPaths["codex-resources/zsh/bin/zsh"]; !ok {
		t.Fatal("darwin expected paths missing zsh")
	}
	windowsPaths := expectedUpstreamPaths(windowsTarget)
	for _, path := range []string{"codex-resources/codex-command-runner.exe", "codex-resources/codex-windows-sandbox-setup.exe"} {
		if _, ok := windowsPaths[path]; !ok {
			t.Fatalf("windows expected paths missing %q", path)
		}
	}
	if codeModeHostExecutable(windowsTarget) != "codex-code-mode-host.exe" || rgExecutable(windowsTarget) != "rg.exe" {
		t.Fatal("windows executable helpers mismatch")
	}
}

func TestWriteReleaseEvidenceCopiesExplicitArtifacts(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	stageDir := buildFakePackage(t, t.TempDir(), target)
	licensePath := filepath.Join(t.TempDir(), "LICENSE")
	noticePath := filepath.Join(t.TempDir(), "NOTICE")
	sbomPath := filepath.Join(t.TempDir(), "sbom.json")
	provenancePath := filepath.Join(t.TempDir(), "provenance.json")
	signaturePath := filepath.Join(t.TempDir(), "native-signatures.json")
	for path, contents := range map[string]string{
		licensePath:    "license\n",
		noticePath:     "notice\n",
		sbomPath:       `{"sbom":true}` + "\n",
		provenancePath: `{"provenance":true}` + "\n",
		signaturePath:  `{"available":true,"verified":true}` + "\n",
	} {
		if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
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
		NativeSignPath: signaturePath,
	}, strings.Repeat("a", 64), 123); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{licenseFileName, noticeFileName, sbomFileName, provenanceFileName, nativeSignaturesFileName} {
		if _, err := os.Stat(filepath.Join(stageDir, path)); err != nil {
			t.Fatalf("missing release evidence file %q: %v", path, err)
		}
	}
	if err := copyFile(filepath.Join(stageDir, "copied.txt"), filepath.Join(stageDir, licenseFileName), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(stageDir, "missing.txt"), filepath.Join(stageDir, "does-not-exist"), 0o644); err == nil {
		t.Fatal("copyFile() unexpectedly succeeded for a missing source")
	}
}

func TestVerifyInstalledRootAndMetadataBranches(t *testing.T) {
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

	if _, err := VerifyInstalled(t.TempDir(), release.Manifest.RuntimeVersion, target, record); !errors.Is(err, ErrRuntimeNotInstalled) {
		t.Fatalf("VerifyInstalled(missing) = %v", err)
	}

	notDir := filepath.Join(t.TempDir(), "not-a-dir")
	if err := os.WriteFile(notDir, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, err := verifyInstalledRoot(notDir, record, release.Manifest.RuntimeVersion, target); err == nil {
		t.Fatal("verifyInstalledRoot(notDir) unexpectedly succeeded")
	}

	cacheRoot := t.TempDir()
	if err := extractArchive(release.ArchivePaths[target.Triple], cacheRoot, record, DefaultMaxArchiveBytes); err != nil {
		t.Fatal(err)
	}
	if err := writeInstallMetadata(cacheRoot, record, release.Manifest.RuntimeVersion, target); err != nil {
		t.Fatal(err)
	}
	metadataBytes, err := os.ReadFile(filepath.Join(cacheRoot, ".codex-sdk-runtime.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(metadataBytes, []byte(record.ArchiveSHA256)) {
		t.Fatalf("metadata did not include archive hash: %s", metadataBytes)
	}

	binaryPath := filepath.Join(cacheRoot, "bin", target.Executable)
	if err := os.WriteFile(binaryPath, []byte("corrupt"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, ok, err := verifyInstalledRoot(cacheRoot, record, release.Manifest.RuntimeVersion, target); err != nil || ok {
		t.Fatalf("verifyInstalledRoot(corrupt) = ok=%v err=%v", ok, err)
	}
}

func TestResolveInstallInputsAndRedirectPolicyBranches(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, err := resolveInstallInputs(context.Background(), InstallOptions{
		BaseURL: "http://example.com/runtime",
		Target:  &target,
	}, target); err == nil {
		t.Fatal("resolveInstallInputs(http) unexpectedly succeeded")
	}

	client := withSecureRedirectPolicy(&http.Client{})
	req := &http.Request{URL: &url.URL{Scheme: "https", Host: "example.com"}}
	via := []*http.Request{{URL: &url.URL{Scheme: "https", Host: "example.com"}}}
	if err := client.CheckRedirect(req, via); err != nil {
		t.Fatalf("same-host redirect unexpectedly failed: %v", err)
	}
	if err := client.CheckRedirect(&http.Request{URL: &url.URL{Scheme: "http", Host: "example.com"}}, via); err == nil {
		t.Fatal("non-https redirect unexpectedly succeeded")
	}
	if err := client.CheckRedirect(&http.Request{URL: &url.URL{Scheme: "https", Host: "other.example.com"}}, via); err == nil {
		t.Fatal("cross-host redirect unexpectedly succeeded")
	}

	client = withSecureRedirectPolicy(&http.Client{
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error { return errors.New("base redirect policy") },
	})
	if err := client.CheckRedirect(req, via); err == nil || !strings.Contains(err.Error(), "base redirect policy") {
		t.Fatalf("base redirect policy error = %v", err)
	}

	timeoutClient := withTimeoutHTTPClient(nil, time.Second)
	if timeoutClient.Timeout != time.Second {
		t.Fatalf("withTimeoutHTTPClient(nil) timeout = %v", timeoutClient.Timeout)
	}
	preserved := withTimeoutHTTPClient(&http.Client{Timeout: 2 * time.Second}, time.Second)
	if preserved.Timeout != 2*time.Second {
		t.Fatalf("withTimeoutHTTPClient(preserved) timeout = %v", preserved.Timeout)
	}
}

func TestManifestValidationSpecificFailures(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	manifest := release.Manifest
	if err := manifest.Validate(); err != nil {
		t.Fatalf("baseline manifest failed validation: %v", err)
	}

	cases := []struct {
		name   string
		mutate func(*Manifest)
	}{
		{
			name: "duplicate target",
			mutate: func(m *Manifest) {
				m.Targets[1] = m.Targets[0]
			},
		},
		{
			name: "invalid archive sha",
			mutate: func(m *Manifest) {
				m.Targets[0].ArchiveSHA256 = "bad"
			},
		},
		{
			name: "unsorted files",
			mutate: func(m *Manifest) {
				files := append([]ManifestFile(nil), m.Targets[0].Files...)
				files[0], files[1] = files[1], files[0]
				m.Targets[0].Files = files
			},
		},
		{
			name: "invalid file path",
			mutate: func(m *Manifest) {
				m.Targets[0].Files[0].Path = "../evil"
			},
		},
		{
			name: "missing metadata",
			mutate: func(m *Manifest) {
				m.SDKVersion = ""
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := manifest
			mutated.Targets = append([]ManifestTarget(nil), manifest.Targets...)
			tc.mutate(&mutated)
			if err := mutated.Validate(); err == nil {
				t.Fatalf("Validate() unexpectedly succeeded for %s", tc.name)
			}
		})
	}
}

func TestStageUpstreamArchiveRejectsBadEntries(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	validRoot := t.TempDir()
	validArchive := filepath.Join(validRoot, "valid.tar.gz")
	if _, err := PackageArchive(PackageOptions{
		PackageDir:     buildFakePackage(t, validRoot, target),
		OutputArchive:  validArchive,
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}); err != nil {
		t.Fatal(err)
	}

	t.Run("missing file", func(t *testing.T) {
		archive := filepath.Join(t.TempDir(), "missing.tar.gz")
		writeCustomArchive(t, archive, []tarEntry{{
			name:     "bin/codex",
			contents: []byte("codex"),
			mode:     0o755,
		}})
		if err := stageUpstreamArchive(archive, filepath.Join(t.TempDir(), "stage"), target); err == nil {
			t.Fatal("stageUpstreamArchive() unexpectedly succeeded with missing files")
		}
	})

	t.Run("unexpected file", func(t *testing.T) {
		archive := filepath.Join(t.TempDir(), "unexpected.tar.gz")
		writeCustomArchive(t, archive, []tarEntry{{
			name:     "bin/codex",
			contents: []byte("codex"),
			mode:     0o755,
		}, {
			name:     "bin/codex-code-mode-host",
			contents: []byte("host"),
			mode:     0o755,
		}, {
			name:     "codex-package.json",
			contents: []byte("{}"),
			mode:     0o644,
		}, {
			name:     "codex-path/rg",
			contents: []byte("rg"),
			mode:     0o755,
		}, {
			name:     "codex-resources/bwrap",
			contents: []byte("bwrap"),
			mode:     0o755,
		}, {
			name:     "codex-resources/zsh/bin/zsh",
			contents: []byte("zsh"),
			mode:     0o755,
		}, {
			name:     "surprise.txt",
			contents: []byte("nope"),
			mode:     0o644,
		}})
		if err := stageUpstreamArchive(archive, filepath.Join(t.TempDir(), "stage"), target); err == nil {
			t.Fatal("stageUpstreamArchive() unexpectedly succeeded with an unexpected file")
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		archive := filepath.Join(t.TempDir(), "invalid-path.tar.gz")
		writeCustomArchive(t, archive, []tarEntry{{
			name:     "../evil",
			contents: []byte("nope"),
			mode:     0o644,
		}})
		if err := stageUpstreamArchive(archive, filepath.Join(t.TempDir(), "stage"), target); err == nil {
			t.Fatal("stageUpstreamArchive() unexpectedly succeeded with a traversal path")
		}
	})

	t.Run("unsupported type", func(t *testing.T) {
		archive := filepath.Join(t.TempDir(), "symlink.tar.gz")
		writeCustomArchive(t, archive, []tarEntry{{
			name:     "bin/codex",
			typeflag: 2,
			linkname: "elsewhere",
			mode:     0o755,
		}})
		if err := stageUpstreamArchive(archive, filepath.Join(t.TempDir(), "stage"), target); err == nil {
			t.Fatal("stageUpstreamArchive() unexpectedly succeeded with a symlink")
		}
	})

	if err := stageUpstreamArchive(validArchive, filepath.Join(t.TempDir(), "stage"), target); err != nil {
		t.Fatalf("valid stageUpstreamArchive() failed: %v", err)
	}
}

func TestExtractArchiveRejectsMismatchBranches(t *testing.T) {
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
	archivePath := release.ArchivePaths[target.Triple]

	badSHA := record
	badSHA.ArchiveSHA256 = strings.Repeat("0", 64)
	if err := extractArchive(archivePath, filepath.Join(t.TempDir(), "sha"), badSHA, DefaultMaxArchiveBytes); err == nil {
		t.Fatal("extractArchive() unexpectedly succeeded with a bad archive hash")
	}

	badSize := record
	badSize.ArchiveSize++
	if err := extractArchive(archivePath, filepath.Join(t.TempDir(), "size"), badSize, DefaultMaxArchiveBytes); err == nil {
		t.Fatal("extractArchive() unexpectedly succeeded with a bad archive size")
	}

	if err := extractArchive(archivePath, filepath.Join(t.TempDir(), "limit"), record, 1); err == nil {
		t.Fatal("extractArchive() unexpectedly succeeded with a tiny size limit")
	}

	missingFile := record
	missingFile.Files = append(append([]ManifestFile(nil), record.Files...), ManifestFile{
		Path:   "bin/missing",
		SHA256: strings.Repeat("a", 64),
		Size:   1,
		Mode:   0o755,
		Role:   "executable",
	})
	if err := extractArchive(archivePath, filepath.Join(t.TempDir(), "missing"), missingFile, DefaultMaxArchiveBytes); err == nil {
		t.Fatal("extractArchive() unexpectedly succeeded with a missing expected file")
	}
}

func TestWriteEvidenceAndSignatureHelpers(t *testing.T) {
	t.Parallel()

	path := filepath.Join(t.TempDir(), "evidence.json")
	if err := WriteEvidence(path, map[string]string{"status": "ok"}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "\"status\": \"ok\"") {
		t.Fatalf("unexpected evidence payload: %s", body)
	}

	if _, err := SignManifest([]byte("manifest"), "runtime-manifest-v3", []byte("short")); err == nil {
		t.Fatal("SignManifest() unexpectedly accepted a short private key")
	}

	badSigPath := filepath.Join(t.TempDir(), "bad.sig")
	if err := os.WriteFile(badSigPath, []byte(`{"keyId":"x","algorithm":"ed25519","signature":"!!!"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadSignatureEnvelope(badSigPath); err == nil {
		t.Fatal("LoadSignatureEnvelope() unexpectedly accepted bad base64")
	}

	if got := decodePublicKey(base64.StdEncoding.EncodeToString(bytes.Repeat([]byte{3}, ed25519.PublicKeySize))); len(got) != ed25519.PublicKeySize {
		t.Fatalf("decodePublicKey() len = %d", len(got))
	}
}
