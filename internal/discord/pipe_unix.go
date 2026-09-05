//go:build !windows

package discord

import (
	"fmt"
	"io"
	"net"
	"os"
	"path/filepath"
)

// A second client takes the next number rather than failing, so all ten count.
func dial() (io.ReadWriteCloser, error) {
	for _, dir := range where() {
		for i := 0; i < 10; i++ {
			socket := filepath.Join(dir, fmt.Sprintf("discord-ipc-%d", i))

			if conn, err := net.Dial("unix", socket); err == nil {
				return conn, nil
			}
		}
	}

	return nil, ErrNoClient
}

// A Discord from Flatpak or Snap cannot write to the shared runtime directory
// and puts the socket in its own corner of it.
func where() []string {
	var dirs []string

	for _, name := range []string{"XDG_RUNTIME_DIR", "TMPDIR", "TMP", "TEMP"} {
		if dir := os.Getenv(name); dir != "" {
			dirs = append(dirs, dir)
		}
	}

	dirs = append(dirs, "/tmp")

	var all []string

	for _, dir := range dirs {
		all = append(all, dir,
			filepath.Join(dir, "app", "com.discordapp.Discord"),
			filepath.Join(dir, "app", "com.discordapp.DiscordCanary"),
			filepath.Join(dir, "snap.discord"))
	}

	return all
}
