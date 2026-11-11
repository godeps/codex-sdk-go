//go:build unix

package appserver

import (
	"errors"
	"os/exec"
	"syscall"
)

func exitedByInterrupt(err error) bool {
	var exit *exec.ExitError
	if !errors.As(err, &exit) || exit.ProcessState == nil {
		return false
	}
	status, ok := exit.Sys().(syscall.WaitStatus)
	return ok && status.Signaled() && status.Signal() == syscall.SIGINT
}
