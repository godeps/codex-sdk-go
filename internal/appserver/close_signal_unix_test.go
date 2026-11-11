//go:build unix

package appserver

import (
	"errors"
	"io"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

// A real child lets us distinguish an OS signal from an exit code or stderr.
func TestCloseSignalPeer(t *testing.T) {
	mode := os.Getenv("CODEX_SDK_SHUTDOWN_PEER")
	if mode == "" {
		return
	}
	signal.Reset(os.Interrupt, syscall.SIGTERM)
	received := make(chan os.Signal, 1)
	if mode == "interrupt-then-exit" || mode == "interrupt-then-term" {
		signal.Notify(received, os.Interrupt)
	}
	if mode == "ignore" {
		signal.Ignore(os.Interrupt)
	}
	if err := os.WriteFile(os.Getenv("CODEX_SDK_PEER_READY"), []byte("ready"), 0600); err != nil {
		os.Exit(3)
	}
	switch mode {
	case "exit130":
		_, _ = io.Copy(io.Discard, os.Stdin)
		_, _ = os.Stderr.WriteString("signal: interrupt\n")
		os.Exit(130)
	case "interrupt-then-exit":
		<-received
		os.Exit(17)
	case "interrupt-then-term":
		<-received
		_ = syscall.Kill(os.Getpid(), syscall.SIGTERM)
	}
	time.Sleep(30 * time.Second)
	os.Exit(9)
}

func newShutdownPeer(t *testing.T, mode string) *Client {
	t.Helper()
	path, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ready := filepath.Join(t.TempDir(), "ready")
	client := New(Config{
		LaunchArgs: []string{path, "-test.run=^TestCloseSignalPeer$"},
		Env:        map[string]string{"CODEX_SDK_SHUTDOWN_PEER": mode, "CODEX_SDK_PEER_READY": ready},
		Limits:     Limits{ShutdownWait: 35 * time.Millisecond, KillWait: 350 * time.Millisecond},
	})
	if err = client.startProcess(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = client.Close() })
	deadline := time.Now().Add(5 * time.Second)
	for {
		if _, err = os.Stat(ready); err == nil {
			break
		}
		if time.Now().After(deadline) {
			t.Fatal("child did not become ready")
		}
		time.Sleep(5 * time.Millisecond)
	}
	return client
}

func TestCloseAcceptsOnlyItsOwnInterrupt(t *testing.T) {
	t.Run("requested interrupt, concurrent and repeated close", func(t *testing.T) {
		client := newShutdownPeer(t, "default")
		done := make(chan error, 3)
		for range 3 {
			go func() { done <- client.Close() }()
		}
		for range 3 {
			if err := <-done; err != nil {
				t.Fatalf("Close: %v", err)
			}
		}
		if err := client.Close(); err != nil {
			t.Fatal(err)
		}
		if err := client.Wait(); !exitedByInterrupt(err) {
			t.Fatalf("Wait lost raw SIGINT: %v", err)
		}
	})
	for _, tc := range []struct {
		mode string
		code int
		sig  syscall.Signal
	}{
		{"exit130", 130, 0}, {"interrupt-then-exit", 17, 0},
		{"interrupt-then-term", -1, syscall.SIGTERM}, {"ignore", -1, syscall.SIGKILL},
		{"external", -1, syscall.SIGINT},
	} {
		t.Run(tc.mode, func(t *testing.T) {
			client := newShutdownPeer(t, tc.mode)
			if tc.mode == "external" {
				if err := client.proc.Process.Signal(os.Interrupt); err != nil {
					t.Fatal(err)
				}
				// The process is already dead before SDK shutdown begins.
				if err := client.Wait(); !exitedByInterrupt(err) {
					t.Fatalf("external signal: %v", err)
				}
			}
			for _, call := range []func() error{client.Close, client.Close, client.Wait} {
				err := call()
				var ex *exec.ExitError
				if !errors.As(err, &ex) || ex.ExitCode() != tc.code {
					t.Fatalf("expected exit %d, got %v", tc.code, err)
				}
				st := ex.Sys().(syscall.WaitStatus)
				if (tc.sig != 0) != st.Signaled() || tc.sig != 0 && st.Signal() != tc.sig {
					t.Fatalf("unexpected wait status %v", st)
				}
			}
		})
	}
}
