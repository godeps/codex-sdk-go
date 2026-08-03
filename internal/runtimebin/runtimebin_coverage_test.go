package runtimebin

import (
	"archive/tar"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallRootPanicsOnInvalidManagedVersion(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("InstallRoot() unexpectedly accepted an invalid version")
		}
	}()
	_ = InstallRoot(t.TempDir(), "", target)
}

func TestInstallRejectsManifestVersionMismatch(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}

	_, err = Install(context.Background(), InstallOptions{
		CacheRoot:      t.TempDir(),
		RuntimeVersion: "0.144.5",
		Target:         &target,
		ManifestPath:   release.ManifestPath,
		SignaturePath:  release.SignaturePath,
		ArchivePath:    release.ArchivePaths[target.Triple],
		TrustRoots:     release.TrustRoots,
	})
	if err == nil || !strings.Contains(err.Error(), "does not match requested") {
		t.Fatalf("Install() manifest version mismatch = %v", err)
	}
}

func TestResolveInstallInputsOfflineErrors(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}

	missingManifest := filepath.Join(t.TempDir(), "missing-manifest.json")
	if _, _, _, _, _, err := resolveInstallInputs(context.Background(), InstallOptions{
		ManifestPath:  missingManifest,
		SignaturePath: missingManifest + ".sig",
		ArchivePath:   missingManifest + ".tar.gz",
	}, target); err == nil {
		t.Fatal("resolveInstallInputs() unexpectedly accepted a missing offline manifest")
	}

	root := t.TempDir()
	manifestPath := filepath.Join(root, "manifest.json")
	signaturePath := filepath.Join(root, "manifest.json.sig")
	archivePath := filepath.Join(root, "archive.tar.gz")
	if err := os.WriteFile(manifestPath, []byte(`{"schemaVersion":1}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(signaturePath, []byte(`{"keyId":"x","algorithm":"ed25519","signature":"!!!"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(archivePath, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, err := resolveInstallInputs(context.Background(), InstallOptions{
		ManifestPath:  manifestPath,
		SignaturePath: signaturePath,
		ArchivePath:   archivePath,
	}, target); err == nil {
		t.Fatal("resolveInstallInputs() unexpectedly accepted an invalid offline manifest")
	}
}

func TestResolveInstallInputsOnlineURLValidation(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}

	cases := []struct {
		name string
		opts InstallOptions
		want string
	}{
		{
			name: "base url must be https",
			opts: InstallOptions{BaseURL: "http://example.com/runtime"},
			want: "https base URL",
		},
		{
			name: "manifest url must be https",
			opts: InstallOptions{
				ManifestURL:  "http://example.com/manifest.json",
				SignatureURL: "https://example.com/manifest.json.sig",
				BaseURL:      "https://example.com/runtime",
			},
			want: "https manifest URLs",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, _, _, _, _, err := resolveInstallInputs(context.Background(), tc.opts, target)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("resolveInstallInputs() error = %v, want substring %q", err, tc.want)
			}
		})
	}

	release := buildFakeRelease(t)
	manifestBytes, err := os.ReadFile(release.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	signatureBytes, err := os.ReadFile(release.SignaturePath)
	if err != nil {
		t.Fatal(err)
	}
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/manifest.json":
			_, _ = w.Write(manifestBytes)
		case "/manifest.json.sig":
			_, _ = w.Write(signatureBytes)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	_, _, _, _, _, err = resolveInstallInputs(context.Background(), InstallOptions{
		ManifestURL:     server.URL + "/manifest.json",
		SignatureURL:    server.URL + "/manifest.json.sig",
		BaseURL:         "http://example.com/runtime",
		HTTPClient:      server.Client(),
		MaxArchiveBytes: 1 << 20,
	}, target)
	if err == nil || !strings.Contains(err.Error(), "https archive URLs") {
		t.Fatalf("resolveInstallInputs() archive URL validation = %v", err)
	}
}

func TestDownloadURLRejectsInvalidRequestAndOversizedBodies(t *testing.T) {
	t.Parallel()

	if _, err := downloadURL(context.Background(), &http.Client{}, "://bad-url", 32); err == nil {
		t.Fatal("downloadURL() unexpectedly accepted an invalid URL")
	}

	t.Run("content-length", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Set("Content-Length", "4096")
			_, _ = w.Write([]byte("ok"))
		}))
		defer server.Close()

		if _, err := downloadURL(context.Background(), server.Client(), server.URL, 8); err == nil || !strings.Contains(err.Error(), "exceeds size limit") {
			t.Fatalf("downloadURL() content-length guard = %v", err)
		}
	})

	t.Run("body-bytes", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			w.Header().Del("Content-Length")
			_, _ = w.Write([]byte("0123456789"))
		}))
		defer server.Close()

		if _, err := downloadURL(context.Background(), server.Client(), server.URL, 4); err == nil || !strings.Contains(err.Error(), "exceeds size limit") {
			t.Fatalf("downloadURL() body limit guard = %v", err)
		}
	})
}

func TestAcquireLockCancelsWhileWaiting(t *testing.T) {
	t.Parallel()

	lockPath := filepath.Join(t.TempDir(), "runtime.lock")
	if err := os.WriteFile(lockPath, []byte("held"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	release, err := acquireLock(ctx, lockPath, 0)
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("acquireLock() error = %v", err)
	}
	if release != nil {
		t.Fatal("acquireLock() returned a release func on cancellation")
	}
}

func TestManifestValidationAdditionalFailures(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	cases := []struct {
		name   string
		mutate func(*Manifest)
		want   string
	}{
		{
			name: "schema version",
			mutate: func(m *Manifest) {
				m.SchemaVersion++
			},
			want: "schemaVersion",
		},
		{
			name: "runtime version",
			mutate: func(m *Manifest) {
				m.RuntimeVersion = "bad"
			},
			want: "unsupported runtime version",
		},
		{
			name: "target metadata mismatch",
			mutate: func(m *Manifest) {
				m.Targets[0].Executable = "wrong"
			},
			want: "does not match",
		},
		{
			name: "missing archive metadata",
			mutate: func(m *Manifest) {
				m.Targets[0].ArchiveName = ""
			},
			want: "missing archive metadata",
		},
		{
			name: "invalid upstream sha",
			mutate: func(m *Manifest) {
				m.Targets[0].UpstreamArchiveSHA256 = "bad"
			},
			want: "invalid upstream archive sha256",
		},
		{
			name: "invalid upstream size",
			mutate: func(m *Manifest) {
				m.Targets[0].UpstreamArchiveSHA256 = strings.Repeat("a", 64)
				m.Targets[0].UpstreamArchiveSize = 0
			},
			want: "invalid upstream archive size",
		},
		{
			name: "no files",
			mutate: func(m *Manifest) {
				m.Targets[0].Files = nil
			},
			want: "has no files",
		},
		{
			name: "invalid file sha",
			mutate: func(m *Manifest) {
				m.Targets[0].Files[0].SHA256 = "bad"
			},
			want: "invalid file sha256",
		},
		{
			name: "incomplete file metadata",
			mutate: func(m *Manifest) {
				m.Targets[0].Files[0].Mode = 0
			},
			want: "incomplete file metadata",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			mutated := release.Manifest
			mutated.Targets = append([]ManifestTarget(nil), release.Manifest.Targets...)
			files := append([]ManifestFile(nil), release.Manifest.Targets[0].Files...)
			mutated.Targets[0].Files = files
			tc.mutate(&mutated)
			if err := mutated.Validate(); err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("Validate() error = %v, want substring %q", err, tc.want)
			}
		})
	}
}

func TestManifestLookupAndLoadMissingFiles(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	if _, err := release.Manifest.Target("missing-target"); err == nil {
		t.Fatal("Manifest.Target() unexpectedly found a missing target")
	}
	if _, _, err := LoadManifest(filepath.Join(t.TempDir(), "missing.json")); err == nil {
		t.Fatal("LoadManifest() unexpectedly accepted a missing file")
	}
	if _, err := LoadSignatureEnvelope(filepath.Join(t.TempDir(), "missing.sig")); err == nil {
		t.Fatal("LoadSignatureEnvelope() unexpectedly accepted a missing file")
	}
}

func TestPackageArchiveAndHelpersRejectUnexpectedInputs(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	packageDir := buildFakePackage(t, t.TempDir(), target)
	if err := os.WriteFile(filepath.Join(packageDir, "unexpected.txt"), []byte("nope"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := PackageArchive(PackageOptions{
		PackageDir:     packageDir,
		OutputArchive:  filepath.Join(t.TempDir(), "archive.tar.gz"),
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}); err == nil || !strings.Contains(err.Error(), "unexpected package file") {
		t.Fatalf("PackageArchive() error = %v", err)
	}

	if _, err := BuildManifest(DefaultSDKVersion, "bad", DefaultUpstreamRepo, releaseLikeTargets(t)); err == nil {
		t.Fatal("BuildManifest() unexpectedly accepted an invalid runtime version")
	}
	if _, _, err := fileSHA256AndSize(filepath.Join(t.TempDir(), "missing.bin")); err == nil {
		t.Fatal("fileSHA256AndSize() unexpectedly accepted a missing file")
	}
}

func TestCollectPackageFilesClassifiesReleaseEvidenceRoles(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	root := buildFakePackage(t, t.TempDir(), target)
	extraFiles := map[string]string{
		licenseFileName:          "license\n",
		noticeFileName:           "notice\n",
		sbomFileName:             "{}\n",
		provenanceFileName:       "{}\n",
		nativeSignaturesFileName: "{}\n",
	}
	for name, body := range extraFiles {
		if err := os.WriteFile(filepath.Join(root, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	files, err := collectPackageFiles(root)
	if err != nil {
		t.Fatal(err)
	}
	roles := map[string]string{}
	for _, file := range files {
		roles[file.Path] = file.Role
	}
	want := map[string]string{
		licenseFileName:          "license",
		noticeFileName:           "notice",
		sbomFileName:             "sbom",
		provenanceFileName:       "provenance",
		nativeSignaturesFileName: "native-signature-evidence",
	}
	for path, role := range want {
		if roles[path] != role {
			t.Fatalf("collectPackageFiles() role for %s = %q, want %q", path, roles[path], role)
		}
	}
}

func TestWriteJSONFileRejectsUnsupportedValue(t *testing.T) {
	t.Parallel()

	if err := writeJSONFile(filepath.Join(t.TempDir(), "bad.json"), map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("writeJSONFile() unexpectedly accepted an unsupported value")
	}
}

func TestVerifyInstalledRootReturnsFalseWhenFileMissing(t *testing.T) {
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
	root := t.TempDir()
	if err := extractArchive(release.ArchivePaths[target.Triple], root, record, DefaultMaxArchiveBytes); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(root, filepath.FromSlash(record.Files[0].Path))); err != nil {
		t.Fatal(err)
	}
	_, ok, err := verifyInstalledRoot(root, record, release.Manifest.RuntimeVersion, target)
	if err != nil || ok {
		t.Fatalf("verifyInstalledRoot() = ok=%v err=%v", ok, err)
	}
}

func TestResolveRuntimeErrorBranches(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}

	if _, err := ResolveRuntime(ResolveOptions{
		Env:            map[string]string{"CODEX_RUNTIME_PATH": filepath.Join(t.TempDir(), "missing")},
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         &target,
	}); err == nil || !strings.Contains(err.Error(), "CODEX_RUNTIME_PATH") {
		t.Fatalf("ResolveRuntime() env error = %v", err)
	}

	if _, err := ResolveRuntime(ResolveOptions{
		RuntimeVersion: DefaultRuntimeVersion,
		AllowPATH:      false,
		Target:         &target,
	}); !errors.Is(err, ErrRuntimeNotInstalled) {
		t.Fatalf("ResolveRuntime() missing install = %v", err)
	}

	if _, err := ResolveRuntime(ResolveOptions{
		RuntimeVersion: "bad",
		Target:         &target,
	}); err == nil || !strings.Contains(err.Error(), "unsupported runtime version") {
		t.Fatalf("ResolveRuntime() invalid version = %v", err)
	}
}

func TestResolverAndTargetHelpers(t *testing.T) {
	env := map[string]string{"PATH": strings.Join([]string{"/a", "/b"}, string(os.PathListSeparator))}
	PrependPathDirs(env, []string{"", "/x", "/x", "/a"})
	if env["PATH"] != strings.Join([]string{"/x", "/a", "/b"}, string(os.PathListSeparator)) {
		t.Fatalf("PrependPathDirs() = %q", env["PATH"])
	}

	t.Setenv("OS", "Windows_NT")
	caseFolded := map[string]string{"PaTh": "C:\\Windows"}
	if PathEnvKey(caseFolded) != "PaTh" {
		t.Fatalf("PathEnvKey() = %q", PathEnvKey(caseFolded))
	}

	if joinRuntimePath("/tmp/root/", "bin") != "/tmp/root/bin" {
		t.Fatal("joinRuntimePath() did not preserve trailing slash")
	}
	if len(TargetPathDirs("/tmp/root")) != 2 {
		t.Fatal("TargetPathDirs() unexpected length")
	}
	if got := binaryBaseName("tool.exe"); got != "tool" {
		t.Fatalf("binaryBaseName() = %q", got)
	}
	if len(environmentMap()) == 0 {
		t.Fatal("environmentMap() unexpectedly empty")
	}
}

func TestTrustAndEvidenceHelperBranches(t *testing.T) {
	t.Parallel()

	didPanic := false
	func() {
		defer func() {
			didPanic = recover() != nil
		}()
		_ = decodePublicKey("%%%")
	}()
	if !didPanic {
		t.Fatal("decodePublicKey() unexpectedly accepted invalid base64")
	}

	release := buildFakeRelease(t)
	if err := VerifyManifestSignature(release.ManifestBytes, SignatureEnvelope{
		KeyID:     release.SignatureEnvelope.KeyID,
		Algorithm: "rsa",
		Signature: release.SignatureEnvelope.Signature,
	}, release.TrustRoots); err == nil || !strings.Contains(err.Error(), "unsupported signature algorithm") {
		t.Fatalf("VerifyManifestSignature() algorithm error = %v", err)
	}
	if err := VerifyManifestSignature(release.ManifestBytes, SignatureEnvelope{
		KeyID:     release.SignatureEnvelope.KeyID,
		Algorithm: SignatureAlgorithm,
		Signature: "!!!",
	}, release.TrustRoots); err == nil || !strings.Contains(err.Error(), "invalid signature encoding") {
		t.Fatalf("VerifyManifestSignature() encoding error = %v", err)
	}
	if err := WriteEvidence(filepath.Join(t.TempDir(), "bad.json"), map[string]any{"bad": make(chan int)}); err == nil {
		t.Fatal("WriteEvidence() unexpectedly accepted an unsupported value")
	}
}

func TestRepackRuntimeAndDownloadErrorBranches(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if _, _, err := DownloadUpstreamArchive(context.Background(), RepackOptions{
		RuntimeVersion: "bad",
		Target:         target,
	}); err == nil {
		t.Fatal("DownloadUpstreamArchive() unexpectedly accepted an invalid runtime version")
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

	if _, err := RepackRuntime(context.Background(), RepackOptions{
		Target:          target,
		UpstreamArchive: filepath.Join(root, "missing.tar.gz"),
		OutputArchive:   filepath.Join(root, "repacked.tar.gz"),
	}); err == nil {
		t.Fatal("RepackRuntime() unexpectedly accepted a missing upstream archive")
	}
}

func TestInstallUsesDefaultVersionAndRejectsUntrustedDefaultRoots(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	_, err = Install(context.Background(), InstallOptions{
		CacheRoot:     t.TempDir(),
		Target:        &target,
		ManifestPath:  release.ManifestPath,
		SignaturePath: release.SignaturePath,
		ArchivePath:   release.ArchivePaths[target.Triple],
	})
	if err == nil || !strings.Contains(err.Error(), "untrusted manifest signing key") {
		t.Fatalf("Install() default root rejection = %v", err)
	}
}

func TestInstallUsesDefaultVersionWithExplicitTrustRoots(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	result, err := Install(context.Background(), InstallOptions{
		CacheRoot:     t.TempDir(),
		Target:        &target,
		ManifestPath:  release.ManifestPath,
		SignaturePath: release.SignaturePath,
		ArchivePath:   release.ArchivePaths[target.Triple],
		TrustRoots:    release.TrustRoots,
	})
	if err != nil {
		t.Fatalf("Install() defaulted offline install failed: %v", err)
	}
	if result.Runtime.Version != DefaultRuntimeVersion || result.Runtime.Source != "managed" {
		t.Fatalf("Install() defaulted result = %#v", result.Runtime)
	}
}

func TestRepackEvidenceWriters(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	stageDir := buildFakePackage(t, t.TempDir(), target)
	if err := copyOrDefaultLicense(stageDir, ""); err != nil {
		t.Fatal(err)
	}
	licenseBody, err := os.ReadFile(filepath.Join(stageDir, licenseFileName))
	if err != nil {
		t.Fatal(err)
	}
	if len(licenseBody) == 0 {
		t.Fatal("copyOrDefaultLicense() wrote an empty license")
	}
	if err := copyOrGeneratedNotice(stageDir, RepackOptions{
		SDKVersion:     DefaultSDKVersion,
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeSBOM(stageDir, RepackOptions{
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}); err != nil {
		t.Fatal(err)
	}
	if err := writeProvenance(stageDir, RepackOptions{
		SDKVersion:     DefaultSDKVersion,
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}, strings.Repeat("a", 64), 123); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(stageDir, sbomFileName)); err != nil {
		t.Fatalf("writeSBOM() missing output: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stageDir, provenanceFileName)); err != nil {
		t.Fatalf("writeProvenance() missing output: %v", err)
	}
	if err := copyOrGeneratedNotice(stageDir, RepackOptions{
		RuntimeVersion: "bad",
		Target:         target,
	}); err == nil {
		t.Fatal("copyOrGeneratedNotice() unexpectedly accepted an invalid runtime version")
	}
}

func TestRepackRuntimeDownloadsUpstreamWhenRequested(t *testing.T) {
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
		if r.URL.Path != "/"+target.UpstreamAsset {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write(archiveBytes)
	}))
	defer server.Close()

	record, err := RepackRuntime(context.Background(), RepackOptions{
		Target:           target,
		DownloadUpstream: true,
		UpstreamBaseURL:  server.URL,
		HTTPClient:       server.Client(),
		OutputArchive:    filepath.Join(root, "repacked.tar.gz"),
	})
	if err != nil {
		t.Fatalf("RepackRuntime() download path failed: %v", err)
	}
	if record.UpstreamArchiveSHA256 == "" || record.ArchiveName == "" {
		t.Fatalf("RepackRuntime() incomplete record: %#v", record)
	}
}

func TestWriteReleaseEvidenceRejectsMissingExplicitArtifacts(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	stageDir := buildFakePackage(t, t.TempDir(), target)
	cases := []RepackOptions{
		{RuntimeVersion: DefaultRuntimeVersion, Target: target, LicensePath: filepath.Join(stageDir, "missing-license")},
		{RuntimeVersion: DefaultRuntimeVersion, Target: target, NoticePath: filepath.Join(stageDir, "missing-notice")},
		{RuntimeVersion: DefaultRuntimeVersion, Target: target, SBOMPath: filepath.Join(stageDir, "missing-sbom")},
		{RuntimeVersion: DefaultRuntimeVersion, Target: target, ProvenancePath: filepath.Join(stageDir, "missing-provenance")},
		{RuntimeVersion: DefaultRuntimeVersion, Target: target, NativeSignPath: filepath.Join(stageDir, "missing-native-sign")},
	}
	for _, opts := range cases {
		if err := writeReleaseEvidence(filepath.Join(t.TempDir(), "stage"), opts, strings.Repeat("a", 64), 1); err == nil {
			t.Fatalf("writeReleaseEvidence() unexpectedly accepted opts %#v", opts)
		}
	}
}

func TestDownloadUpstreamArchiveAndStageErrorBranches(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}

	t.Run("download http error", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
			http.NotFound(w, nil)
		}))
		defer server.Close()
		if _, _, err := DownloadUpstreamArchive(context.Background(), RepackOptions{
			RuntimeVersion:  DefaultRuntimeVersion,
			Target:          target,
			UpstreamBaseURL: server.URL,
			HTTPClient:      server.Client(),
		}); err == nil {
			t.Fatal("DownloadUpstreamArchive() unexpectedly accepted a 404 upstream response")
		}
	})

	t.Run("stage missing archive", func(t *testing.T) {
		if err := stageUpstreamArchive(filepath.Join(t.TempDir(), "missing.tar.gz"), t.TempDir(), target); err == nil {
			t.Fatal("stageUpstreamArchive() unexpectedly accepted a missing archive")
		}
	})

	t.Run("stage invalid gzip", func(t *testing.T) {
		archivePath := filepath.Join(t.TempDir(), "bad.tar.gz")
		if err := os.WriteFile(archivePath, []byte("bad"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := stageUpstreamArchive(archivePath, t.TempDir(), target); err == nil {
			t.Fatal("stageUpstreamArchive() unexpectedly accepted a non-gzip archive")
		}
	})

	t.Run("stage directory entries", func(t *testing.T) {
		archivePath := filepath.Join(t.TempDir(), "with-dir.tar.gz")
		writeCustomArchive(t, archivePath, []tarEntry{
			{name: "bin", typeflag: tar.TypeDir, mode: 0o755},
			{name: "bin/codex", contents: []byte("codex"), mode: 0o755},
			{name: "bin/codex-code-mode-host", contents: []byte("host"), mode: 0o755},
			{name: "codex-package.json", contents: []byte("{}"), mode: 0o644},
			{name: "codex-path/rg", contents: []byte("rg"), mode: 0o755},
			{name: "codex-resources/bwrap", contents: []byte("bwrap"), mode: 0o755},
			{name: "codex-resources/zsh/bin/zsh", contents: []byte("zsh"), mode: 0o755},
		})
		if err := stageUpstreamArchive(archivePath, t.TempDir(), target); err != nil {
			t.Fatalf("stageUpstreamArchive() with dir entry failed: %v", err)
		}
	})
}

func TestWriteReleaseEvidenceAndCopyHelpersErrorBranches(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	stageDir := buildFakePackage(t, t.TempDir(), target)
	if err := os.WriteFile(filepath.Join(stageDir, "surprise.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSBOM(stageDir, RepackOptions{
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}); err == nil {
		t.Fatal("writeSBOM() unexpectedly accepted an unexpected stage file")
	}
	if err := writeProvenance(stageDir, RepackOptions{
		RuntimeVersion: "bad",
		Target:         target,
	}, strings.Repeat("a", 64), 1); err == nil {
		t.Fatal("writeProvenance() unexpectedly accepted an invalid runtime version")
	}

	destParent := filepath.Join(t.TempDir(), "file-parent")
	if err := os.WriteFile(destParent, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(t.TempDir(), "source.txt")
	if err := os.WriteFile(source, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(filepath.Join(destParent, "child.txt"), source, 0o644); err == nil {
		t.Fatal("copyFile() unexpectedly accepted a file parent directory")
	}
	if err := os.Mkdir(filepath.Join(t.TempDir(), "dest-dir"), 0o755); err != nil {
		t.Fatal(err)
	}
	destDir := filepath.Join(t.TempDir(), "dir-target")
	if err := os.Mkdir(destDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(destDir, source, 0o644); err == nil {
		t.Fatal("copyFile() unexpectedly accepted a destination directory")
	}
}

func TestWriteReleaseEvidenceFieldSpecificFailures(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	stageDir := buildFakePackage(t, filepath.Join(base, "pkg"), target)
	licensePath := filepath.Join(base, "LICENSE")
	if err := os.WriteFile(licensePath, []byte("license"), 0o644); err != nil {
		t.Fatal(err)
	}
	noticePath := filepath.Join(base, "NOTICE")
	if err := os.WriteFile(noticePath, []byte("notice"), 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("notice", func(t *testing.T) {
		if err := writeReleaseEvidence(filepath.Join(base, "notice-stage"), RepackOptions{
			RuntimeVersion: DefaultRuntimeVersion,
			Target:         target,
			LicensePath:    licensePath,
			NoticePath:     filepath.Join(base, "missing-notice"),
		}, strings.Repeat("a", 64), 1); err == nil {
			t.Fatal("writeReleaseEvidence() unexpectedly accepted a missing notice")
		}
	})

	t.Run("sbom", func(t *testing.T) {
		if err := writeReleaseEvidence(filepath.Join(base, "sbom-stage"), RepackOptions{
			RuntimeVersion: DefaultRuntimeVersion,
			Target:         target,
			LicensePath:    licensePath,
			NoticePath:     noticePath,
			SBOMPath:       filepath.Join(base, "missing-sbom"),
		}, strings.Repeat("a", 64), 1); err == nil {
			t.Fatal("writeReleaseEvidence() unexpectedly accepted a missing sbom")
		}
	})

	t.Run("provenance", func(t *testing.T) {
		if err := writeReleaseEvidence(filepath.Join(base, "prov-stage"), RepackOptions{
			RuntimeVersion: DefaultRuntimeVersion,
			Target:         target,
			LicensePath:    licensePath,
			NoticePath:     noticePath,
			ProvenancePath: filepath.Join(base, "missing-provenance"),
		}, strings.Repeat("a", 64), 1); err == nil {
			t.Fatal("writeReleaseEvidence() unexpectedly accepted a missing provenance")
		}
	})

	t.Run("native sign", func(t *testing.T) {
		if err := writeReleaseEvidence(filepath.Join(base, "sign-stage"), RepackOptions{
			RuntimeVersion: DefaultRuntimeVersion,
			Target:         target,
			LicensePath:    licensePath,
			NoticePath:     noticePath,
			NativeSignPath: filepath.Join(base, "missing-sign"),
		}, strings.Repeat("a", 64), 1); err == nil {
			t.Fatal("writeReleaseEvidence() unexpectedly accepted a missing native-sign file")
		}
	})

	_ = stageDir
}

func TestWriteReleaseEvidenceGeneratedPathFailuresAndLicenseCopy(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	base := t.TempDir()
	noticePath := filepath.Join(base, "NOTICE")
	if err := os.WriteFile(noticePath, []byte("notice"), 0o644); err != nil {
		t.Fatal(err)
	}
	sbomPath := filepath.Join(base, "sbom.json")
	if err := os.WriteFile(sbomPath, []byte("{}"), 0o644); err != nil {
		t.Fatal(err)
	}

	stageDir := buildFakePackage(t, filepath.Join(base, "pkg"), target)
	if err := copyOrDefaultLicense(stageDir, ""); err != nil {
		t.Fatalf("copyOrDefaultLicense() should copy repo LICENSE: %v", err)
	}
	if _, err := os.Stat(filepath.Join(stageDir, licenseFileName)); err != nil {
		t.Fatalf("copyOrDefaultLicense() missing staged license: %v", err)
	}

	sbomFailureStage := buildFakePackage(t, filepath.Join(base, "sbom-fail"), target)
	if err := os.WriteFile(filepath.Join(sbomFailureStage, "surprise.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeReleaseEvidence(sbomFailureStage, RepackOptions{
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
		NoticePath:     noticePath,
	}, strings.Repeat("a", 64), 1); err == nil {
		t.Fatal("writeReleaseEvidence() unexpectedly accepted generated sbom from a bad stage")
	}

	provFailureStage := buildFakePackage(t, filepath.Join(base, "prov-fail"), target)
	if err := os.WriteFile(filepath.Join(provFailureStage, "surprise.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeReleaseEvidence(provFailureStage, RepackOptions{
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
		NoticePath:     noticePath,
		SBOMPath:       sbomPath,
	}, strings.Repeat("a", 64), 1); err == nil {
		t.Fatal("writeReleaseEvidence() unexpectedly accepted generated provenance from a bad stage")
	}

	if err := writeProvenance(provFailureStage, RepackOptions{
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}, strings.Repeat("a", 64), 1); err == nil {
		t.Fatal("writeProvenance() unexpectedly accepted an unexpected stage file")
	}
}

func TestRemoveAndMetadataHelpers(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	record := releaseLikeTargets(t)[0]
	root := filepath.Join(t.TempDir(), "root")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := writeInstallMetadata(root, record, DefaultRuntimeVersion, target); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(root, ".codex-sdk-runtime.json")); err != nil {
		t.Fatalf("writeInstallMetadata() missing file: %v", err)
	}
	cacheRoot := t.TempDir()
	installRoot := InstallRoot(cacheRoot, DefaultRuntimeVersion, target)
	if err := os.MkdirAll(installRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := Remove(cacheRoot, DefaultRuntimeVersion, target); err != nil {
		t.Fatalf("Remove() failed: %v", err)
	}
	if _, err := os.Stat(installRoot); !os.IsNotExist(err) {
		t.Fatalf("Remove() did not delete install root: %v", err)
	}
}

func TestSanitizeRequestErrorPassthrough(t *testing.T) {
	t.Parallel()

	err := errors.New("plain boom")
	if got := sanitizeRequestError(err); !errors.Is(got, err) {
		t.Fatalf("sanitizeRequestError() = %v", got)
	}
}

func TestWriteSBOMSkipsExistingSBOMFile(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	stageDir := buildFakePackage(t, t.TempDir(), target)
	if err := os.WriteFile(filepath.Join(stageDir, sbomFileName), []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeSBOM(stageDir, RepackOptions{
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         target,
	}); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(stageDir, sbomFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(body), "\"files\"") {
		t.Fatalf("writeSBOM() did not rewrite document: %s", body)
	}
}

func TestInstallReturnsExistingManagedRuntime(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()
	opts := InstallOptions{
		CacheRoot:      cacheRoot,
		RuntimeVersion: release.Manifest.RuntimeVersion,
		Target:         &target,
		ManifestPath:   release.ManifestPath,
		SignaturePath:  release.SignaturePath,
		ArchivePath:    release.ArchivePaths[target.Triple],
		TrustRoots:     release.TrustRoots,
	}
	first, err := Install(context.Background(), opts)
	if err != nil {
		t.Fatalf("first Install() failed: %v", err)
	}
	second, err := Install(context.Background(), opts)
	if err != nil {
		t.Fatalf("second Install() failed: %v", err)
	}
	if second.Runtime.BinaryPath != first.Runtime.BinaryPath || second.Runtime.Source != "managed" {
		t.Fatalf("second Install() returned %#v", second.Runtime)
	}
}

func TestInstallFailsWhenLockContextIsCancelled(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()
	lockPath := InstallRoot(cacheRoot, release.Manifest.RuntimeVersion, target) + ".lock"
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(lockPath, []byte("held"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = Install(ctx, InstallOptions{
		CacheRoot:      cacheRoot,
		RuntimeVersion: release.Manifest.RuntimeVersion,
		Target:         &target,
		ManifestPath:   release.ManifestPath,
		SignaturePath:  release.SignaturePath,
		ArchivePath:    release.ArchivePaths[target.Triple],
		TrustRoots:     release.TrustRoots,
		StaleLockAfter: 0,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Install() canceled lock wait = %v", err)
	}
}

func TestInstallFailsOnOfflineInputErrors(t *testing.T) {
	t.Parallel()

	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := Install(context.Background(), InstallOptions{
		CacheRoot:      t.TempDir(),
		RuntimeVersion: DefaultRuntimeVersion,
		Target:         &target,
		ManifestPath:   filepath.Join(t.TempDir(), "missing-manifest.json"),
		SignaturePath:  filepath.Join(t.TempDir(), "missing-signature.json"),
		ArchivePath:    filepath.Join(t.TempDir(), "missing.tar.gz"),
	}); err == nil {
		t.Fatal("Install() unexpectedly accepted missing offline inputs")
	}
}

func TestResolveInstallInputsOnlineValidationFailures(t *testing.T) {
	t.Parallel()

	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	release := buildFakeRelease(t)
	manifestBytes, err := os.ReadFile(release.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}

	t.Run("invalid manifest json", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/manifest.json":
				_, _ = w.Write([]byte("{"))
			case "/manifest.json.sig":
				_, _ = w.Write([]byte(`{"keyId":"x","algorithm":"ed25519","signature":"abc"}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		if _, _, _, _, _, err := resolveInstallInputs(context.Background(), InstallOptions{
			ManifestURL:     server.URL + "/manifest.json",
			SignatureURL:    server.URL + "/manifest.json.sig",
			BaseURL:         server.URL,
			HTTPClient:      server.Client(),
			MaxArchiveBytes: 1 << 20,
		}, target); err == nil {
			t.Fatal("resolveInstallInputs() unexpectedly accepted invalid manifest json")
		}
	})

	t.Run("invalid manifest validation", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/manifest.json":
				_, _ = w.Write([]byte(`{"schemaVersion":1,"sdkVersion":"v0.2.0","runtimeVersion":"0.144.4","upstreamRepo":"openai/codex","upstreamTag":"rust-v0.144.4","targets":[]}`))
			case "/manifest.json.sig":
				_, _ = w.Write([]byte(`{"keyId":"x","algorithm":"ed25519","signature":"abc"}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		if _, _, _, _, _, err := resolveInstallInputs(context.Background(), InstallOptions{
			ManifestURL:     server.URL + "/manifest.json",
			SignatureURL:    server.URL + "/manifest.json.sig",
			BaseURL:         server.URL,
			HTTPClient:      server.Client(),
			MaxArchiveBytes: 1 << 20,
		}, target); err == nil {
			t.Fatal("resolveInstallInputs() unexpectedly accepted an invalid manifest")
		}
	})

	t.Run("invalid signature json", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/manifest.json":
				_, _ = w.Write(manifestBytes)
			case "/manifest.json.sig":
				_, _ = w.Write([]byte("{"))
			default:
				http.NotFound(w, r)
			}
		}))
		defer server.Close()
		if _, _, _, _, _, err := resolveInstallInputs(context.Background(), InstallOptions{
			ManifestURL:     server.URL + "/manifest.json",
			SignatureURL:    server.URL + "/manifest.json.sig",
			BaseURL:         server.URL,
			HTTPClient:      server.Client(),
			MaxArchiveBytes: 1 << 20,
		}, target); err == nil {
			t.Fatal("resolveInstallInputs() unexpectedly accepted invalid signature json")
		}
	})

	t.Run("host policy mismatch", func(t *testing.T) {
		manifestServer := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			switch r.URL.Path {
			case "/manifest.json":
				_, _ = w.Write(manifestBytes)
			case "/manifest.json.sig":
				_, _ = w.Write([]byte(`{"keyId":"x","algorithm":"ed25519","signature":"abc"}`))
			default:
				http.NotFound(w, r)
			}
		}))
		defer manifestServer.Close()
		archiveServer := httptest.NewTLSServer(http.NotFoundHandler())
		defer archiveServer.Close()
		if _, _, _, _, _, err := resolveInstallInputs(context.Background(), InstallOptions{
			ManifestURL:     manifestServer.URL + "/manifest.json",
			SignatureURL:    manifestServer.URL + "/manifest.json.sig",
			BaseURL:         archiveServer.URL,
			HTTPClient:      manifestServer.Client(),
			MaxArchiveBytes: 1 << 20,
		}, target); err == nil || !strings.Contains(err.Error(), "share one host") {
			t.Fatalf("resolveInstallInputs() host policy error = %v", err)
		}
	})
}

func TestRepackRuntimePropagatesInternalFailures(t *testing.T) {
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

	t.Run("download failure", func(t *testing.T) {
		server := httptest.NewTLSServer(http.NotFoundHandler())
		defer server.Close()
		if _, err := RepackRuntime(context.Background(), RepackOptions{
			Target:           target,
			DownloadUpstream: true,
			UpstreamBaseURL:  server.URL,
			HTTPClient:       server.Client(),
			OutputArchive:    filepath.Join(root, "download-fail.tar.gz"),
		}); err == nil {
			t.Fatal("RepackRuntime() unexpectedly accepted a failing download")
		}
	})

	t.Run("stage failure", func(t *testing.T) {
		stagePath := filepath.Join(root, "not-a-dir")
		if err := os.WriteFile(stagePath, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := RepackRuntime(context.Background(), RepackOptions{
			Target:          target,
			UpstreamArchive: upstreamArchive,
			StageDir:        stagePath,
			OutputArchive:   filepath.Join(root, "stage-fail.tar.gz"),
		}); err == nil {
			t.Fatal("RepackRuntime() unexpectedly accepted an invalid stage path")
		}
	})

	t.Run("release evidence failure", func(t *testing.T) {
		if _, err := RepackRuntime(context.Background(), RepackOptions{
			Target:          target,
			UpstreamArchive: upstreamArchive,
			StageDir:        filepath.Join(root, "evidence-stage"),
			OutputArchive:   filepath.Join(root, "evidence-fail.tar.gz"),
			NoticePath:      filepath.Join(root, "missing-notice"),
		}); err == nil {
			t.Fatal("RepackRuntime() unexpectedly accepted missing release evidence")
		}
	})

	t.Run("package archive failure", func(t *testing.T) {
		server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.URL.Path != "/"+target.UpstreamAsset {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write(archiveBytes)
		}))
		defer server.Close()
		outputParent := filepath.Join(root, "archive-parent-file")
		if err := os.WriteFile(outputParent, []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := RepackRuntime(context.Background(), RepackOptions{
			Target:           target,
			DownloadUpstream: true,
			UpstreamBaseURL:  server.URL,
			HTTPClient:       server.Client(),
			OutputArchive:    filepath.Join(outputParent, "fail.tar.gz"),
		}); err == nil {
			t.Fatal("RepackRuntime() unexpectedly accepted an invalid output parent")
		}
	})
}

func TestResolveInstallInputsOfflineRejectsInvalidSignatureEnvelope(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	signaturePath := filepath.Join(t.TempDir(), "manifest.sig")
	if err := os.WriteFile(signaturePath, []byte(`{"keyId":"","algorithm":"ed25519","signature":"abc"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, _, _, _, _, err := resolveInstallInputs(context.Background(), InstallOptions{
		ManifestPath:  release.ManifestPath,
		SignaturePath: signaturePath,
		ArchivePath:   release.ArchivePaths[target.Triple],
	}, target); err == nil {
		t.Fatal("resolveInstallInputs() unexpectedly accepted an invalid signature envelope")
	}
}

func TestResolveRuntimeAllowPATHMissingReturnsInstallHint(t *testing.T) {
	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", t.TempDir())
	if _, err := ResolveRuntime(ResolveOptions{
		RuntimeVersion: DefaultRuntimeVersion,
		AllowPATH:      true,
		Target:         &target,
	}); !errors.Is(err, ErrRuntimeNotInstalled) {
		t.Fatalf("ResolveRuntime() PATH miss = %v", err)
	}
}

func TestLookPathUsesOnlySuppliedEnvironmentAndSkipsInvalidCandidates(t *testing.T) {
	t.Setenv("OS", "")
	first := t.TempDir()
	second := t.TempDir()
	if err := os.Mkdir(filepath.Join(first, "codex"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(first, "not-executable"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(second, "codex")
	if err := os.WriteFile(want, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"PATH": string(os.PathListSeparator) + first + string(os.PathListSeparator) + second}
	got, err := lookPath("codex", env, false)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("lookPath() = %q, want %q", got, want)
	}
	if runtime.GOOS != "windows" {
		if _, err := lookPath("not-executable", env, false); !errors.Is(err, exec.ErrNotFound) {
			t.Fatalf("lookPath(non-executable) = %v", err)
		}
	}
}

func TestLookPathWindowsExtensionsFromSuppliedEnvironment(t *testing.T) {
	t.Setenv("OS", "Windows_NT")
	dir := t.TempDir()
	want := filepath.Join(dir, "codex.EXE")
	if err := os.WriteFile(want, []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := lookPath("codex", map[string]string{"Path": dir, "PATHEXT": ";.EXE"}, false)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.EqualFold(got, want) {
		t.Fatalf("lookPath() = %q, want %q", got, want)
	}
}

func TestPathEnvKeyFallsBackToPATHWhenMissing(t *testing.T) {
	t.Setenv("OS", "Windows_NT")
	if key := PathEnvKey(map[string]string{"HOME": "x"}); key != "PATH" {
		t.Fatalf("PathEnvKey() fallback = %q", key)
	}
}

func TestPackagerErrorBranches(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := collectPackageFiles(filepath.Join(t.TempDir(), "missing-root")); err == nil {
		t.Fatal("collectPackageFiles() unexpectedly accepted a missing root")
	}

	root := t.TempDir()
	packageDir := buildFakePackage(t, root, target)
	files, err := collectPackageFiles(packageDir)
	if err != nil {
		t.Fatal(err)
	}
	parentFile := filepath.Join(root, "not-a-dir")
	if err := os.WriteFile(parentFile, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := writeArchive(filepath.Join(parentFile, "archive.tar.gz"), packageDir, files); err == nil {
		t.Fatal("writeArchive() unexpectedly accepted a file parent")
	}
	if err := os.Remove(filepath.Join(packageDir, filepath.FromSlash(files[0].Path))); err != nil {
		t.Fatal(err)
	}
	if err := writeArchive(filepath.Join(root, "archive.tar.gz"), packageDir, files); err == nil {
		t.Fatal("writeArchive() unexpectedly accepted a missing source file")
	}
}

func TestExtractArchiveContentValidationBranches(t *testing.T) {
	t.Parallel()

	makeRecord := func(archivePath string, files []ManifestFile) ManifestTarget {
		sha, size := mustFileSHA(t, archivePath)
		return ManifestTarget{
			ArchiveSHA256: sha,
			ArchiveSize:   size,
			Files:         files,
		}
	}

	t.Run("missing archive file", func(t *testing.T) {
		err := extractArchive(filepath.Join(t.TempDir(), "missing.tar.gz"), t.TempDir(), ManifestTarget{}, DefaultMaxArchiveBytes)
		if err == nil {
			t.Fatal("extractArchive() unexpectedly accepted a missing archive")
		}
	})

	t.Run("invalid gzip stream", func(t *testing.T) {
		archivePath := filepath.Join(t.TempDir(), "bad.tar.gz")
		if err := os.WriteFile(archivePath, []byte("not-gzip"), 0o644); err != nil {
			t.Fatal(err)
		}
		record := makeRecord(archivePath, nil)
		if err := extractArchive(archivePath, t.TempDir(), record, DefaultMaxArchiveBytes); err == nil {
			t.Fatal("extractArchive() unexpectedly accepted a non-gzip stream")
		}
	})

	t.Run("unexpected file", func(t *testing.T) {
		archivePath := filepath.Join(t.TempDir(), "unexpected.tar.gz")
		writeCustomArchive(t, archivePath, []tarEntry{{name: "bin/extra", contents: []byte("x"), mode: 0o755}})
		record := makeRecord(archivePath, []ManifestFile{{Path: "bin/codex", SHA256: sha256Hex([]byte("x")), Size: 1, Mode: 0o755, Role: "executable"}})
		if err := extractArchive(archivePath, t.TempDir(), record, DefaultMaxArchiveBytes); err == nil || !strings.Contains(err.Error(), "unexpected archive file") {
			t.Fatalf("extractArchive() unexpected file = %v", err)
		}
	})

	t.Run("invalid path", func(t *testing.T) {
		archivePath := filepath.Join(t.TempDir(), "invalid-path.tar.gz")
		writeCustomArchive(t, archivePath, []tarEntry{{name: "../evil", contents: []byte("x"), mode: 0o644}})
		record := makeRecord(archivePath, []ManifestFile{{Path: "bin/codex", SHA256: sha256Hex([]byte("x")), Size: 1, Mode: 0o755, Role: "executable"}})
		if err := extractArchive(archivePath, t.TempDir(), record, DefaultMaxArchiveBytes); err == nil || !strings.Contains(err.Error(), "invalid archive path") {
			t.Fatalf("extractArchive() invalid path = %v", err)
		}
	})

	t.Run("file size mismatch", func(t *testing.T) {
		archivePath := filepath.Join(t.TempDir(), "size-mismatch.tar.gz")
		writeCustomArchive(t, archivePath, []tarEntry{{name: "bin/codex", contents: []byte("xy"), mode: 0o755}})
		record := makeRecord(archivePath, []ManifestFile{{Path: "bin/codex", SHA256: sha256Hex([]byte("xy")), Size: 1, Mode: 0o755, Role: "executable"}})
		if err := extractArchive(archivePath, t.TempDir(), record, DefaultMaxArchiveBytes); err == nil || !strings.Contains(err.Error(), "archive file size mismatch") {
			t.Fatalf("extractArchive() size mismatch = %v", err)
		}
	})

	t.Run("file sha mismatch", func(t *testing.T) {
		archivePath := filepath.Join(t.TempDir(), "sha-mismatch.tar.gz")
		writeCustomArchive(t, archivePath, []tarEntry{{name: "bin/codex", contents: []byte("xy"), mode: 0o755}})
		record := makeRecord(archivePath, []ManifestFile{{Path: "bin/codex", SHA256: strings.Repeat("0", 64), Size: 2, Mode: 0o755, Role: "executable"}})
		if err := extractArchive(archivePath, t.TempDir(), record, DefaultMaxArchiveBytes); err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
			t.Fatalf("extractArchive() sha mismatch = %v", err)
		}
	})
}

func releaseLikeTargets(t *testing.T) []ManifestTarget {
	t.Helper()

	records := make([]ManifestTarget, 0, len(SupportedTargets()))
	for _, target := range SupportedTargets() {
		records = append(records, ManifestTarget{
			GOOS:                  target.GOOS,
			GOARCH:                target.GOARCH,
			Triple:                target.Triple,
			Executable:            target.Executable,
			UpstreamAsset:         target.UpstreamAsset,
			UpstreamArchiveSHA256: strings.Repeat("a", sha256.Size*2),
			UpstreamArchiveSize:   1,
			ArchiveName:           target.Triple + ".tar.gz",
			ArchiveSHA256:         strings.Repeat("b", sha256.Size*2),
			ArchiveSize:           1,
			Files: []ManifestFile{{
				Path:   "bin/" + target.Executable,
				SHA256: hex.EncodeToString(sha256.New().Sum(nil)),
				Size:   1,
				Mode:   0o755,
				Role:   "executable",
			}},
		})
	}
	return records
}
