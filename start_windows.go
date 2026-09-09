//go:build windows

package main

import (
	"os"
	"os/exec"
)

const runs = `HKCU\Software\Microsoft\Windows\CurrentVersion\Run`

func starting() bool {
	return exec.Command("reg", "query", runs, "/v", "netwatch").Run() == nil
}

func start(with bool) error {
	if !with {
		_ = exec.Command("reg", "delete", runs, "/v", "netwatch", "/f").Run()

		return nil
	}

	where, err := os.Executable()
	if err != nil {
		return err
	}

	return exec.Command("reg", "add", runs, "/v", "netwatch",
		"/t", "REG_SZ", "/d", where, "/f").Run()
}
