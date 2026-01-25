package codex

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

const internalOriginatorEnv = "CODEX_INTERNAL_ORIGINATOR_OVERRIDE"
const goSDKOriginator = "codex_sdk_go"

// CodexExecArgs describes CLI arguments for codex exec.
type CodexExecArgs struct {
	Input string

	BaseURL  string
	APIKey   string
	ThreadID string
	Images   []string

	Model                 string
	SandboxMode           SandboxMode
	WorkingDirectory      string
	AdditionalDirectories []string
	SkipGitRepoCheck      bool
	OutputSchemaFile      string
	ModelReasoningEffort  ModelReasoningEffort
	NetworkAccessEnabled  *bool
	WebSearchMode         WebSearchMode
	WebSearchEnabled      *bool
	ApprovalPolicy        ApprovalMode

	Context context.Context
}

// ExecStream streams lines from the Codex CLI.
type ExecStream struct {
	Lines <-chan string
	wait  func() error
}

// Wait blocks until the process exits and returns any error.
func (s *ExecStream) Wait() error {
	if s == nil || s.wait == nil {
		return nil
	}
	return s.wait()
}

// CodexExec launches the Codex CLI.
type CodexExec struct {
	executablePath string
	envOverride    map[string]string
}

// NewCodexExec creates a new exec wrapper.
func NewCodexExec(executablePath string, env map[string]string) *CodexExec {
	path := executablePath
	if path == "" {
		path = findCodexPath()
	}
	return &CodexExec{
		executablePath: path,
		envOverride:    env,
	}
}

// Run launches the Codex CLI and streams JSONL lines.
func (c *CodexExec) Run(args CodexExecArgs) (*ExecStream, error) {
	commandArgs := []string{"exec", "--experimental-json"}

	if args.Model != "" {
		commandArgs = append(commandArgs, "--model", args.Model)
	}
	if args.SandboxMode != "" {
		commandArgs = append(commandArgs, "--sandbox", string(args.SandboxMode))
	}
	if args.WorkingDirectory != "" {
		commandArgs = append(commandArgs, "--cd", args.WorkingDirectory)
	}
	if len(args.AdditionalDirectories) > 0 {
		for _, dir := range args.AdditionalDirectories {
			commandArgs = append(commandArgs, "--add-dir", dir)
		}
	}
	if args.SkipGitRepoCheck {
		commandArgs = append(commandArgs, "--skip-git-repo-check")
	}
	if args.OutputSchemaFile != "" {
		commandArgs = append(commandArgs, "--output-schema", args.OutputSchemaFile)
	}
	if args.ModelReasoningEffort != "" {
		commandArgs = append(commandArgs, "--config", fmt.Sprintf("model_reasoning_effort=%q", args.ModelReasoningEffort))
	}
	if args.NetworkAccessEnabled != nil {
		commandArgs = append(commandArgs, "--config", fmt.Sprintf("sandbox_workspace_write.network_access=%t", *args.NetworkAccessEnabled))
	}
	if args.WebSearchMode != "" {
		commandArgs = append(commandArgs, "--config", fmt.Sprintf("web_search=%q", args.WebSearchMode))
	} else if args.WebSearchEnabled != nil {
		if *args.WebSearchEnabled {
			commandArgs = append(commandArgs, "--config", "web_search=\"live\"")
		} else {
			commandArgs = append(commandArgs, "--config", "web_search=\"disabled\"")
		}
	}
	if args.ApprovalPolicy != "" {
		commandArgs = append(commandArgs, "--config", fmt.Sprintf("approval_policy=%q", args.ApprovalPolicy))
	}
	if len(args.Images) > 0 {
		for _, image := range args.Images {
			commandArgs = append(commandArgs, "--image", image)
		}
	}
	if args.ThreadID != "" {
		commandArgs = append(commandArgs, "resume", args.ThreadID)
	}

	ctx := args.Context
	if ctx == nil {
		ctx = context.Background()
	}

	cmd := exec.CommandContext(ctx, c.executablePath, commandArgs...)
	cmd.Env = buildEnv(c.envOverride, args.BaseURL, args.APIKey)
	cmd.Stdin = strings.NewReader(args.Input)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, err
	}

	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		return nil, err
	}

	lines := make(chan string)
	scanErrCh := make(chan error, 1)

	go func() {
		defer close(lines)
		defer close(scanErrCh)
		scanner := bufio.NewScanner(stdout)
		buf := make([]byte, 0, 64*1024)
		scanner.Buffer(buf, 10*1024*1024)
		for scanner.Scan() {
			lines <- scanner.Text()
		}
		if err := scanner.Err(); err != nil {
			scanErrCh <- err
		}
	}()

	wait := func() error {
		var scanErr error
		for err := range scanErrCh {
			if err != nil {
				scanErr = err
			}
		}
		waitErr := cmd.Wait()
		if scanErr != nil {
			return scanErr
		}
		if waitErr == nil {
			return nil
		}
		var exitErr *exec.ExitError
		if errors.As(waitErr, &exitErr) {
			return fmt.Errorf("codex exec exited with %v: %s", exitErr.ProcessState, strings.TrimSpace(stderr.String()))
		}
		return waitErr
	}

	return &ExecStream{Lines: lines, wait: wait}, nil
}

func buildEnv(override map[string]string, baseURL string, apiKey string) []string {
	env := map[string]string{}
	if override != nil {
		for key, value := range override {
			env[key] = value
		}
	} else {
		for _, entry := range os.Environ() {
			parts := strings.SplitN(entry, "=", 2)
			if len(parts) == 2 {
				env[parts[0]] = parts[1]
			}
		}
	}
	if _, ok := env[internalOriginatorEnv]; !ok {
		env[internalOriginatorEnv] = goSDKOriginator
	}
	if baseURL != "" {
		env["OPENAI_BASE_URL"] = baseURL
	}
	if apiKey != "" {
		env["CODEX_API_KEY"] = apiKey
	}
	list := make([]string, 0, len(env))
	for key, value := range env {
		list = append(list, key+"="+value)
	}
	return list
}

func findCodexPath() string {
	if path, err := exec.LookPath("codex"); err == nil {
		return path
	}

	targetTriple := ""
	switch runtime.GOOS {
	case "linux", "android":
		switch runtime.GOARCH {
		case "amd64":
			targetTriple = "x86_64-unknown-linux-musl"
		case "arm64":
			targetTriple = "aarch64-unknown-linux-musl"
		}
	case "darwin":
		switch runtime.GOARCH {
		case "amd64":
			targetTriple = "x86_64-apple-darwin"
		case "arm64":
			targetTriple = "aarch64-apple-darwin"
		}
	case "windows":
		switch runtime.GOARCH {
		case "amd64":
			targetTriple = "x86_64-pc-windows-msvc"
		case "arm64":
			targetTriple = "aarch64-pc-windows-msvc"
		}
	}
	if targetTriple == "" {
		panic(fmt.Sprintf("unsupported platform: %s (%s)", runtime.GOOS, runtime.GOARCH))
	}

	_, currentFile, _, ok := runtime.Caller(0)
	if !ok {
		panic("unable to determine codex sdk path")
	}
	moduleDir := filepath.Dir(currentFile)
	vendorRoot := filepath.Join(moduleDir, "vendor")
	archRoot := filepath.Join(vendorRoot, targetTriple)
	codexBinaryName := "codex"
	if runtime.GOOS == "windows" {
		codexBinaryName = "codex.exe"
	}
	return filepath.Join(archRoot, "codex", codexBinaryName)
}
