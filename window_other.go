//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"runtime"
)

// Neither of these takes an app mode, so this is an ordinary tab.
func window(address string) error {
	opener := "xdg-open"
	if runtime.GOOS == "darwin" {
		opener = "open"
	}

	if err := exec.Command(opener, address).Run(); err != nil {
		return errors.New("nothing here would open " + address)
	}

	return nil
}
