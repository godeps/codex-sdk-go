package runtimebin

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestReleaseTagForVersion(t *testing.T) {
	t.Parallel()

	cases := map[string]string{
		"0.144.4":               "rust-v0.144.4",
		"rust-v0.144.4-alpha.1": "rust-v0.144.4-alpha.1",
		"v0.144.4-beta.2":       "rust-v0.144.4-beta.2",
		"0.144.4rc3":            "rust-v0.144.4-rc.3",
		"0.144.4a1.post2":       "rust-v0.144.4-alpha.1.2",
	}
	for input, want := range cases {
		got, err := ReleaseTagForVersion(input)
		if err != nil {
			t.Fatalf("ReleaseTagForVersion(%q) error: %v", input, err)
		}
		if got != want {
			t.Fatalf("ReleaseTagForVersion(%q) = %q, want %q", input, got, want)
		}
	}

	if _, err := ReleaseTagForVersion("oops"); err == nil {
		t.Fatal("expected invalid version to fail")
	}
}

func TestPackageArchiveIsDeterministic(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	packageDir := buildFakePackage(t, root, target)
	firstArchive := filepath.Join(root, "first.tar.gz")
	secondArchive := filepath.Join(root, "second.tar.gz")

	first, err := PackageArchive(PackageOptions{PackageDir: packageDir, OutputArchive: firstArchive, RuntimeVersion: DefaultRuntimeVersion, Target: target})
	if err != nil {
		t.Fatal(err)
	}
	second, err := PackageArchive(PackageOptions{PackageDir: packageDir, OutputArchive: secondArchive, RuntimeVersion: DefaultRuntimeVersion, Target: target})
	if err != nil {
		t.Fatal(err)
	}

	if first.ArchiveSHA256 != second.ArchiveSHA256 {
		t.Fatalf("deterministic archive mismatch: %s vs %s", first.ArchiveSHA256, second.ArchiveSHA256)
	}
	if first.ArchiveSize != second.ArchiveSize {
		t.Fatalf("deterministic archive size mismatch: %d vs %d", first.ArchiveSize, second.ArchiveSize)
	}
}

func TestManifestSigningAndVerification(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	if err := release.Manifest.Validate(); err != nil {
		t.Fatal(err)
	}
	if err := VerifyManifestSignature(release.ManifestBytes, release.SignatureEnvelope, release.TrustRoots); err != nil {
		t.Fatalf("verify manifest signature: %v", err)
	}
	tampered := append([]byte(nil), release.ManifestBytes...)
	tampered[len(tampered)-2] ^= 1
	if err := VerifyManifestSignature(tampered, release.SignatureEnvelope, release.TrustRoots); err == nil {
		t.Fatal("expected tampered manifest verification to fail")
	}
	if err := VerifyManifestSignature(release.ManifestBytes, SignatureEnvelope{
		KeyID:     "unknown",
		Algorithm: SignatureAlgorithm,
		Signature: release.SignatureEnvelope.Signature,
	}, release.TrustRoots); err == nil {
		t.Fatal("expected unknown key to fail")
	}
}

func TestTrustRootRotationOverlap(t *testing.T) {
	t.Parallel()

	manifest := []byte("{\"schemaVersion\":1}\n")
	oldSeed := bytes.Repeat([]byte{1}, ed25519.SeedSize)
	newSeed := bytes.Repeat([]byte{2}, ed25519.SeedSize)
	oldKey := ed25519.NewKeyFromSeed(oldSeed)
	newKey := ed25519.NewKeyFromSeed(newSeed)

	oldEnvelope, err := SignManifest(manifest, "old", oldKey)
	if err != nil {
		t.Fatal(err)
	}
	newEnvelope, err := SignManifest(manifest, "new", newKey)
	if err != nil {
		t.Fatal(err)
	}
	overlap := []TrustRoot{
		{KeyID: "old", PublicKey: oldKey.Public().(ed25519.PublicKey)},
		{KeyID: "new", PublicKey: newKey.Public().(ed25519.PublicKey)},
	}
	if err := VerifyManifestSignature(manifest, oldEnvelope, overlap); err != nil {
		t.Fatalf("old key overlap verify: %v", err)
	}
	if err := VerifyManifestSignature(manifest, newEnvelope, overlap); err != nil {
		t.Fatalf("new key overlap verify: %v", err)
	}
	newOnly := []TrustRoot{{KeyID: "new", PublicKey: newKey.Public().(ed25519.PublicKey)}}
	if err := VerifyManifestSignature(manifest, oldEnvelope, newOnly); err == nil {
		t.Fatal("expected retired key to fail after rotation window")
	}
	if err := VerifyManifestSignature(manifest, newEnvelope, newOnly); err != nil {
		t.Fatalf("new key verify after retirement: %v", err)
	}
}

func TestInstallOfflineAndResolve(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()

	result, err := Install(context.Background(), InstallOptions{
		CacheRoot:      cacheRoot,
		RuntimeVersion: release.Manifest.RuntimeVersion,
		Target:         &target,
		ManifestPath:   release.ManifestPath,
		SignaturePath:  release.SignaturePath,
		ArchivePath:    release.ArchivePaths[target.Triple],
		TrustRoots:     release.TrustRoots,
	})
	if err != nil {
		t.Fatalf("install runtime: %v", err)
	}

	if _, err := os.Stat(result.Runtime.BinaryPath); err != nil {
		t.Fatalf("installed runtime missing: %v", err)
	}

	resolved, err := ResolveRuntime(ResolveOptions{
		CacheRoot:      cacheRoot,
		RuntimeVersion: release.Manifest.RuntimeVersion,
		Target:         &target,
	})
	if err != nil {
		t.Fatalf("resolve managed runtime: %v", err)
	}
	if resolved.Source != "managed" {
		t.Fatalf("resolve source = %q, want managed", resolved.Source)
	}

	explicit, err := ResolveRuntime(ResolveOptions{
		ExplicitBinary: result.Runtime.BinaryPath,
		RuntimeVersion: release.Manifest.RuntimeVersion,
		Target:         &target,
	})
	if err != nil {
		t.Fatalf("resolve explicit runtime: %v", err)
	}
	if explicit.Source != "explicit" {
		t.Fatalf("explicit source = %q", explicit.Source)
	}

	envResolved, err := ResolveRuntime(ResolveOptions{
		Env:            map[string]string{"CODEX_RUNTIME_PATH": result.Runtime.BinaryPath},
		RuntimeVersion: release.Manifest.RuntimeVersion,
		Target:         &target,
	})
	if err != nil {
		t.Fatalf("resolve env runtime: %v", err)
	}
	if envResolved.Source != "env" {
		t.Fatalf("env source = %q", envResolved.Source)
	}
}

func TestConcurrentInstallConverges(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()
	var wg sync.WaitGroup
	errCh := make(chan error, 8)
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := Install(context.Background(), InstallOptions{
				CacheRoot:      cacheRoot,
				RuntimeVersion: release.Manifest.RuntimeVersion,
				Target:         &target,
				ManifestPath:   release.ManifestPath,
				SignaturePath:  release.SignaturePath,
				ArchivePath:    release.ArchivePaths[target.Triple],
				TrustRoots:     release.TrustRoots,
			})
			errCh <- err
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent install error: %v", err)
		}
	}
	record, err := release.Manifest.Target(target.Triple)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := VerifyInstalled(cacheRoot, release.Manifest.RuntimeVersion, target, record); err != nil {
		t.Fatalf("verify installed runtime: %v", err)
	}
}

func TestInstallRejectsTraversalAndSymlinkArchives(t *testing.T) {
	t.Parallel()

	release := buildFakeRelease(t)
	target, err := CurrentTarget()
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()

	for _, tc := range []struct {
		name    string
		archive string
		modify  func(*ManifestTarget, string)
	}{
		{
			name:    "traversal",
			archive: filepath.Join(t.TempDir(), "traversal.tar.gz"),
			modify: func(record *ManifestTarget, archive string) {
				writeCustomArchive(t, archive, []tarEntry{
					{name: "../evil", contents: []byte("evil"), mode: 0o644},
				})
				record.ArchiveSHA256, record.ArchiveSize = mustFileSHA(t, archive)
				record.Files = []ManifestFile{{Path: "codex-package.json", SHA256: sha256Hex([]byte("{}")), Size: 2, Mode: 0o644, Role: "metadata"}}
			},
		},
		{
			name:    "symlink",
			archive: filepath.Join(t.TempDir(), "symlink.tar.gz"),
			modify: func(record *ManifestTarget, archive string) {
				writeCustomArchive(t, archive, []tarEntry{
					{name: "bin/codex", contents: []byte("x"), mode: 0o755, typeflag: tar.TypeSymlink, linkname: "/tmp/x"},
				})
				record.ArchiveSHA256, record.ArchiveSize = mustFileSHA(t, archive)
				record.Files = []ManifestFile{{Path: "bin/codex", SHA256: sha256Hex([]byte("x")), Size: 1, Mode: 0o755, Role: "executable"}}
			},
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			badManifest := release.Manifest
			for i := range badManifest.Targets {
				if badManifest.Targets[i].Triple == target.Triple {
					tc.modify(&badManifest.Targets[i], tc.archive)
				}
			}
			if err := writeSignedManifestSet(t, release.Root, badManifest, release.PrivateKey, release.TrustRoots[0].KeyID); err != nil {
				t.Fatal(err)
			}
			_, err := Install(context.Background(), InstallOptions{
				CacheRoot:      cacheRoot,
				RuntimeVersion: badManifest.RuntimeVersion,
				Target:         &target,
				ManifestPath:   filepath.Join(release.Root, "runtime", "manifest.json"),
				SignaturePath:  filepath.Join(release.Root, "runtime", "manifest.json.sig"),
				ArchivePath:    tc.archive,
				TrustRoots:     release.TrustRoots,
			})
			if err == nil {
				t.Fatal("expected malicious archive install to fail")
			}
		})
	}
}

func TestPrependPathDirsPreservesWindowsPathKey(t *testing.T) {
	t.Setenv("OS", "Windows_NT")
	env := map[string]string{
		"PATH": "/usr/bin",
		"Path": "C:\\Windows",
	}
	PrependPathDirs(env, []string{"C:\\Codex\\codex-path"})
	if _, ok := env["PATH"]; ok {
		t.Fatal("expected PATH entry to be collapsed into Path")
	}
	if env["Path"] != "C:\\Codex\\codex-path"+string(os.PathListSeparator)+"C:\\Windows" {
		t.Fatalf("unexpected Path value %q", env["Path"])
	}
}

func TestRemoveRejectsVersionTraversal(t *testing.T) {
	t.Parallel()

	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	cacheRoot := t.TempDir()
	if err := Remove(cacheRoot, "../../escape", target); err == nil {
		t.Fatal("expected traversal version to fail")
	}
	if err := Remove(cacheRoot, "", target); err == nil {
		t.Fatal("expected empty version to fail")
	}
}

func TestDownloadRejectsRedirectDowngradeAndCrossHost(t *testing.T) {
	t.Parallel()

	httpServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer httpServer.Close()

	redirectToHTTP := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, httpServer.URL+"/asset?token=secret", http.StatusFound)
	}))
	defer redirectToHTTP.Close()

	client := withSecureRedirectPolicy(redirectToHTTP.Client())
	_, err := downloadURL(context.Background(), client, redirectToHTTP.URL+"/asset?token=secret", 1024)
	if err == nil || !strings.Contains(err.Error(), "non-https") {
		t.Fatalf("expected non-https redirect failure, got %v", err)
	}
	if strings.Contains(err.Error(), "secret") {
		t.Fatalf("redirect error leaked query secret: %v", err)
	}

	targetTLS := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}))
	defer targetTLS.Close()

	redirectToOtherHost := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, targetTLS.URL+"/asset", http.StatusFound)
	}))
	defer redirectToOtherHost.Close()

	client = withSecureRedirectPolicy(targetTLS.Client())
	_, err = downloadURL(context.Background(), client, redirectToOtherHost.URL+"/asset", 1024)
	if err == nil || !strings.Contains(err.Error(), "unexpected host") {
		t.Fatalf("expected cross-host redirect failure, got %v", err)
	}
}

func TestRepackRuntimeAddsEvidenceFiles(t *testing.T) {
	t.Parallel()

	root := t.TempDir()
	target, err := LookupTarget("linux", "amd64")
	if err != nil {
		t.Fatal(err)
	}
	upstreamDir := buildFakePackage(t, root, target)
	upstreamArchive := filepath.Join(root, "upstream.tar.gz")
	if _, err := PackageArchive(PackageOptions{PackageDir: upstreamDir, OutputArchive: upstreamArchive, RuntimeVersion: DefaultRuntimeVersion, Target: target}); err != nil {
		t.Fatal(err)
	}
	stageDir := filepath.Join(root, "stage")
	repackedArchive := filepath.Join(root, "repacked.tar.gz")
	record, err := RepackRuntime(context.Background(), RepackOptions{
		SDKVersion:      DefaultSDKVersion,
		RuntimeVersion:  DefaultRuntimeVersion,
		Target:          target,
		UpstreamArchive: upstreamArchive,
		StageDir:        stageDir,
		OutputArchive:   repackedArchive,
	})
	if err != nil {
		t.Fatal(err)
	}
	if record.UpstreamArchiveSHA256 == "" || record.UpstreamArchiveSize == 0 {
		t.Fatalf("expected upstream archive metadata, got %#v", record)
	}
	for _, path := range []string{
		licenseFileName,
		noticeFileName,
		sbomFileName,
		provenanceFileName,
		nativeSignaturesFileName,
		"codex-resources/zsh/bin/zsh",
	} {
		if _, err := os.Stat(filepath.Join(stageDir, path)); err != nil {
			t.Fatalf("expected staged file %s: %v", path, err)
		}
	}
}

func TestSanitizeURLRemovesSecrets(t *testing.T) {
	t.Parallel()
	got := sanitizeURL("https://user:pass@example.com/path?token=secret#frag")
	if strings.Contains(got, "secret") || strings.Contains(got, "pass") {
		t.Fatalf("sanitizeURL leaked secret: %s", got)
	}
}

type fakeRelease struct {
	Root              string
	Manifest          Manifest
	ManifestBytes     []byte
	ManifestPath      string
	SignaturePath     string
	ArchivePaths      map[string]string
	SignatureEnvelope SignatureEnvelope
	PrivateKey        ed25519.PrivateKey
	TrustRoots        []TrustRoot
}

func buildFakeRelease(t *testing.T) fakeRelease {
	t.Helper()

	root := t.TempDir()
	records := make([]ManifestTarget, 0, len(SupportedTargets()))
	archivePaths := map[string]string{}
	for _, target := range SupportedTargets() {
		packageDir := buildFakePackage(t, root, target)
		archivePath := filepath.Join(root, target.Triple+".tar.gz")
		record, err := PackageArchive(PackageOptions{
			PackageDir:     packageDir,
			OutputArchive:  archivePath,
			RuntimeVersion: DefaultRuntimeVersion,
			Target:         target,
		})
		if err != nil {
			t.Fatal(err)
		}
		archivePaths[target.Triple] = archivePath
		records = append(records, record)
	}
	manifest, err := BuildManifest(DefaultSDKVersion, DefaultRuntimeVersion, DefaultUpstreamRepo, records)
	if err != nil {
		t.Fatal(err)
	}
	seed := bytes.Repeat([]byte{7}, ed25519.SeedSize)
	privateKey := ed25519.NewKeyFromSeed(seed)
	keyID := "runtime-manifest-test"
	trustRoots := []TrustRoot{{KeyID: keyID, PublicKey: privateKey.Public().(ed25519.PublicKey)}}
	if err := writeSignedManifestSet(t, root, manifest, privateKey, keyID); err != nil {
		t.Fatal(err)
	}
	manifestBytes, err := os.ReadFile(filepath.Join(root, "runtime", "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	signatureBytes, err := os.ReadFile(filepath.Join(root, "runtime", "manifest.json.sig"))
	if err != nil {
		t.Fatal(err)
	}
	var envelope SignatureEnvelope
	if err := json.Unmarshal(signatureBytes, &envelope); err != nil {
		t.Fatal(err)
	}
	return fakeRelease{
		Root:              root,
		Manifest:          manifest,
		ManifestBytes:     manifestBytes,
		ManifestPath:      filepath.Join(root, "runtime", "manifest.json"),
		SignaturePath:     filepath.Join(root, "runtime", "manifest.json.sig"),
		ArchivePaths:      archivePaths,
		SignatureEnvelope: envelope,
		PrivateKey:        privateKey,
		TrustRoots:        trustRoots,
	}
}

func buildFakePackage(t *testing.T, root string, target TargetSpec) string {
	t.Helper()

	packageDir := filepath.Join(root, target.Triple)
	mustMkdirAll(t, filepath.Join(packageDir, "bin"))
	mustMkdirAll(t, filepath.Join(packageDir, "codex-path"))
	mustMkdirAll(t, filepath.Join(packageDir, "codex-resources"))
	mustWriteFile(t, filepath.Join(packageDir, "codex-package.json"), []byte("{\"variant\":\"codex\"}\n"), 0o644)
	mustWriteFile(t, filepath.Join(packageDir, "bin", target.Executable), []byte("#!/bin/sh\necho codex "+target.Triple+"\n"), 0o755)
	hostName := "codex-code-mode-host"
	rgName := "rg"
	if target.GOOS == "windows" {
		hostName += ".exe"
		rgName += ".exe"
	}
	mustWriteFile(t, filepath.Join(packageDir, "bin", hostName), []byte("host "+target.Triple+"\n"), 0o755)
	mustWriteFile(t, filepath.Join(packageDir, "codex-path", rgName), []byte("rg "+target.Triple+"\n"), 0o755)
	if target.GOOS == "linux" {
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "bwrap"), []byte("bwrap\n"), 0o755)
		mustMkdirAll(t, filepath.Join(packageDir, "codex-resources", "zsh", "bin"))
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "zsh", "bin", "zsh"), []byte("zsh\n"), 0o755)
	}
	if target.GOOS == "darwin" {
		mustMkdirAll(t, filepath.Join(packageDir, "codex-resources", "zsh", "bin"))
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "zsh", "bin", "zsh"), []byte("zsh\n"), 0o755)
	}
	if target.GOOS == "windows" {
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "codex-command-runner.exe"), []byte("runner\n"), 0o755)
		mustWriteFile(t, filepath.Join(packageDir, "codex-resources", "codex-windows-sandbox-setup.exe"), []byte("sandbox\n"), 0o755)
	}
	if target.GOOS != "windows" {
		mustWriteFile(t, filepath.Join(packageDir, "bin", target.Executable), []byte("#!/bin/sh\nexit 0\n"), 0o755)
	}
	return packageDir
}

func writeSignedManifestSet(t *testing.T, root string, manifest Manifest, privateKey ed25519.PrivateKey, keyID string) error {
	t.Helper()

	runtimeDir := filepath.Join(root, "runtime")
	mustMkdirAll(t, runtimeDir)
	manifestBytes, err := manifest.CanonicalBytes()
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(runtimeDir, "manifest.json"), append(manifestBytes, '\n'), 0o644); err != nil {
		return err
	}
	envelope, err := SignManifest(append(manifestBytes, '\n'), keyID, privateKey)
	if err != nil {
		return err
	}
	signatureBytes, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(runtimeDir, "manifest.json.sig"), append(signatureBytes, '\n'), 0o644)
}

type tarEntry struct {
	name     string
	contents []byte
	mode     int64
	typeflag byte
	linkname string
}

func writeCustomArchive(t *testing.T, path string, entries []tarEntry) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()
	gz := gzip.NewWriter(file)
	defer gz.Close()
	tw := tar.NewWriter(gz)
	defer tw.Close()
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{
			Name:     entry.name,
			Mode:     entry.mode,
			Size:     int64(len(entry.contents)),
			Typeflag: typeflag,
			Linkname: entry.linkname,
		}
		if err := tw.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if typeflag == tar.TypeReg {
			if _, err := tw.Write(entry.contents); err != nil {
				t.Fatal(err)
			}
		}
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

func mustFileSHA(t *testing.T, path string) (string, int64) {
	t.Helper()
	sha, size, err := fileSHA256AndSize(path)
	if err != nil {
		t.Fatal(err)
	}
	return sha, size
}

func sha256Hex(bytes []byte) string {
	sum := sha256.Sum256(bytes)
	return hex.EncodeToString(sum[:])
}
