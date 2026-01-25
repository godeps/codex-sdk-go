package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	"github.com/godeps/codex-sdk-go"
)

func main() {
	var (
		prompt   = flag.String("prompt", "Summarize repository status", "Prompt to send to Codex")
		threadID = flag.String("thread", "", "Thread ID to resume (optional)")
		stream   = flag.Bool("stream", false, "Stream events instead of waiting for completion")
		workdir  = flag.String("workdir", "", "Working directory for Codex CLI")
		skipGit  = flag.Bool("skip-git-check", true, "Skip git repository check")
		model    = flag.String("model", "", "Model name override")
		apiKey   = flag.String("api-key", "", "CODEX_API_KEY override")
		baseURL  = flag.String("base-url", "", "OPENAI_BASE_URL override")
		approval = flag.String("approval", "never", "Approval policy (never|on-request|on-failure|untrusted)")
		timeout  = flag.Duration("timeout", 5*time.Minute, "Timeout for the turn")
	)
	flag.Parse()

	if *workdir == "" {
		if wd, err := os.Getwd(); err == nil {
			*workdir = wd
		}
	}

	options := codex.CodexOptions{
		BaseURL: *baseURL,
		APIKey:  *apiKey,
	}
	threadOptions := codex.ThreadOptions{
		WorkingDirectory: *workdir,
		SkipGitRepoCheck: *skipGit,
		Model:            *model,
		SandboxMode:      codex.SandboxDangerFullAccess,
	}
	if *approval != "" {
		threadOptions.ApprovalPolicy = codex.ApprovalMode(*approval)
	}

	client := codex.NewCodex(options)
	var thread *codex.Thread
	if *threadID != "" {
		thread = client.ResumeThread(*threadID, threadOptions)
	} else {
		thread = client.StartThread(threadOptions)
	}

	ctx := context.Background()
	var cancel context.CancelFunc
	if *timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, *timeout)
		defer cancel()
	}

	turnOptions := codex.TurnOptions{Context: ctx}

	if *stream {
		streamed, err := thread.RunStreamed(codex.TextInput(*prompt), turnOptions)
		if err != nil {
			fatal(err)
		}
		for event := range streamed.Events {
			switch event.Type {
			case "turn.started":
				fmt.Println("turn started")
			case "item.completed":
				fmt.Printf("item: %#v\n", event.Item)
			case "item.started", "item.updated":
				fmt.Printf("item %s: %#v\n", event.Type, event.Item)
			case "turn.completed":
				fmt.Printf("usage: %+v\n", event.Usage)
			case "thread.started":
				fmt.Printf("thread: %s\n", event.ThreadID)
			case "turn.failed":
				fmt.Printf("turn failed: %v\n", event.Error)
			case "error":
				fmt.Printf("stream error: %s\n", event.Message)
			}
		}
		if err := <-streamed.Done; err != nil {
			fatal(err)
		}
		return
	}

	turn, err := thread.Run(codex.TextInput(*prompt), turnOptions)
	if err != nil {
		fatal(err)
	}

	fmt.Printf("thread: %s\n", thread.ID())
	fmt.Printf("response:\n%s\n", turn.FinalResponse)
	fmt.Printf("items: %d\n", len(turn.Items))
	if turn.Usage != nil {
		fmt.Printf("usage: %+v\n", *turn.Usage)
	}
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, err)
	os.Exit(1)
}
