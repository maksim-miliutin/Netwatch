//go:build !windows

package main

func tray(address string, open func(), quit func()) {
	select {}
}
