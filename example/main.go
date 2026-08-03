package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godeps/codex-sdk-go"
)

func main() {
	var (
		prompt           = flag.String("prompt", "Summarize repository status", "Prompt to send to Codex")
		workdir          = flag.String("workdir", "", "Working directory for Codex CLI")
		skipGit          = flag.Bool("skip-git-check", true, "Skip git repository check")
		model            = flag.String("model", "", "Model name override")
		apiKey           = flag.String("api-key", "", "CODEX_API_KEY override")
		baseURL          = flag.String("base-url", "", "OPENAI_BASE_URL override")
		codexPath        = flag.String("codex-path", "", "Explicit Codex executable path")
		runtimeCacheRoot = flag.String("runtime-cache-root", "", "Managed runtime cache root")
		runtimeVersion   = flag.String("runtime-version", "", "Managed runtime version")
		allowPATH        = flag.Bool("allow-path", true, "Allow PATH lookup for the managed runtime")
		approval         = flag.String("approval", "never", "Approval policy (never|on-request|on-failure|untrusted)")
		timeout          = flag.Duration("timeout", 5*time.Minute, "Timeout for the turn")
	)
	flag.Parse()

	if *workdir == "" {
		if wd, err := os.Getwd(); err == nil {
			*workdir = wd
		}
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if *timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	var opts []codex.Option
	switch {
	case *codexPath != "":
		opts = append(opts, codex.WithCodexPath(*codexPath))
	default:
		if *runtimeCacheRoot != "" {
			opts = append(opts, codex.WithRuntimeCacheRoot(*runtimeCacheRoot))
		}
		if *runtimeVersion != "" {
			opts = append(opts, codex.WithRuntimeVersion(*runtimeVersion))
		}
		if *allowPATH {
			opts = append(opts, codex.WithAllowPATH(true))
		}
	}
	if *baseURL != "" {
		opts = append(opts, codex.WithBaseURL(*baseURL))
	}
	if *apiKey != "" {
		opts = append(opts, codex.WithAPIKey(*apiKey))
	}

	client, err := codex.NewClient(ctx, opts...)
	if err != nil {
		fatal(err)
	}
	defer client.Close()

	threadOptions := codex.ThreadOptions{
		WorkingDirectory: *workdir,
		SkipGitRepoCheck: *skipGit,
		Model:            *model,
		SandboxMode:      codex.SandboxDangerFullAccess,
	}
	if *approval != "" {
		threadOptions.ApprovalPolicy = codex.ApprovalMode(*approval)
	}

	thread, err := client.StartThread(ctx, threadOptions)
	if err != nil {
		fatal(err)
	}

	turn, err := thread.RunContext(ctx, codex.TextInput(*prompt), codex.TurnOptions{Context: ctx})
	if err != nil {
		fatal(err)
	}

	fmt.Printf("thread: %s\n", thread.ID())
	fmt.Printf("prompt: %s\n", strings.TrimSpace(*prompt))
	fmt.Printf("response:\n%s\n", turn.FinalResponse)
	fmt.Printf("status: %s\n", turn.Status)
	if turn.Error != nil {
		fmt.Printf("error: %s\n", turn.Error.Message)
	}
	if turn.Usage != nil {
		fmt.Printf("usage: %+v\n", *turn.Usage)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
