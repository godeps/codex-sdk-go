package main

import (
	"bufio"
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
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

	clientOptions := []codex.Option{codex.WithAllowPATH(true)}
	if *apiKey != "" {
		clientOptions = append(clientOptions, codex.WithAPIKey(*apiKey))
	}
	if *baseURL != "" {
		clientOptions = append(clientOptions, codex.WithBaseURL(*baseURL))
	}
	client, err := codex.NewClient(context.Background(), clientOptions...)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start client:", err)
		return
	}
	defer client.Close()

	threadOptions := codex.ThreadOptions{
		WorkingDirectory: ".",
		SkipGitRepoCheck: true,
		SandboxMode:      codex.SandboxDangerFullAccess,
		ApprovalPolicy:   codex.ApprovalNever,
	}
	if *model != "" {
		threadOptions.Model = *model
	}
	thread, err := client.StartThread(context.Background(), threadOptions)
	if err != nil {
		fmt.Fprintln(os.Stderr, "start thread:", err)
		return
	}

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
			handle, err := thread.StartTurnContext(ctx, codex.TextInput(line), codex.TurnOptions{})
			if err != nil {
				cancel()
				fmt.Fprintln(os.Stderr, "run error:", err)
				continue
			}
			streamed, err := handle.StreamContext(ctx)
			if err != nil {
				cancel()
				fmt.Fprintln(os.Stderr, "stream error:", err)
				continue
			}
			for {
				event, err := streamed.Next(ctx)
				if errors.Is(err, io.EOF) {
					break
				}
				if err != nil {
					fmt.Fprintln(os.Stderr, "stream error:", err)
					break
				}
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
			_ = streamed.Close()
			cancel()
			continue
		}

		turn, err := thread.RunContext(ctx, codex.TextInput(line), codex.TurnOptions{})
		cancel()
		if err != nil {
			fmt.Fprintln(os.Stderr, "run error:", err)
			continue
		}

		fmt.Println(turn.FinalResponse)
	}
}
