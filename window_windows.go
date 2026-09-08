//go:build windows

package main

import (
	"errors"
	"os/exec"
)

// A window of its own rather than a tab. start is used rather than a path:
// it goes through the registry that knows where browsers live.
func window(address string) error {
	for _, browser := range []string{"msedge", "chrome"} {
		if err := exec.Command("cmd", "/c", "start", "", browser, "--app="+address).Run(); err == nil {
			return nil
		}
	}

	return errors.New("no edge or chrome here to open a window with")
}
