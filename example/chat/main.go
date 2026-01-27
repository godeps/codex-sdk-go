package main

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/godeps/codex-sdk-go"
)

func main() {
	stream := flag.Bool("stream", true, "Stream events while the turn runs")
	timeout := flag.Duration("timeout", 5*time.Minute, "Timeout per turn")
	apiKey := flag.String("api-key", "", "CODEX_API_KEY override")
	baseURL := flag.String("base-url", "", "OPENAI_BASE_URL override")
	model := flag.String("model", "", "Model to use (e.g., claude-sonnet-4, gpt-4)")
	flag.Parse()

	options := codex.CodexOptions{}
	if *apiKey != "" {
		options.APIKey = *apiKey
	}
	if *baseURL != "" {
		options.BaseURL = *baseURL
	}
	client := codex.NewCodex(options)

	threadOptions := codex.ThreadOptions{
		WorkingDirectory: ".",
		SkipGitRepoCheck: true,
		SandboxMode:      codex.SandboxDangerFullAccess,
		ApprovalPolicy:   codex.ApprovalNever,
	}
	if *model != "" {
		threadOptions.Model = *model
	}
	thread := client.StartThread(threadOptions)

	reader := bufio.NewReader(os.Stdin)
	fmt.Println("Enter message (type 'exit' to quit):")

	for {
		fmt.Print("> ")
		line, err := reader.ReadString('\n')
		if err != nil {
			fmt.Fprintln(os.Stderr, "read error:", err)
			return
		}
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" {
			return
		}

		ctx, cancel := context.WithTimeout(context.Background(), *timeout)
		if *stream {
			streamed, err := thread.RunStreamed(codex.TextInput(line), codex.TurnOptions{Context: ctx})
			if err != nil {
				cancel()
				fmt.Fprintln(os.Stderr, "run error:", err)
				continue
			}
			for event := range streamed.Events {
				switch event.Type {
				case "item.completed":
					fmt.Printf("item: %#v\n", event.Item)
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
				fmt.Fprintln(os.Stderr, "run error:", err)
			}
			cancel()
			continue
		}

		turn, err := thread.Run(codex.TextInput(line), codex.TurnOptions{Context: ctx})
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "run error:", err)
			continue
		}

		fmt.Println(turn.FinalResponse)
	}
}
