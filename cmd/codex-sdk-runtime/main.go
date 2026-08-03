package main

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/godeps/codex-sdk-go/internal/runtimebin"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	if len(args) == 0 {
		return usageError()
	}
	switch args[0] {
	case "package":
		return runPackage(args[1:])
	case "repack":
		return runRepack(ctx, args[1:])
	case "manifest":
		return runManifest(args[1:])
	case "sign":
		return runSign(args[1:])
	case "install":
		return runInstall(ctx, args[1:])
	case "native-smoke":
		return runNativeSmoke(ctx, args[1:])
	case "verify":
		return runVerify(args[1:])
	case "path":
		return runPath(args[1:])
	case "remove":
		return runRemove(args[1:])
	case "version":
		return runVersion(args[1:])
	default:
		return usageError()
	}
}

func runPackage(args []string) error {
	fs := flag.NewFlagSet("package", flag.ContinueOnError)
	var (
		packageDir     = fs.String("package-dir", "", "package directory to archive")
		archivePath    = fs.String("archive", "", "output archive path")
		targetTriple   = fs.String("target", "", "target triple")
		runtimeVersion = fs.String("runtime-version", runtimebin.DefaultRuntimeVersion, "runtime version")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *packageDir == "" || *archivePath == "" || *targetTriple == "" {
		return errors.New("package requires --package-dir, --archive, and --target")
	}
	target, err := runtimebin.LookupTargetByTriple(*targetTriple)
	if err != nil {
		return err
	}
	record, err := runtimebin.PackageArchive(runtimebin.PackageOptions{
		PackageDir:     *packageDir,
		OutputArchive:  *archivePath,
		RuntimeVersion: *runtimeVersion,
		Target:         target,
	})
	if err != nil {
		return err
	}
	return writeJSON(os.Stdout, record)
}

func runRepack(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("repack", flag.ContinueOnError)
	var (
		targetTriple     = fs.String("target", "", "target triple")
		runtimeVersion   = fs.String("runtime-version", runtimebin.DefaultRuntimeVersion, "runtime version")
		sdkVersion       = fs.String("sdk-version", runtimebin.DefaultSDKVersion, "sdk version")
		upstreamArchive  = fs.String("upstream-archive", "", "path to upstream codex-package archive")
		downloadUpstream = fs.Bool("download-upstream", false, "download the upstream codex-package asset")
		upstreamBaseURL  = fs.String("upstream-base-url", "", "https base URL for upstream release assets")
		httpTimeout      = fs.Duration("http-timeout", runtimebin.DefaultHTTPTimeout, "HTTP timeout for upstream downloads")
		stageDir         = fs.String("stage-dir", "", "output stage directory")
		archivePath      = fs.String("archive", "", "output repacked archive path")
		recordPath       = fs.String("record", "", "optional output path for the target record JSON")
		licensePath      = fs.String("license", "", "optional LICENSE file to include")
		noticePath       = fs.String("notice", "", "optional NOTICE file to include")
		sbomPath         = fs.String("sbom", "", "optional SPDX SBOM JSON to include")
		provenancePath   = fs.String("provenance", "", "optional provenance JSON to include")
		nativeSignPath   = fs.String("native-signatures", "", "optional native-signature evidence JSON to include")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *targetTriple == "" || *archivePath == "" {
		return errors.New("repack requires --target and --archive")
	}
	target, err := runtimebin.LookupTargetByTriple(*targetTriple)
	if err != nil {
		return err
	}
	record, err := runtimebin.RepackRuntime(ctx, runtimebin.RepackOptions{
		SDKVersion:       *sdkVersion,
		RuntimeVersion:   *runtimeVersion,
		Target:           target,
		UpstreamArchive:  *upstreamArchive,
		DownloadUpstream: *downloadUpstream,
		UpstreamBaseURL:  *upstreamBaseURL,
		HTTPTimeout:      *httpTimeout,
		StageDir:         *stageDir,
		OutputArchive:    *archivePath,
		LicensePath:      *licensePath,
		NoticePath:       *noticePath,
		SBOMPath:         *sbomPath,
		ProvenancePath:   *provenancePath,
		NativeSignPath:   *nativeSignPath,
	})
	if err != nil {
		return err
	}
	if *recordPath != "" {
		bytes, err := json.MarshalIndent(record, "", "  ")
		if err != nil {
			return err
		}
		if err := os.WriteFile(*recordPath, append(bytes, '\n'), 0o644); err != nil {
			return err
		}
	}
	return writeJSON(os.Stdout, record)
}

type manifestRecordFlag []string

func (m *manifestRecordFlag) String() string {
	return strings.Join(*m, ",")
}

func (m *manifestRecordFlag) Set(value string) error {
	*m = append(*m, value)
	return nil
}

func runManifest(args []string) error {
	fs := flag.NewFlagSet("manifest", flag.ContinueOnError)
	var (
		sdkVersion     = fs.String("sdk-version", runtimebin.DefaultSDKVersion, "sdk version")
		runtimeVersion = fs.String("runtime-version", runtimebin.DefaultRuntimeVersion, "runtime version")
		upstreamRepo   = fs.String("upstream-repo", runtimebin.DefaultUpstreamRepo, "upstream repository")
		outputPath     = fs.String("output", "", "manifest output path")
		recordPaths    manifestRecordFlag
	)
	fs.Var(&recordPaths, "record", "path to a target record JSON produced by package")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *outputPath == "" || len(recordPaths) == 0 {
		return errors.New("manifest requires --output and at least one --record")
	}
	targets := make([]runtimebin.ManifestTarget, 0, len(recordPaths))
	for _, path := range recordPaths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var target runtimebin.ManifestTarget
		if err := json.Unmarshal(bytes, &target); err != nil {
			return err
		}
		targets = append(targets, target)
	}
	manifest, err := runtimebin.BuildManifest(*sdkVersion, *runtimeVersion, *upstreamRepo, targets)
	if err != nil {
		return err
	}
	bytes, err := manifest.CanonicalBytes()
	if err != nil {
		return err
	}
	return os.WriteFile(*outputPath, append(bytes, '\n'), 0o644)
}

func runSign(args []string) error {
	fs := flag.NewFlagSet("sign", flag.ContinueOnError)
	var (
		manifestPath = fs.String("manifest", "", "manifest path")
		outputPath   = fs.String("output", "", "signature output path")
		keyID        = fs.String("key-id", "", "trust root key identifier")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	privateKey := os.Getenv("CODEX_RUNTIME_MANIFEST_PRIVATE_KEY_BASE64")
	if *manifestPath == "" || *outputPath == "" || *keyID == "" || privateKey == "" {
		return errors.New("sign requires --manifest, --output, --key-id, and CODEX_RUNTIME_MANIFEST_PRIVATE_KEY_BASE64")
	}
	manifestBytes, err := os.ReadFile(*manifestPath)
	if err != nil {
		return err
	}
	decoded, err := decodeBase64(privateKey)
	if err != nil {
		return err
	}
	envelope, err := runtimebin.SignManifest(manifestBytes, *keyID, decoded)
	if err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(envelope, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(*outputPath, append(bytes, '\n'), 0o644)
}

func runInstall(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("install", flag.ContinueOnError)
	var (
		cacheRoot      = fs.String("cache-root", "", "runtime cache root")
		targetTriple   = fs.String("target", "", "target triple; defaults to current platform")
		runtimeVersion = fs.String("runtime-version", runtimebin.DefaultRuntimeVersion, "runtime version")
		manifestPath   = fs.String("manifest", "", "manifest path")
		signaturePath  = fs.String("manifest-signature", "", "manifest signature path")
		archivePath    = fs.String("archive", "", "archive path")
		baseURL        = fs.String("base-url", "", "https base URL for manifest and archives")
		httpTimeout    = fs.Duration("http-timeout", runtimebin.DefaultHTTPTimeout, "HTTP timeout for online manifest/archive downloads")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	target, err := lookupTargetFlag(*targetTriple)
	if err != nil {
		return err
	}
	result, err := runtimebin.Install(ctx, runtimebin.InstallOptions{
		CacheRoot:      *cacheRoot,
		RuntimeVersion: *runtimeVersion,
		Target:         target,
		ManifestPath:   *manifestPath,
		SignaturePath:  *signaturePath,
		ArchivePath:    *archivePath,
		BaseURL:        *baseURL,
		HTTPTimeout:    *httpTimeout,
	})
	if err != nil {
		return err
	}
	return writeJSON(os.Stdout, result)
}

func runNativeSmoke(ctx context.Context, args []string) error {
	fs := flag.NewFlagSet("native-smoke", flag.ContinueOnError)
	var (
		runtimePath    = fs.String("runtime-path", "", "path to the runtime binary")
		rootDir        = fs.String("root-dir", "", "runtime root directory containing bin/ and codex-path/")
		targetTriple   = fs.String("target", "", "target triple")
		runtimeVersion = fs.String("runtime-version", runtimebin.DefaultRuntimeVersion, "runtime version")
		recordPath     = fs.String("record", "", "optional target record JSON path")
		manifestHash   = fs.String("manifest-sha256", "", "optional canonical manifest sha256")
		evidencePath   = fs.String("evidence", "", "optional evidence JSON output path")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *runtimePath == "" || *rootDir == "" || *targetTriple == "" {
		return errors.New("native-smoke requires --runtime-path, --root-dir, and --target")
	}
	target, err := runtimebin.LookupTargetByTriple(*targetTriple)
	if err != nil {
		return err
	}
	var record *runtimebin.ManifestTarget
	if *recordPath != "" {
		bytes, err := os.ReadFile(*recordPath)
		if err != nil {
			return err
		}
		var value runtimebin.ManifestTarget
		if err := json.Unmarshal(bytes, &value); err != nil {
			return err
		}
		record = &value
	}
	result, err := runNativeSmokeCommand(ctx, *runtimePath, *rootDir, target, *runtimeVersion, record, *manifestHash)
	if err != nil {
		return err
	}
	if *evidencePath != "" {
		if err := runtimebin.WriteEvidence(*evidencePath, result); err != nil {
			return err
		}
	}
	return writeJSON(os.Stdout, result)
}

func runVerify(args []string) error {
	fs := flag.NewFlagSet("verify", flag.ContinueOnError)
	var (
		cacheRoot      = fs.String("cache-root", "", "runtime cache root")
		targetTriple   = fs.String("target", "", "target triple")
		runtimeVersion = fs.String("runtime-version", runtimebin.DefaultRuntimeVersion, "runtime version")
		manifestPath   = fs.String("manifest", filepath.Join("runtime", "manifest.json"), "manifest path")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	target, err := lookupTargetFlag(*targetTriple)
	if err != nil {
		return err
	}
	manifest, _, err := runtimebin.LoadManifest(*manifestPath)
	if err != nil {
		return err
	}
	record, err := manifest.Target(target.Triple)
	if err != nil {
		return err
	}
	result, err := runtimebin.VerifyInstalled(*cacheRoot, *runtimeVersion, *target, record)
	if err != nil {
		return err
	}
	return writeJSON(os.Stdout, result)
}

func runPath(args []string) error {
	fs := flag.NewFlagSet("path", flag.ContinueOnError)
	var (
		cacheRoot      = fs.String("cache-root", "", "runtime cache root")
		targetTriple   = fs.String("target", "", "target triple")
		runtimeVersion = fs.String("runtime-version", runtimebin.DefaultRuntimeVersion, "runtime version")
		allowPATH      = fs.Bool("allow-path", false, "allow PATH fallback")
		explicit       = fs.String("runtime-path", "", "explicit runtime binary path")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	target, err := lookupTargetFlag(*targetTriple)
	if err != nil {
		return err
	}
	resolved, err := runtimebin.ResolveRuntime(runtimebin.ResolveOptions{
		ExplicitBinary: *explicit,
		Env:            environmentMap(),
		CacheRoot:      *cacheRoot,
		RuntimeVersion: *runtimeVersion,
		AllowPATH:      *allowPATH,
		Target:         target,
	})
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, resolved.BinaryPath)
	return nil
}

func runRemove(args []string) error {
	fs := flag.NewFlagSet("remove", flag.ContinueOnError)
	var (
		cacheRoot      = fs.String("cache-root", "", "runtime cache root")
		targetTriple   = fs.String("target", "", "target triple")
		runtimeVersion = fs.String("runtime-version", runtimebin.DefaultRuntimeVersion, "runtime version")
	)
	if err := fs.Parse(args); err != nil {
		return err
	}
	target, err := lookupTargetFlag(*targetTriple)
	if err != nil {
		return err
	}
	return runtimebin.Remove(*cacheRoot, *runtimeVersion, *target)
}

func runVersion(args []string) error {
	if len(args) != 1 {
		return errors.New("version requires one argument")
	}
	tag, err := runtimebin.ReleaseTagForVersion(args[0])
	if err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, tag)
	return nil
}

func usageError() error {
	return errors.New("usage: codex-sdk-runtime <package|repack|manifest|sign|install|native-smoke|verify|path|remove|version>")
}

func lookupTargetFlag(triple string) (*runtimebin.TargetSpec, error) {
	if triple == "" {
		target, err := runtimebin.CurrentTarget()
		if err != nil {
			return nil, err
		}
		return &target, nil
	}
	target, err := runtimebin.LookupTargetByTriple(triple)
	if err != nil {
		return nil, err
	}
	return &target, nil
}

func writeJSON(file *os.File, value any) error {
	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func environmentMap() map[string]string {
	env := map[string]string{}
	for _, entry := range os.Environ() {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) == 2 {
			env[parts[0]] = parts[1]
		}
	}
	return env
}

func decodeBase64(value string) ([]byte, error) {
	decoded, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, err
	}
	return decoded, nil
}
