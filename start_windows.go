//go:build windows

package main

import (
	"os"
	"os/exec"
	"syscall"
)

const runs = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

// reg is a console program, and Windows flashes a window for one unless it is
// told not to. This is asked every time the page draws itself.
func quietly(name string, args ...string) *exec.Cmd {
	made := exec.Command(name, args...)
	made.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}

	return made
}

func starting() bool {
	return quietly("reg", "query", runs, "/v", "netwatch").Run() == nil
}

func start(with bool) error {
	if !with {
		_ = quietly("reg", "delete", runs, "/v", "netwatch", "/f").Run()

		return nil
	}

	where, err := os.Executable()
	if err != nil {
		return err
	}

	return quietly("reg", "add", runs, "/v", "netwatch",
		"/t", "REG_SZ", "/d", where, "/f").Run()
}
