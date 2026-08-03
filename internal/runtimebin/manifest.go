package runtimebin

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

const (
	ManifestSchemaVersion = 1
	SignatureAlgorithm    = "ed25519"
)

type Manifest struct {
	SchemaVersion  int              `json:"schemaVersion"`
	SDKVersion     string           `json:"sdkVersion"`
	RuntimeVersion string           `json:"runtimeVersion"`
	UpstreamRepo   string           `json:"upstreamRepo"`
	UpstreamTag    string           `json:"upstreamTag"`
	Targets        []ManifestTarget `json:"targets"`
}

type ManifestTarget struct {
	GOOS                  string         `json:"goos"`
	GOARCH                string         `json:"goarch"`
	Triple                string         `json:"triple"`
	Executable            string         `json:"executable"`
	UpstreamAsset         string         `json:"upstreamAsset"`
	UpstreamArchiveSHA256 string         `json:"upstreamArchiveSha256,omitempty"`
	UpstreamArchiveSize   int64          `json:"upstreamArchiveSize,omitempty"`
	ArchiveName           string         `json:"archiveName"`
	ArchiveSHA256         string         `json:"archiveSha256"`
	ArchiveSize           int64          `json:"archiveSize"`
	Files                 []ManifestFile `json:"files"`
}

type ManifestFile struct {
	Path   string `json:"path"`
	SHA256 string `json:"sha256"`
	Size   int64  `json:"size"`
	Mode   int64  `json:"mode"`
	Role   string `json:"role"`
}

type SignatureEnvelope struct {
	KeyID     string `json:"keyId"`
	Algorithm string `json:"algorithm"`
	Signature string `json:"signature"`
}

func (m Manifest) CanonicalBytes() ([]byte, error) {
	sorted := m
	sort.Slice(sorted.Targets, func(i, j int) bool {
		return sorted.Targets[i].Triple < sorted.Targets[j].Triple
	})
	for i := range sorted.Targets {
		sort.Slice(sorted.Targets[i].Files, func(a, b int) bool {
			return sorted.Targets[i].Files[a].Path < sorted.Targets[i].Files[b].Path
		})
	}
	return json.MarshalIndent(sorted, "", "  ")
}

func (m Manifest) Validate() error {
	if m.SchemaVersion != ManifestSchemaVersion {
		return fmt.Errorf("runtimebin: manifest schemaVersion=%d, want %d", m.SchemaVersion, ManifestSchemaVersion)
	}
	if _, err := NormalizeRuntimeVersion(m.RuntimeVersion); err != nil {
		return err
	}
	if m.SDKVersion == "" || m.UpstreamRepo == "" || m.UpstreamTag == "" {
		return fmt.Errorf("runtimebin: manifest metadata is incomplete")
	}
	if len(m.Targets) != len(supportedTargets) {
		return fmt.Errorf("runtimebin: manifest has %d targets, want %d", len(m.Targets), len(supportedTargets))
	}
	seen := map[string]struct{}{}
	for _, target := range m.Targets {
		spec, err := LookupTargetByTriple(target.Triple)
		if err != nil {
			return err
		}
		if _, ok := seen[target.Triple]; ok {
			return fmt.Errorf("runtimebin: duplicate target %q", target.Triple)
		}
		seen[target.Triple] = struct{}{}
		if target.GOOS != spec.GOOS || target.GOARCH != spec.GOARCH || target.Executable != spec.Executable || target.UpstreamAsset != spec.UpstreamAsset {
			return fmt.Errorf("runtimebin: manifest target %q does not match supported target metadata", target.Triple)
		}
		if target.ArchiveName == "" || target.ArchiveSHA256 == "" || target.ArchiveSize <= 0 {
			return fmt.Errorf("runtimebin: manifest target %q is missing archive metadata", target.Triple)
		}
		if target.UpstreamArchiveSHA256 != "" {
			if _, err := hex.DecodeString(target.UpstreamArchiveSHA256); err != nil || len(target.UpstreamArchiveSHA256) != sha256.Size*2 {
				return fmt.Errorf("runtimebin: manifest target %q has invalid upstream archive sha256", target.Triple)
			}
			if target.UpstreamArchiveSize <= 0 {
				return fmt.Errorf("runtimebin: manifest target %q has invalid upstream archive size", target.Triple)
			}
		}
		if _, err := hex.DecodeString(target.ArchiveSHA256); err != nil || len(target.ArchiveSHA256) != sha256.Size*2 {
			return fmt.Errorf("runtimebin: manifest target %q has invalid archive sha256", target.Triple)
		}
		if len(target.Files) == 0 {
			return fmt.Errorf("runtimebin: manifest target %q has no files", target.Triple)
		}
		lastPath := ""
		for _, file := range target.Files {
			if file.Path == "" || file.Path != strings.TrimPrefix(file.Path, "/") || strings.HasPrefix(file.Path, "..") || strings.Contains(file.Path, `\`) {
				return fmt.Errorf("runtimebin: invalid file path %q", file.Path)
			}
			if file.Path <= lastPath {
				return fmt.Errorf("runtimebin: manifest files for %q are not sorted", target.Triple)
			}
			lastPath = file.Path
			if _, err := hex.DecodeString(file.SHA256); err != nil || len(file.SHA256) != sha256.Size*2 {
				return fmt.Errorf("runtimebin: invalid file sha256 %q", file.Path)
			}
			if file.Size < 0 || file.Mode <= 0 || file.Role == "" {
				return fmt.Errorf("runtimebin: incomplete file metadata for %q", file.Path)
			}
		}
	}
	return nil
}

func (m Manifest) Target(triple string) (ManifestTarget, error) {
	for _, target := range m.Targets {
		if target.Triple == triple {
			return target, nil
		}
	}
	return ManifestTarget{}, fmt.Errorf("runtimebin: manifest missing target %q", triple)
}

func LoadManifest(path string) (Manifest, []byte, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		return Manifest{}, nil, err
	}
	if err := manifest.Validate(); err != nil {
		return Manifest{}, nil, err
	}
	return manifest, bytes, nil
}

func LoadSignatureEnvelope(path string) (SignatureEnvelope, error) {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return SignatureEnvelope{}, err
	}
	var envelope SignatureEnvelope
	if err := json.Unmarshal(bytes, &envelope); err != nil {
		return SignatureEnvelope{}, err
	}
	if envelope.KeyID == "" || envelope.Algorithm != SignatureAlgorithm || envelope.Signature == "" {
		return SignatureEnvelope{}, fmt.Errorf("runtimebin: invalid signature envelope")
	}
	if _, err := base64.StdEncoding.DecodeString(envelope.Signature); err != nil {
		return SignatureEnvelope{}, fmt.Errorf("runtimebin: invalid signature encoding: %w", err)
	}
	return envelope, nil
}
