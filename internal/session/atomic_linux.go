//go:build linux

package session

import "golang.org/x/sys/unix"

// A colliding final directory (including an empty one) is never replaced.
func publishSession(temp, final string) error {
	return unix.Renameat2(unix.AT_FDCWD, temp, unix.AT_FDCWD, final, unix.RENAME_NOREPLACE)
}
