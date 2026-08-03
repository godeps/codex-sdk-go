package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	codex "github.com/godeps/codex-sdk-go"
)

func main() {
	var (
		prompt           = flag.String("prompt", "Start the turn and keep it open", "Prompt to send to Codex")
		followUp         = flag.String("follow-up", "Please add a concise checklist.", "Follow-up input to steer into the active turn")
		workdir          = flag.String("workdir", "", "Working directory for Codex CLI")
		codexPath        = flag.String("codex-path", "", "Explicit Codex executable path")
		runtimeCacheRoot = flag.String("runtime-cache-root", "", "Managed runtime cache root")
		runtimeVersion   = flag.String("runtime-version", "", "Managed runtime version")
		allowPATH        = flag.Bool("allow-path", true, "Allow PATH lookup for the managed runtime")
		timeout          = flag.Duration("timeout", 5*time.Minute, "Timeout for the turn")
	)
	flag.Parse()

	if *workdir == "" {
		if wd, err := os.Getwd(); err == nil {
			*workdir = wd
		}
	}

	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()

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

	client, err := codex.NewClient(ctx, opts...)
	if err != nil {
		fatal(err)
	}
	defer client.Close()

	thread, err := client.StartThread(ctx, codex.ThreadOptions{
		WorkingDirectory: *workdir,
		SkipGitRepoCheck: true,
		SandboxMode:      codex.SandboxDangerFullAccess,
		ApprovalPolicy:   codex.ApprovalNever,
	})
	if err != nil {
		fatal(err)
	}

	handle, err := thread.StartTurnContext(ctx, codex.TextInput(*prompt), codex.TurnOptions{Context: ctx})
	if err != nil {
		fatal(err)
	}

	if err := handle.SteerContext(ctx, codex.TextInput(*followUp)); err != nil {
		fatal(err)
	}
	if err := handle.InterruptContext(ctx); err != nil {
		fatal(err)
	}

	fmt.Printf("turn: %s\n", handle.ID())
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
