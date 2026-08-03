package runtimebin

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCanonicalRuntimeManifestIsSignedAndComplete(t *testing.T) {
	manifestPath := filepath.Join("..", "..", "runtime", "manifest.json")
	signaturePath := manifestPath + ".sig"
	manifest, manifestBytes, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.SDKVersion != DefaultSDKVersion || manifest.RuntimeVersion != DefaultRuntimeVersion {
		t.Fatalf("canonical versions = SDK %q, runtime %q", manifest.SDKVersion, manifest.RuntimeVersion)
	}
	if len(manifest.Targets) != len(SupportedTargets()) {
		t.Fatalf("canonical target count = %d, want %d", len(manifest.Targets), len(SupportedTargets()))
	}
	for _, target := range SupportedTargets() {
		if _, err := manifest.Target(target.Triple); err != nil {
			t.Fatalf("canonical manifest missing %s: %v", target.Triple, err)
		}
	}
	envelope, err := LoadSignatureEnvelope(signaturePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := VerifyManifestSignature(manifestBytes, envelope, DefaultTrustRoots()); err != nil {
		t.Fatalf("verify canonical signature: %v", err)
	}
	if info, err := os.Stat(signaturePath); err != nil || info.Size() == 0 {
		t.Fatalf("canonical signature file: info=%v err=%v", info, err)
	}
}
