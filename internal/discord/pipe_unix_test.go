//go:build !windows

package discord

import (
	"path/filepath"
	"strings"
	"testing"
)

// Where the socket can be is the one part of finding Discord that is pure
// thought, and the one part a machine without Discord on it can check.
func TestLooksWhereTheSocketCanBe(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "/run/user/1000")
	t.Setenv("TMPDIR", "")
	t.Setenv("TMP", "")
	t.Setenv("TEMP", "")

	dirs := where()

	for _, wanted := range []string{
		"/run/user/1000",
		filepath.Join("/run/user/1000", "app", "com.discordapp.Discord"),
		filepath.Join("/run/user/1000", "snap.discord"),
		"/tmp",
	} {
		if !holds(dirs, wanted) {
			t.Errorf("never looks in %s", wanted)
		}
	}
}

// A sandboxed Discord cannot write to the shared runtime directory and puts
// the socket in its own corner of it.
func TestLooksInTheSandboxCorners(t *testing.T) {
	t.Setenv("XDG_RUNTIME_DIR", "")
	t.Setenv("TMPDIR", "")
	t.Setenv("TMP", "")
	t.Setenv("TEMP", "")

	corners := 0

	for _, dir := range where() {
		if strings.Contains(dir, "com.discordapp") || strings.Contains(dir, "snap.discord") {
			corners++
		}
	}

	if corners < 3 {
		t.Errorf("only %d corners looked in", corners)
	}
}

func holds(all []string, one string) bool {
	for _, each := range all {
		if each == one {
			return true
		}
	}

	return false
}
