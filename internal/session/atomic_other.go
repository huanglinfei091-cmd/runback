//go:build !linux

package session

import "fmt"

func publishSession(temp, final string) error {
	return fmt.Errorf("atomic session publication currently requires Linux")
}
