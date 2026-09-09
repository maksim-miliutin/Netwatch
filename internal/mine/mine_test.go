package mine

import (
	"os"
	"path/filepath"
	"testing"
)

func fresh(t *testing.T) *List {
	t.Helper()

	list, err := Open(filepath.Join(t.TempDir(), "mine"))
	if err != nil {
		t.Fatal(err)
	}

	return list
}

func TestKnowsNothingAtFirst(t *testing.T) {
	if got := fresh(t).All(); len(got) != 0 {
		t.Errorf("got %v", got)
	}
}

func TestKeepsWhatWasAdded(t *testing.T) {
	list := fresh(t)

	if err := list.Add("plvideo.ru", "Plvideo"); err != nil {
		t.Fatal(err)
	}

	if got := list.All()["plvideo.ru"]; got != "Plvideo" {
		t.Errorf("got %q", got)
	}
}

func TestRemembersAcrossRuns(t *testing.T) {
	file := filepath.Join(t.TempDir(), "mine")

	first, _ := Open(file)
	_ = first.Add("plvideo.ru", "Plvideo")

	second, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}

	if got := second.All()["plvideo.ru"]; got != "Plvideo" {
		t.Errorf("got %q", got)
	}
}

// A name with spaces in it, and a file somebody typed into by hand.
func TestReadsAFileWrittenByHand(t *testing.T) {
	file := filepath.Join(t.TempDir(), "mine")

	if err := os.WriteFile(file, []byte("  some.tv  Some TV \n\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	list, err := Open(file)
	if err != nil {
		t.Fatal(err)
	}

	if got := list.All()["some.tv"]; got != "Some TV" {
		t.Errorf("got %q", got)
	}
}
