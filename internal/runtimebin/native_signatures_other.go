//go:build !darwin && !windows

package runtimebin

func CollectNativeSignatureEvidence(_ string, target TargetSpec) (NativeSignatureEvidence, error) {
	switch target.GOOS {
	case "darwin":
		return NativeSignatureEvidence{Available: false, Reason: "codesign verification requires a darwin host"}, nil
	case "windows":
		return NativeSignatureEvidence{Available: false, Reason: "Authenticode verification requires a windows host"}, nil
	default:
		return NativeSignatureEvidence{Available: false, Reason: "native code signatures are not distributed for this target"}, nil
	}
}
