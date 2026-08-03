package runtimebin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"
)

const (
	DefaultMaxArchiveBytes = 256 << 20
	DefaultHTTPTimeout     = 5 * time.Minute
	lockPollInterval       = 50 * time.Millisecond
	defaultStaleLockAfter  = 30 * time.Minute
)

type InstallOptions struct {
	CacheRoot       string
	RuntimeVersion  string
	Target          *TargetSpec
	ManifestPath    string
	SignaturePath   string
	ArchivePath     string
	ManifestURL     string
	SignatureURL    string
	BaseURL         string
	HTTPClient      *http.Client
	TrustRoots      []TrustRoot
	MaxArchiveBytes int64
	StaleLockAfter  time.Duration
	HTTPTimeout     time.Duration
}

type InstallResult struct {
	Runtime ResolvedRuntime
	Target  ManifestTarget
}

type installMetadata struct {
	RuntimeVersion string `json:"runtimeVersion"`
	Triple         string `json:"triple"`
	ArchiveName    string `json:"archiveName"`
	ArchiveSHA256  string `json:"archiveSha256"`
	InstalledAt    string `json:"installedAt"`
}

func InstallRoot(cacheRoot string, version string, target TargetSpec) string {
	root, err := managedInstallRoot(cacheRoot, version, target)
	if err != nil {
		panic(err)
	}
	return root
}

func Install(ctx context.Context, opts InstallOptions) (InstallResult, error) {
	target := targetOrCurrent(opts.Target)
	cacheRoot := defaultCacheRoot(opts.CacheRoot)
	if opts.RuntimeVersion == "" {
		opts.RuntimeVersion = DefaultRuntimeVersion
	}
	if opts.StaleLockAfter <= 0 {
		opts.StaleLockAfter = defaultStaleLockAfter
	}
	if opts.MaxArchiveBytes <= 0 {
		opts.MaxArchiveBytes = DefaultMaxArchiveBytes
	}
	manifest, manifestBytes, envelope, archivePath, cleanup, err := resolveInstallInputs(ctx, opts, target)
	if err != nil {
		return InstallResult{}, err
	}
	defer cleanup()
	if len(opts.TrustRoots) == 0 {
		opts.TrustRoots = DefaultTrustRoots()
	}
	if err := VerifyManifestSignature(manifestBytes, envelope, opts.TrustRoots); err != nil {
		return InstallResult{}, err
	}
	record, err := manifest.Target(target.Triple)
	if err != nil {
		return InstallResult{}, err
	}
	if manifest.RuntimeVersion != opts.RuntimeVersion {
		return InstallResult{}, fmt.Errorf("runtimebin: manifest runtime version %s does not match requested %s", manifest.RuntimeVersion, opts.RuntimeVersion)
	}
	finalRoot, err := managedInstallRoot(cacheRoot, opts.RuntimeVersion, target)
	if err != nil {
		return InstallResult{}, err
	}
	lockPath := finalRoot + ".lock"
	releaseLock, err := acquireLock(ctx, lockPath, opts.StaleLockAfter)
	if err != nil {
		return InstallResult{}, err
	}
	defer releaseLock()
	if result, ok, err := verifyInstalledRoot(finalRoot, record, opts.RuntimeVersion, target); err != nil {
		return InstallResult{}, err
	} else if ok {
		return InstallResult{Runtime: result, Target: record}, nil
	}
	tempRoot, err := os.MkdirTemp(filepath.Dir(finalRoot), target.Triple+".install-*")
	if err != nil {
		return InstallResult{}, err
	}
	defer os.RemoveAll(tempRoot)

	if err := extractArchive(archivePath, tempRoot, record, opts.MaxArchiveBytes); err != nil {
		return InstallResult{}, err
	}
	if err := writeInstallMetadata(tempRoot, record, opts.RuntimeVersion, target); err != nil {
		return InstallResult{}, err
	}
	if err := os.MkdirAll(filepath.Dir(finalRoot), 0o755); err != nil {
		return InstallResult{}, err
	}
	if err := os.Rename(tempRoot, finalRoot); err != nil {
		if errors.Is(err, os.ErrExist) || strings.Contains(err.Error(), "file exists") {
			if result, ok, verifyErr := verifyInstalledRoot(finalRoot, record, opts.RuntimeVersion, target); verifyErr == nil && ok {
				return InstallResult{Runtime: result, Target: record}, nil
			}
		}
		return InstallResult{}, err
	}
	result, ok, err := verifyInstalledRoot(finalRoot, record, opts.RuntimeVersion, target)
	if err != nil {
		return InstallResult{}, err
	}
	if !ok {
		return InstallResult{}, fmt.Errorf("runtimebin: installed runtime verification failed")
	}
	return InstallResult{Runtime: result, Target: record}, nil
}

func Remove(cacheRoot string, runtimeVersion string, target TargetSpec) error {
	root, err := managedInstallRoot(cacheRoot, runtimeVersion, target)
	if err != nil {
		return err
	}
	return os.RemoveAll(root)
}

func VerifyInstalled(cacheRoot string, runtimeVersion string, target TargetSpec, manifest ManifestTarget) (ResolvedRuntime, error) {
	root, err := managedInstallRoot(cacheRoot, runtimeVersion, target)
	if err != nil {
		return ResolvedRuntime{}, err
	}
	result, ok, err := verifyInstalledRoot(root, manifest, runtimeVersion, target)
	if err != nil {
		return ResolvedRuntime{}, err
	}
	if !ok {
		return ResolvedRuntime{}, ErrRuntimeNotInstalled
	}
	return result, nil
}

func resolveInstallInputs(ctx context.Context, opts InstallOptions, target TargetSpec) (Manifest, []byte, SignatureEnvelope, string, func(), error) {
	if opts.ManifestPath != "" && opts.SignaturePath != "" && opts.ArchivePath != "" {
		manifest, manifestBytes, err := LoadManifest(opts.ManifestPath)
		if err != nil {
			return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
		}
		envelope, err := LoadSignatureEnvelope(opts.SignaturePath)
		if err != nil {
			return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
		}
		return manifest, manifestBytes, envelope, opts.ArchivePath, func() {}, nil
	}
	client := withSecureRedirectPolicy(withTimeoutHTTPClient(opts.HTTPClient, opts.HTTPTimeout))
	manifestURL := opts.ManifestURL
	signatureURL := opts.SignatureURL
	if manifestURL == "" {
		baseURL := strings.TrimRight(opts.BaseURL, "/")
		if !strings.HasPrefix(baseURL, "https://") {
			return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, fmt.Errorf("runtimebin: online install requires an https base URL")
		}
		manifestURL = baseURL + "/codex-sdk-go-runtime-manifest.json"
		signatureURL = baseURL + "/codex-sdk-go-runtime-manifest.json.sig"
	}
	if !strings.HasPrefix(manifestURL, "https://") || !strings.HasPrefix(signatureURL, "https://") {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, fmt.Errorf("runtimebin: online install requires https manifest URLs")
	}
	manifestBytes, err := downloadURL(ctx, client, manifestURL, opts.MaxArchiveBytes)
	if err != nil {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	var manifest Manifest
	if err := json.Unmarshal(manifestBytes, &manifest); err != nil {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	signatureBytes, err := downloadURL(ctx, client, signatureURL, 64<<10)
	if err != nil {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	var envelope SignatureEnvelope
	if err := json.Unmarshal(signatureBytes, &envelope); err != nil {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	record, err := manifest.Target(target.Triple)
	if err != nil {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	archiveURL := opts.BaseURL
	if archiveURL != "" {
		archiveURL = strings.TrimRight(opts.BaseURL, "/") + "/" + record.ArchiveName
	}
	if !strings.HasPrefix(archiveURL, "https://") {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, fmt.Errorf("runtimebin: online install requires https archive URLs")
	}
	if err := enforceSameHostPolicy(manifestURL, signatureURL, archiveURL); err != nil {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	tempArchive, err := os.CreateTemp("", "codex-runtime-*.tar.gz")
	if err != nil {
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	tempArchive.Close()
	bytes, err := downloadURL(ctx, client, archiveURL, opts.MaxArchiveBytes)
	if err != nil {
		os.Remove(tempArchive.Name())
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	if err := os.WriteFile(tempArchive.Name(), bytes, 0o644); err != nil {
		os.Remove(tempArchive.Name())
		return Manifest{}, nil, SignatureEnvelope{}, "", func() {}, err
	}
	return manifest, manifestBytes, envelope, tempArchive.Name(), func() { _ = os.Remove(tempArchive.Name()) }, nil
}

func downloadURL(ctx context.Context, client *http.Client, rawURL string, maxBytes int64) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	resp, err := client.Do(req)
	if err != nil {
		return nil, sanitizeRequestError(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("runtimebin: download %s failed: %s", sanitizeURL(rawURL), resp.Status)
	}
	if resp.ContentLength > maxBytes {
		return nil, fmt.Errorf("runtimebin: download %s exceeds size limit", sanitizeURL(rawURL))
	}
	reader := io.LimitReader(resp.Body, maxBytes+1)
	data, err := io.ReadAll(reader)
	if err != nil {
		return nil, err
	}
	if int64(len(data)) > maxBytes {
		return nil, fmt.Errorf("runtimebin: download %s exceeds size limit", sanitizeURL(rawURL))
	}
	return data, nil
}

func acquireLock(ctx context.Context, lockPath string, staleAfter time.Duration) (func(), error) {
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, err
	}
	for {
		file, err := os.OpenFile(lockPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o644)
		if err == nil {
			_, _ = file.WriteString(time.Now().UTC().Format(time.RFC3339))
			return func() {
				file.Close()
				os.Remove(lockPath)
			}, nil
		}
		if !os.IsExist(err) {
			return nil, err
		}
		if staleAfter > 0 {
			info, statErr := os.Stat(lockPath)
			if statErr == nil && time.Since(info.ModTime()) > staleAfter {
				_ = os.Remove(lockPath)
				continue
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(lockPollInterval):
		}
	}
}

func extractArchive(archivePath string, dest string, record ManifestTarget, maxArchiveBytes int64) error {
	sha, size, err := fileSHA256AndSize(archivePath)
	if err != nil {
		return err
	}
	if sha != record.ArchiveSHA256 {
		return fmt.Errorf("runtimebin: archive sha256 mismatch: got %s want %s", sha, record.ArchiveSHA256)
	}
	if size != record.ArchiveSize {
		return fmt.Errorf("runtimebin: archive size mismatch: got %d want %d", size, record.ArchiveSize)
	}
	if size > maxArchiveBytes {
		return fmt.Errorf("runtimebin: archive exceeds size limit")
	}
	file, err := os.Open(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()
	gz, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	expected := map[string]ManifestFile{}
	for _, item := range record.Files {
		expected[item.Path] = item
	}
	written := map[string]struct{}{}
	var total int64
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("runtimebin: unsupported tar entry %q type=%d", header.Name, header.Typeflag)
		}
		clean, ok := safeArchivePath(header.Name)
		if !ok {
			return fmt.Errorf("runtimebin: invalid archive path %q", header.Name)
		}
		expectedFile, ok := expected[clean]
		if !ok {
			return fmt.Errorf("runtimebin: unexpected archive file %q", clean)
		}
		if header.Size != expectedFile.Size {
			return fmt.Errorf("runtimebin: archive file size mismatch for %q", clean)
		}
		total += header.Size
		if total > maxArchiveBytes {
			return fmt.Errorf("runtimebin: extracted files exceed size limit")
		}
		path := filepath.Join(dest, filepath.FromSlash(clean))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return err
		}
		output, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(expectedFile.Mode))
		if err != nil {
			return err
		}
		digest := sha256.New()
		writtenBytes, err := io.Copy(io.MultiWriter(output, digest), tr)
		output.Close()
		if err != nil {
			return err
		}
		if writtenBytes != expectedFile.Size {
			return fmt.Errorf("runtimebin: extracted file size mismatch for %q", clean)
		}
		if hex.EncodeToString(digest.Sum(nil)) != expectedFile.SHA256 {
			return fmt.Errorf("runtimebin: extracted file sha256 mismatch for %q", clean)
		}
		if err := os.Chmod(path, os.FileMode(expectedFile.Mode)); err != nil {
			return err
		}
		written[clean] = struct{}{}
	}
	for _, item := range record.Files {
		if _, ok := written[item.Path]; !ok {
			return fmt.Errorf("runtimebin: archive missing file %q", item.Path)
		}
	}
	return nil
}

func safeArchivePath(name string) (string, bool) {
	if name == "" || strings.Contains(name, `\`) || filepath.IsAbs(name) {
		return "", false
	}
	clean := filepath.Clean(name)
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", false
	}
	return filepath.ToSlash(clean), true
}

func verifyInstalledRoot(root string, record ManifestTarget, runtimeVersion string, target TargetSpec) (ResolvedRuntime, bool, error) {
	info, err := os.Stat(root)
	if err != nil {
		if os.IsNotExist(err) {
			return ResolvedRuntime{}, false, nil
		}
		return ResolvedRuntime{}, false, err
	}
	if !info.IsDir() {
		return ResolvedRuntime{}, false, fmt.Errorf("runtimebin: install root is not a directory: %s", root)
	}
	for _, item := range record.Files {
		path := filepath.Join(root, filepath.FromSlash(item.Path))
		sha, size, err := fileSHA256AndSize(path)
		if err != nil {
			if os.IsNotExist(err) {
				return ResolvedRuntime{}, false, nil
			}
			return ResolvedRuntime{}, false, err
		}
		if sha != item.SHA256 || size != item.Size {
			return ResolvedRuntime{}, false, nil
		}
	}
	return ResolvedRuntime{
		BinaryPath: filepath.Join(root, "bin", target.Executable),
		RootDir:    root,
		PathDirs:   TargetPathDirs(root),
		Source:     "managed",
		Target:     target,
		Version:    runtimeVersion,
	}, true, nil
}

func writeInstallMetadata(root string, record ManifestTarget, runtimeVersion string, target TargetSpec) error {
	bytes, err := json.MarshalIndent(installMetadata{
		RuntimeVersion: runtimeVersion,
		Triple:         target.Triple,
		ArchiveName:    record.ArchiveName,
		ArchiveSHA256:  record.ArchiveSHA256,
		InstalledAt:    time.Now().UTC().Format(time.RFC3339),
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, ".codex-sdk-runtime.json"), append(bytes, '\n'), 0o644)
}

func defaultCacheRoot(root string) string {
	if root != "" {
		return root
	}
	cache, err := os.UserCacheDir()
	if err != nil {
		return filepath.Join(os.TempDir(), "codex-sdk-go-runtime")
	}
	return filepath.Join(cache, "codex-sdk-go")
}

func managedInstallRoot(cacheRoot string, version string, target TargetSpec) (string, error) {
	normalized, err := NormalizeRuntimeVersion(version)
	if err != nil {
		return "", err
	}
	base := filepath.Join(defaultCacheRoot(cacheRoot), "targets", target.Triple)
	root := filepath.Join(base, normalized)
	rel, err := filepath.Rel(base, root)
	if err != nil {
		return "", err
	}
	if rel == "." || rel == "" || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("runtimebin: invalid managed runtime version %q", version)
	}
	return root, nil
}

func withSecureRedirectPolicy(client *http.Client) *http.Client {
	copyClient := *client
	basePolicy := client.CheckRedirect
	copyClient.CheckRedirect = func(req *http.Request, via []*http.Request) error {
		if req.URL == nil || req.URL.Scheme != "https" {
			return fmt.Errorf("runtimebin: redirected to non-https URL")
		}
		if len(via) > 0 && via[0].URL != nil && !strings.EqualFold(via[0].URL.Host, req.URL.Host) {
			return fmt.Errorf("runtimebin: redirected to unexpected host %q", req.URL.Host)
		}
		if basePolicy != nil {
			return basePolicy(req, via)
		}
		if len(via) >= 10 {
			return errors.New("stopped after 10 redirects")
		}
		return nil
	}
	return &copyClient
}

func withTimeoutHTTPClient(client *http.Client, timeout time.Duration) *http.Client {
	if timeout <= 0 {
		timeout = DefaultHTTPTimeout
	}
	if client == nil {
		return &http.Client{Timeout: timeout}
	}
	copyClient := *client
	if copyClient.Timeout == 0 {
		copyClient.Timeout = timeout
	}
	return &copyClient
}

func enforceSameHostPolicy(urls ...string) error {
	var host string
	for _, raw := range urls {
		parsed, err := url.Parse(raw)
		if err != nil {
			return err
		}
		if parsed.Scheme != "https" {
			return fmt.Errorf("runtimebin: URL must use https")
		}
		if host == "" {
			host = parsed.Host
			continue
		}
		if !strings.EqualFold(host, parsed.Host) {
			return fmt.Errorf("runtimebin: URLs must share one host")
		}
	}
	return nil
}

func sanitizeURL(raw string) string {
	parsed, err := url.Parse(raw)
	if err != nil {
		return "<redacted-url>"
	}
	parsed.User = nil
	parsed.RawQuery = ""
	parsed.Fragment = ""
	return parsed.String()
}

func sanitizeRequestError(err error) error {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		return fmt.Errorf("%s %s: %v", urlErr.Op, sanitizeURL(urlErr.URL), urlErr.Err)
	}
	return err
}
