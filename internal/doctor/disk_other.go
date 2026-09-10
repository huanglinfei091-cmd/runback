//go:build !linux

package doctor

import "fmt"

func availableBytes(string) (uint64, error) {
	return 0, fmt.Errorf("disk space check is available on Linux")
}
