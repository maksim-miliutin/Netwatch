//go:build !windows

package main

import (
	"errors"
	"os/exec"
	"runtime"
)

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
