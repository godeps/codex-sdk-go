//go:build !unix

package appserver

// There is no POSIX SIGINT wait status to match on these platforms.
func exitedByInterrupt(error) bool { return false }
