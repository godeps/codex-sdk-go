//go:build darwin

package runtimebin

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func CollectNativeSignatureEvidence(stageDir string, target TargetSpec) (NativeSignatureEvidence, error) {
	switch target.GOOS {
	case "windows":
		return NativeSignatureEvidence{Available: false, Reason: "Authenticode verification requires a windows host"}, nil
	case "darwin":
		records := make([]SignatureRecord, 0, 2)
		for _, path := range []string{
			filepath.Join(stageDir, "bin", target.Executable),
			filepath.Join(stageDir, "bin", codeModeHostExecutable(target)),
		} {
			raw, err := exec.Command("codesign", "--verify", "--verbose=4", path).CombinedOutput()
			status := "verified"
			if err != nil {
				status = "failed"
			}
			records = append(records, SignatureRecord{
				Path:   filepath.Base(path),
				Tool:   "codesign",
				Status: status,
				Raw:    strings.TrimSpace(string(raw)),
			})
		}
		return NativeSignatureEvidence{Available: true, Verified: allVerified(records), Files: records}, nil
	default:
		return NativeSignatureEvidence{Available: false, Reason: "native code signatures are not distributed for this target"}, nil
	}
}
