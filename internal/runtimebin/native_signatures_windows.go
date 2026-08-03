//go:build windows

package runtimebin

import (
	"os/exec"
	"path/filepath"
	"strings"
)

func CollectNativeSignatureEvidence(stageDir string, target TargetSpec) (NativeSignatureEvidence, error) {
	switch target.GOOS {
	case "darwin":
		return NativeSignatureEvidence{Available: false, Reason: "codesign verification requires a darwin host"}, nil
	case "windows":
		records := make([]SignatureRecord, 0, 2)
		for _, path := range []string{
			filepath.Join(stageDir, "bin", target.Executable),
			filepath.Join(stageDir, "bin", codeModeHostExecutable(target)),
		} {
			cmd := exec.Command("powershell", "-NoLogo", "-NoProfile", "-Command", "(Get-AuthenticodeSignature -FilePath '"+path+"') | ConvertTo-Json -Compress")
			raw, err := cmd.CombinedOutput()
			status := "verified"
			if err != nil || !strings.Contains(string(raw), `"Status":"Valid"`) {
				status = "failed"
			}
			records = append(records, SignatureRecord{
				Path:   filepath.Base(path),
				Tool:   "Get-AuthenticodeSignature",
				Status: status,
				Raw:    strings.TrimSpace(string(raw)),
			})
		}
		return NativeSignatureEvidence{Available: true, Verified: allVerified(records), Files: records}, nil
	default:
		return NativeSignatureEvidence{Available: false, Reason: "native code signatures are not distributed for this target"}, nil
	}
}
