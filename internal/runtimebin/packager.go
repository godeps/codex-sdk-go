package runtimebin

import (
	"archive/tar"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type PackageOptions struct {
	PackageDir          string
	OutputArchive       string
	RuntimeVersion      string
	Target              TargetSpec
	UpstreamArchiveSHA  string
	UpstreamArchiveSize int64
}

func PackageArchive(opts PackageOptions) (ManifestTarget, error) {
	files, err := collectPackageFiles(opts.PackageDir)
	if err != nil {
		return ManifestTarget{}, err
	}
	if err := writeArchive(opts.OutputArchive, opts.PackageDir, files); err != nil {
		return ManifestTarget{}, err
	}
	archiveSHA, archiveSize, err := fileSHA256AndSize(opts.OutputArchive)
	if err != nil {
		return ManifestTarget{}, err
	}
	return ManifestTarget{
		GOOS:                  opts.Target.GOOS,
		GOARCH:                opts.Target.GOARCH,
		Triple:                opts.Target.Triple,
		Executable:            opts.Target.Executable,
		UpstreamAsset:         opts.Target.UpstreamAsset,
		UpstreamArchiveSHA256: opts.UpstreamArchiveSHA,
		UpstreamArchiveSize:   opts.UpstreamArchiveSize,
		ArchiveName:           filepath.Base(opts.OutputArchive),
		ArchiveSHA256:         archiveSHA,
		ArchiveSize:           archiveSize,
		Files:                 files,
	}, nil
}

func collectPackageFiles(root string) ([]ManifestFile, error) {
	var files []ManifestFile
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		relative = filepath.ToSlash(relative)
		role := fileRole(relative)
		if role == "" {
			return fmt.Errorf("runtimebin: unexpected package file %q", relative)
		}
		sha, _, err := fileSHA256AndSize(path)
		if err != nil {
			return err
		}
		files = append(files, ManifestFile{
			Path:   relative,
			SHA256: sha,
			Size:   info.Size(),
			Mode:   int64(info.Mode().Perm()),
			Role:   role,
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		return files[i].Path < files[j].Path
	})
	return files, nil
}

func writeArchive(archivePath string, packageDir string, files []ManifestFile) error {
	if err := os.MkdirAll(filepath.Dir(archivePath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(archivePath)
	if err != nil {
		return err
	}
	defer file.Close()

	gz, err := gzip.NewWriterLevel(file, gzip.BestCompression)
	if err != nil {
		return err
	}
	gz.Header.ModTime = time.Unix(0, 0)
	gz.Header.OS = 255
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	for _, record := range files {
		path := filepath.Join(packageDir, filepath.FromSlash(record.Path))
		source, err := os.Open(path)
		if err != nil {
			return err
		}
		header := &tar.Header{
			Name:     record.Path,
			Mode:     record.Mode,
			Size:     record.Size,
			Typeflag: tar.TypeReg,
			ModTime:  time.Unix(0, 0),
			Format:   tar.FormatPAX,
		}
		if err := tw.WriteHeader(header); err != nil {
			source.Close()
			return err
		}
		if _, err := io.Copy(tw, source); err != nil {
			source.Close()
			return err
		}
		source.Close()
	}
	return nil
}

func BuildManifest(sdkVersion string, runtimeVersion string, upstreamRepo string, targets []ManifestTarget) (Manifest, error) {
	upstreamTag, err := ReleaseTagForVersion(runtimeVersion)
	if err != nil {
		return Manifest{}, err
	}
	sort.Slice(targets, func(i, j int) bool {
		return targets[i].Triple < targets[j].Triple
	})
	manifest := Manifest{
		SchemaVersion:  ManifestSchemaVersion,
		SDKVersion:     sdkVersion,
		RuntimeVersion: runtimeVersion,
		UpstreamRepo:   upstreamRepo,
		UpstreamTag:    upstreamTag,
		Targets:        targets,
	}
	return manifest, manifest.Validate()
}

func fileSHA256AndSize(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	digest := sha256.New()
	n, err := io.Copy(digest, file)
	if err != nil {
		return "", 0, err
	}
	return hex.EncodeToString(digest.Sum(nil)), n, nil
}

func fileRole(path string) string {
	switch {
	case path == "codex-package.json":
		return "metadata"
	case path == licenseFileName:
		return "license"
	case path == noticeFileName:
		return "notice"
	case path == sbomFileName:
		return "sbom"
	case path == provenanceFileName:
		return "provenance"
	case path == nativeSignaturesFileName:
		return "native-signature-evidence"
	case strings.HasPrefix(path, "bin/"):
		if strings.Contains(path, "codex-code-mode-host") {
			return "companion"
		}
		return "executable"
	case strings.HasPrefix(path, "codex-path/"):
		return "path"
	case strings.HasPrefix(path, "codex-resources/"):
		return "resource"
	default:
		return ""
	}
}
