//go:build windows

package discord

import (
	"fmt"
	"io"
	"os"
)

// A named pipe is a path on Windows, and a busy one answers with an error
// rather than a wait.
func dial() (io.ReadWriteCloser, error) {
	for i := 0; i < 10; i++ {
		pipe, err := os.OpenFile(fmt.Sprintf(`\\.\pipe\discord-ipc-%d`, i), os.O_RDWR, 0)
		if err == nil {
			return pipe, nil
		}
	}

	return nil, ErrNoClient
}
