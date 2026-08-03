package runtimebin

import (
	"fmt"
	"runtime"
	"sort"
	"strings"
)

const (
	DefaultRuntimeVersion = "0.144.4"
	DefaultUpstreamRepo   = "openai/codex"
)

// DefaultSDKVersion is the release version embedded into generated runtime manifests.
// Override it with -ldflags "-X github.com/godeps/codex-sdk-go/internal/runtimebin.DefaultSDKVersion=vX.Y.Z"
// or the CLI --sdk-version flag during release packaging.
var DefaultSDKVersion = "v0.2.0"

type TargetSpec struct {
	GOOS          string `json:"goos"`
	GOARCH        string `json:"goarch"`
	Triple        string `json:"triple"`
	Executable    string `json:"executable"`
	UpstreamAsset string `json:"upstreamAsset"`
}

var supportedTargets = []TargetSpec{
	{GOOS: "darwin", GOARCH: "amd64", Triple: "x86_64-apple-darwin", Executable: "codex", UpstreamAsset: "codex-package-x86_64-apple-darwin.tar.gz"},
	{GOOS: "darwin", GOARCH: "arm64", Triple: "aarch64-apple-darwin", Executable: "codex", UpstreamAsset: "codex-package-aarch64-apple-darwin.tar.gz"},
	{GOOS: "linux", GOARCH: "amd64", Triple: "x86_64-unknown-linux-musl", Executable: "codex", UpstreamAsset: "codex-package-x86_64-unknown-linux-musl.tar.gz"},
	{GOOS: "linux", GOARCH: "arm64", Triple: "aarch64-unknown-linux-musl", Executable: "codex", UpstreamAsset: "codex-package-aarch64-unknown-linux-musl.tar.gz"},
	{GOOS: "windows", GOARCH: "amd64", Triple: "x86_64-pc-windows-msvc", Executable: "codex.exe", UpstreamAsset: "codex-package-x86_64-pc-windows-msvc.tar.gz"},
	{GOOS: "windows", GOARCH: "arm64", Triple: "aarch64-pc-windows-msvc", Executable: "codex.exe", UpstreamAsset: "codex-package-aarch64-pc-windows-msvc.tar.gz"},
}

func SupportedTargets() []TargetSpec {
	out := make([]TargetSpec, len(supportedTargets))
	copy(out, supportedTargets)
	return out
}

func SortTargets(targets []TargetSpec) {
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Triple < targets[j].Triple
	})
}

func CurrentTarget() (TargetSpec, error) {
	return LookupTarget(runtime.GOOS, runtime.GOARCH)
}

func LookupTarget(goos, goarch string) (TargetSpec, error) {
	for _, target := range supportedTargets {
		if target.GOOS == goos && target.GOARCH == goarch {
			return target, nil
		}
	}
	return TargetSpec{}, fmt.Errorf("runtimebin: unsupported target %s/%s", goos, goarch)
}

func LookupTargetByTriple(triple string) (TargetSpec, error) {
	for _, target := range supportedTargets {
		if target.Triple == triple {
			return target, nil
		}
	}
	return TargetSpec{}, fmt.Errorf("runtimebin: unsupported target triple %q", triple)
}

func ArchiveName(runtimeVersion string, target TargetSpec) string {
	return fmt.Sprintf("codex-sdk-go-runtime_%s_%s_%s.tar.gz", runtimeVersion, target.GOOS, target.GOARCH)
}

func TargetPathDirs(root string) []string {
	return []string{
		joinRuntimePath(root, "bin"),
		joinRuntimePath(root, "codex-path"),
	}
}

func joinRuntimePath(root string, child string) string {
	if strings.HasSuffix(root, "/") || strings.HasSuffix(root, "\\") {
		return root + child
	}
	return root + string(pathSeparator(root)) + child
}

func pathSeparator(root string) rune {
	if strings.ContainsRune(root, '\\') {
		return '\\'
	}
	return '/'
}
