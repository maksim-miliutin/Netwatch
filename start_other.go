//go:build !windows

package main

import "errors"

func starting() bool {
	return false
}

func start(with bool) error {
	return errors.New("starting with the machine is a Windows matter")
}
