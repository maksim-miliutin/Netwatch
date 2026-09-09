package main

import (
	"os"
	"path/filepath"
	"testing"
)

// An exe gets double clicked, not typed, so the id has to survive a run.
func TestRemembersAnIdForNextTime(t *testing.T) {
	dir := t.TempDir()

	if id, err := remembered("424242", dir); err != nil || id != "424242" {
		t.Fatalf("got %q %v", id, err)
	}

	id, err := remembered("", dir)
	if err != nil || id != "424242" {
		t.Errorf("got %q %v the second time", id, err)
	}
}

func TestHasNoIdBeforeAnybodyGivesOne(t *testing.T) {
	id, err := remembered("", t.TempDir())

	if err != nil || id != "" {
		t.Errorf("got %q %v", id, err)
	}
}

// An id is a number, so the word cannot be one.
func TestForgetsTheIdWhenToldOff(t *testing.T) {
	dir := t.TempDir()

	_, _ = remembered("424242", dir)

	if id, err := remembered("off", dir); err != nil || id != "" {
		t.Fatalf("got %q %v", id, err)
	}

	if _, err := os.Stat(filepath.Join(dir, "discord")); !os.IsNotExist(err) {
		t.Error("the file is still there")
	}

	if id, _ := remembered("", dir); id != "" {
		t.Errorf("still remembers %q", id)
	}
}

func TestForgettingNothingIsNotAnError(t *testing.T) {
	if _, err := remembered("off", t.TempDir()); err != nil {
		t.Errorf("got %v", err)
	}
}

// A file somebody typed into by hand has a newline at the end of it.
func TestReadsAnIdWrittenByHand(t *testing.T) {
	dir := t.TempDir()

	if err := os.WriteFile(filepath.Join(dir, "discord"), []byte("  424242 \n"), 0o600); err != nil {
		t.Fatal(err)
	}

	if id, _ := remembered("", dir); id != "424242" {
		t.Errorf("got %q", id)
	}
}
