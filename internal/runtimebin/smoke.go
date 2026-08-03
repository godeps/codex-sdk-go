package runtimebin

import (
	"encoding/json"
	"os"
)

type RunnerIdentity struct {
	Hostname     string `json:"hostname,omitempty"`
	RunnerName   string `json:"runnerName,omitempty"`
	RunnerOS     string `json:"runnerOS,omitempty"`
	RunnerArch   string `json:"runnerArch,omitempty"`
	ImageOS      string `json:"imageOS,omitempty"`
	ImageVersion string `json:"imageVersion,omitempty"`
}

type NativeSmokeUsage struct {
	InputTokens           int64 `json:"input_tokens,omitempty"`
	CachedInputTokens     int64 `json:"cached_input_tokens,omitempty"`
	CacheWriteInputTokens int64 `json:"cache_write_input_tokens,omitempty"`
	OutputTokens          int64 `json:"output_tokens,omitempty"`
	ReasoningOutputTokens int64 `json:"reasoning_output_tokens,omitempty"`
	TotalTokens           int64 `json:"total_tokens,omitempty"`
}

type NativeSmokeResult struct {
	TargetTriple         string `json:"targetTriple"`
	RuntimeVersion       string `json:"runtimeVersion"`
	RuntimeGOOS          string `json:"runtimeGOOS"`
	RuntimeGOARCH        string `json:"runtimeGOARCH"`
	BinaryPath           string `json:"binaryPath"`
	VersionOutput        string `json:"versionOutput"`
	InitializeSucceeded  bool   `json:"initializeSucceeded"`
	CloseSucceeded       bool   `json:"closeSucceeded"`
	ModelListSucceeded   bool   `json:"modelListSucceeded"`
	ThreadStartSucceeded bool   `json:"threadStartSucceeded"`
	TurnStartSucceeded   bool   `json:"turnStartSucceeded"`
	StreamSucceeded      bool   `json:"streamSucceeded"`
	ProtocolVersion      string `json:"protocolVersion"`
	ServerInfo           *struct {
		Name    string `json:"name"`
		Version string `json:"version"`
	} `json:"serverInfo,omitempty"`
	ModelCount            int                     `json:"modelCount,omitempty"`
	ModelID               string                  `json:"modelID,omitempty"`
	ThreadID              string                  `json:"threadID,omitempty"`
	TurnID                string                  `json:"turnID,omitempty"`
	TurnStatus            string                  `json:"turnStatus,omitempty"`
	StreamEventCount      int                     `json:"streamEventCount,omitempty"`
	CompletedItemCount    int                     `json:"completedItemCount,omitempty"`
	FinalResponse         string                  `json:"finalResponse,omitempty"`
	Usage                 *NativeSmokeUsage       `json:"usage,omitempty"`
	MockRequestCount      int                     `json:"mockRequestCount,omitempty"`
	ArchiveName           string                  `json:"archiveName,omitempty"`
	ArchiveSHA256         string                  `json:"archiveSha256,omitempty"`
	ArchiveSize           int64                   `json:"archiveSize,omitempty"`
	UpstreamArchiveSHA256 string                  `json:"upstreamArchiveSha256,omitempty"`
	UpstreamArchiveSize   int64                   `json:"upstreamArchiveSize,omitempty"`
	ManifestSHA256        string                  `json:"manifestSha256,omitempty"`
	Runner                RunnerIdentity          `json:"runner"`
	NativeSignatures      NativeSignatureEvidence `json:"nativeSignatures"`
}

func WriteEvidence(path string, value any) error {
	bytes, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(bytes, '\n'), 0o644)
}
