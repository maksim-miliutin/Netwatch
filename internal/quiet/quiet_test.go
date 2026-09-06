package quiet

import (
	"os"
	"path/filepath"
	"testing"
)

func fresh(t *testing.T) *List {
	t.Helper()

	list, err := Open(filepath.Join(t.TempDir(), "quiet"))
	if err != nil {
		t.Fatal(err)
	}

	return list
}

func TestShowsEverythingBeforeAnybodySaysOtherwise(t *testing.T) {
	if fresh(t).Hidden("youtube") {
		t.Error("hiding something nobody asked to hide")
	}
}

func TestHidesWhatItWasTold(t *testing.T) {
	list := fresh(t)

	if err := list.Hide([]string{"twitch", "spotify"}); err != nil {
		t.Fatal(err)
	}

	if !list.Hidden("twitch") || !list.Hidden("spotify") {
		t.Error("did not hide what it was given")
	}

	if list.Hidden("youtube") {
		t.Error("hid something it was not given")
	}
}

func TestRemembersAcrossRuns(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "quiet")

	first, _ := Open(file)
	_ = first.Hide([]string{"twitch"})

	second, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}

	if !second.Hidden("twitch") {
		t.Error("forgot between runs")
	}
}

// The whole set at once: unticking a box has to take the service off the list,
// not leave it there because nobody mentioned it.
func TestHidingAgainReplacesWhatWasThere(t *testing.T) {
	list := fresh(t)

	_ = list.Hide([]string{"twitch"})
	_ = list.Hide([]string{"spotify"})

	if list.Hidden("twitch") {
		t.Error("still hiding what was left out the second time")
	}
}

// A file somebody typed into by hand has blank lines and stray spaces in it.
func TestReadsAFileWrittenByHand(t *testing.T) {
	file := filepath.Join(t.TempDir(), "quiet")

	if err := os.WriteFile(file, []byte("  twitch \n\n\nspotify\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	list, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}

	if !list.Hidden("twitch") || !list.Hidden("spotify") {
		t.Errorf("read %v", list.Names())
	}
}
