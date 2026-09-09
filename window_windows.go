//go:build windows

package main

import (
	"errors"
	"os/exec"
	"syscall"
)

// start goes through the registry that knows where browsers live, so no path
// has to be guessed at. HideWindow keeps the console it runs in out of sight.
func window(address string) error {
	for _, browser := range []string{"msedge", "chrome"} {
		made := exec.Command("cmd", "/c", "start", "", browser, "--app="+address)
		made.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

		if err := made.Run(); err == nil {
			return nil
		}
	}

	return errors.New("no edge or chrome here to open a window with")
}
