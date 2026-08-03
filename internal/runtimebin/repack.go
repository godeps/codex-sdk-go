package runtimebin

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	licenseFileName          = "LICENSE"
	noticeFileName           = "NOTICE"
	sbomFileName             = "sbom.spdx.json"
	provenanceFileName       = "provenance.json"
	nativeSignaturesFileName = "native-signatures.json"
)

type RepackOptions struct {
	SDKVersion       string
	RuntimeVersion   string
	Target           TargetSpec
	UpstreamArchive  string
	DownloadUpstream bool
	UpstreamBaseURL  string
	HTTPTimeout      time.Duration
	StageDir         string
	OutputArchive    string
	LicensePath      string
	NoticePath       string
	SBOMPath         string
	ProvenancePath   string
	NativeSignPath   string
	HTTPClient       *http.Client
}

type NativeSignatureEvidence struct {
	Available bool              `json:"available"`
	Verified  bool              `json:"verified"`
	Reason    string            `json:"reason,omitempty"`
	Files     []SignatureRecord `json:"files,omitempty"`
}

type SignatureRecord struct {
	Path    string `json:"path"`
	Tool    string `json:"tool,omitempty"`
	Subject string `json:"subject,omitempty"`
	Status  string `json:"status"`
	Raw     string `json:"raw,omitempty"`
}

type Provenance struct {
	GeneratedAt           string   `json:"generatedAt"`
	SDKVersion            string   `json:"sdkVersion"`
	RuntimeVersion        string   `json:"runtimeVersion"`
	Target                string   `json:"target"`
	UpstreamRepo          string   `json:"upstreamRepo"`
	UpstreamTag           string   `json:"upstreamTag"`
	UpstreamAsset         string   `json:"upstreamAsset"`
	UpstreamArchiveSHA256 string   `json:"upstreamArchiveSha256"`
	UpstreamArchiveSize   int64    `json:"upstreamArchiveSize"`
	StageFiles            []string `json:"stageFiles"`
}

func RepackRuntime(ctx context.Context, opts RepackOptions) (ManifestTarget, error) {
	if opts.SDKVersion == "" {
		opts.SDKVersion = DefaultSDKVersion
	}
	if opts.RuntimeVersion == "" {
		opts.RuntimeVersion = DefaultRuntimeVersion
	}
	upstreamArchive := opts.UpstreamArchive
	cleanup := func() {}
	if upstreamArchive == "" || opts.DownloadUpstream {
		path, release, err := DownloadUpstreamArchive(ctx, opts)
		if err != nil {
			return ManifestTarget{}, err
		}
		upstreamArchive = path
		cleanup = release
	}
	defer cleanup()

	stageDir := opts.StageDir
	if stageDir == "" {
		tempDir, err := os.MkdirTemp("", "codex-runtime-stage-*")
		if err != nil {
			return ManifestTarget{}, err
		}
		stageDir = tempDir
		defer os.RemoveAll(stageDir)
	}
	upstreamSHA, upstreamSize, err := fileSHA256AndSize(upstreamArchive)
	if err != nil {
		return ManifestTarget{}, err
	}
	if err := stageUpstreamArchive(upstreamArchive, stageDir, opts.Target); err != nil {
		return ManifestTarget{}, err
	}
	if err := writeReleaseEvidence(stageDir, opts, upstreamSHA, upstreamSize); err != nil {
		return ManifestTarget{}, err
	}
	record, err := PackageArchive(PackageOptions{
		PackageDir:          stageDir,
		OutputArchive:       opts.OutputArchive,
		RuntimeVersion:      opts.RuntimeVersion,
		Target:              opts.Target,
		UpstreamArchiveSHA:  upstreamSHA,
		UpstreamArchiveSize: upstreamSize,
	})
	if err != nil {
		return ManifestTarget{}, err
	}
	return record, nil
}

func DownloadUpstreamArchive(ctx context.Context, opts RepackOptions) (string, func(), error) {
	tag, err := ReleaseTagForVersion(opts.RuntimeVersion)
	if err != nil {
		return "", nil, err
	}
	baseURL := strings.TrimRight(opts.UpstreamBaseURL, "/")
	if baseURL == "" {
		baseURL = fmt.Sprintf("https://github.com/%s/releases/download/%s", DefaultUpstreamRepo, tag)
	}
	if !strings.HasPrefix(baseURL, "https://") {
		return "", nil, fmt.Errorf("runtimebin: upstream base URL must use https")
	}
	client := opts.HTTPClient
	client = withTimeoutHTTPClient(client, opts.HTTPTimeout)
	url := baseURL + "/" + opts.Target.UpstreamAsset
	bytes, err := downloadURL(ctx, client, url, DefaultMaxArchiveBytes)
	if err != nil {
		return "", nil, err
	}
	file, err := os.CreateTemp("", "codex-upstream-*.tar.gz")
	if err != nil {
		return "", nil, err
	}
	if _, err := file.Write(bytes); err != nil {
		file.Close()
		os.Remove(file.Name())
		return "", nil, err
	}
	if err := file.Close(); err != nil {
		os.Remove(file.Name())
		return "", nil, err
	}
	return file.Name(), func() { _ = os.Remove(file.Name()) }, nil
}

func stageUpstreamArchive(archivePath string, stageDir string, target TargetSpec) error {
	if err := os.MkdirAll(stageDir, 0o755); err != nil {
		return err
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
	expected := expectedUpstreamPaths(target)
	seen := map[string]struct{}{}
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		if header.Typeflag == tar.TypeDir {
			continue
		}
		if header.Typeflag != tar.TypeReg {
			return fmt.Errorf("runtimebin: upstream archive entry %q uses unsupported type %d", header.Name, header.Typeflag)
		}
		relative, ok := safeArchivePath(header.Name)
		if !ok {
			return fmt.Errorf("runtimebin: invalid upstream archive path %q", header.Name)
		}
		if _, ok := expected[relative]; !ok {
			return fmt.Errorf("runtimebin: unexpected upstream archive file %q", relative)
		}
		dest := filepath.Join(stageDir, filepath.FromSlash(relative))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		out, err := os.OpenFile(dest, os.O_CREATE|os.O_EXCL|os.O_WRONLY, os.FileMode(header.Mode))
		if err != nil {
			return err
		}
		if _, err := io.Copy(out, tr); err != nil {
			out.Close()
			return err
		}
		if err := out.Close(); err != nil {
			return err
		}
		if err := os.Chmod(dest, os.FileMode(header.Mode)); err != nil {
			return err
		}
		seen[relative] = struct{}{}
	}
	for path := range expected {
		if _, ok := seen[path]; !ok {
			return fmt.Errorf("runtimebin: upstream archive missing %q", path)
		}
	}
	return nil
}

func expectedUpstreamPaths(target TargetSpec) map[string]struct{} {
	paths := map[string]struct{}{
		"bin/" + target.Executable:              {},
		"codex-package.json":                    {},
		"codex-path/" + rgExecutable(target):    {},
		"bin/" + codeModeHostExecutable(target): {},
	}
	switch target.GOOS {
	case "linux":
		paths["codex-resources/bwrap"] = struct{}{}
		paths["codex-resources/zsh/bin/zsh"] = struct{}{}
	case "darwin":
		paths["codex-resources/zsh/bin/zsh"] = struct{}{}
	case "windows":
		paths["codex-resources/codex-command-runner.exe"] = struct{}{}
		paths["codex-resources/codex-windows-sandbox-setup.exe"] = struct{}{}
	}
	return paths
}

func writeReleaseEvidence(stageDir string, opts RepackOptions, upstreamSHA string, upstreamSize int64) error {
	if err := copyOrDefaultLicense(stageDir, opts.LicensePath); err != nil {
		return err
	}
	if err := copyOrGeneratedNotice(stageDir, opts); err != nil {
		return err
	}
	if opts.SBOMPath != "" {
		if err := copyFile(filepath.Join(stageDir, sbomFileName), opts.SBOMPath, 0o644); err != nil {
			return err
		}
	} else {
		if err := writeSBOM(stageDir, opts); err != nil {
			return err
		}
	}
	if opts.ProvenancePath != "" {
		if err := copyFile(filepath.Join(stageDir, provenanceFileName), opts.ProvenancePath, 0o644); err != nil {
			return err
		}
	} else {
		if err := writeProvenance(stageDir, opts, upstreamSHA, upstreamSize); err != nil {
			return err
		}
	}
	if opts.NativeSignPath != "" {
		return copyFile(filepath.Join(stageDir, nativeSignaturesFileName), opts.NativeSignPath, 0o644)
	}
	evidence, err := CollectNativeSignatureEvidence(stageDir, opts.Target)
	if err != nil {
		return err
	}
	return writeJSONFile(filepath.Join(stageDir, nativeSignaturesFileName), evidence)
}

func copyOrDefaultLicense(stageDir string, source string) error {
	dest := filepath.Join(stageDir, licenseFileName)
	if source != "" {
		return copyFile(dest, source, 0o644)
	}
	if _, err := os.Stat("LICENSE"); err == nil {
		return copyFile(dest, "LICENSE", 0o644)
	}
	return os.WriteFile(dest, []byte("Runtime packaging license material not provided.\n"), 0o644)
}

func copyOrGeneratedNotice(stageDir string, opts RepackOptions) error {
	dest := filepath.Join(stageDir, noticeFileName)
	if opts.NoticePath != "" {
		return copyFile(dest, opts.NoticePath, 0o644)
	}
	tag, err := ReleaseTagForVersion(opts.RuntimeVersion)
	if err != nil {
		return err
	}
	notice := fmt.Sprintf("Codex SDK Go runtime package\n\nSDK version: %s\nRuntime version: %s\nUpstream: %s %s\nAsset: %s\n", opts.SDKVersion, opts.RuntimeVersion, DefaultUpstreamRepo, tag, opts.Target.UpstreamAsset)
	return os.WriteFile(dest, []byte(notice), 0o644)
}

func writeSBOM(stageDir string, opts RepackOptions) error {
	type fileEntry struct {
		Path     string `json:"fileName"`
		Checksum string `json:"sha256"`
	}
	files, err := collectPackageFiles(stageDir)
	if err != nil {
		return err
	}
	entries := make([]fileEntry, 0, len(files))
	for _, file := range files {
		if file.Path == sbomFileName {
			continue
		}
		entries = append(entries, fileEntry{Path: file.Path, Checksum: file.SHA256})
	}
	document := map[string]any{
		"spdxVersion": "SPDX-2.3",
		"dataLicense": "CC0-1.0",
		"SPDXID":      "SPDXRef-DOCUMENT",
		"name":        fmt.Sprintf("codex-sdk-go-runtime-%s-%s", opts.RuntimeVersion, opts.Target.Triple),
		"creationInfo": map[string]any{
			"created":  time.Now().UTC().Format(time.RFC3339),
			"creators": []string{"Tool: codex-sdk-runtime"},
		},
		"files": entries,
	}
	return writeJSONFile(filepath.Join(stageDir, sbomFileName), document)
}

func writeProvenance(stageDir string, opts RepackOptions, upstreamSHA string, upstreamSize int64) error {
	tag, err := ReleaseTagForVersion(opts.RuntimeVersion)
	if err != nil {
		return err
	}
	files, err := collectPackageFiles(stageDir)
	if err != nil {
		return err
	}
	stageFiles := make([]string, 0, len(files))
	for _, file := range files {
		stageFiles = append(stageFiles, file.Path)
	}
	sort.Strings(stageFiles)
	provenance := Provenance{
		GeneratedAt:           time.Now().UTC().Format(time.RFC3339),
		SDKVersion:            opts.SDKVersion,
		RuntimeVersion:        opts.RuntimeVersion,
		Target:                opts.Target.Triple,
		UpstreamRepo:          DefaultUpstreamRepo,
		UpstreamTag:           tag,
		UpstreamAsset:         opts.Target.UpstreamAsset,
		UpstreamArchiveSHA256: upstreamSHA,
		UpstreamArchiveSize:   upstreamSize,
		StageFiles:            stageFiles,
	}
	return writeJSONFile(filepath.Join(stageDir, provenanceFileName), provenance)
}

func allVerified(records []SignatureRecord) bool {
	for _, record := range records {
		if record.Status != "verified" {
			return false
		}
	}
	return len(records) > 0
}

func codeModeHostExecutable(target TargetSpec) string {
	if target.GOOS == "windows" {
		return "codex-code-mode-host.exe"
	}
	return "codex-code-mode-host"
}

func rgExecutable(target TargetSpec) string {
	if target.GOOS == "windows" {
		return "rg.exe"
	}
	return "rg"
}

func copyFile(dest string, source string, mode os.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	out, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	if _, err := io.Copy(out, in); err != nil {
		out.Close()
		return err
	}
	return out.Close()
}

func writeJSONFile(path string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(bytes, '\n'), 0o644)
}

func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
