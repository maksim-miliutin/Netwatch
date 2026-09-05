package store

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"netwatch/internal/play"
)

func fresh(t *testing.T) *Store {
	t.Helper()

	store, err := Open(filepath.Join(t.TempDir(), "plays.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	return store
}

func one(id string, at time.Time) play.Play {
	return play.Play{Service: "youtube", ID: id, Title: id, At: at}
}

func TestKeepsWhatItWasGiven(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	if err := store.Add(one("a", now)); err != nil {
		t.Fatal(err)
	}

	got, err := store.All()
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 1 || got[0].ID != "a" {
		t.Errorf("got %+v", got)
	}
}

func TestHandsBackTheNewestFirst(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("old", now.Add(-time.Hour)))
	_ = store.Add(one("new", now))

	got, _ := store.All()
	if got[0].ID != "new" {
		t.Errorf("got %s first, wanted the newest", got[0].ID)
	}
}

// A tab left open reports itself on every check. The same thing twice running
// is one watch, and counting it twice would fill the list with one video.
func TestDoesNotWriteTheSameThingTwiceRunning(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))
	_ = store.Add(one("a", now.Add(time.Minute)))

	got, _ := store.All()
	if len(got) != 1 {
		t.Errorf("wrote %d, wanted one", len(got))
	}
}

// Going back to something after watching another thing is a second watch.
func TestWritesTheSameThingAgainAfterSomethingElse(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))
	_ = store.Add(one("b", now.Add(time.Minute)))
	_ = store.Add(one("a", now.Add(2*time.Minute)))

	got, _ := store.All()
	if len(got) != 3 {
		t.Errorf("wrote %d, wanted three", len(got))
	}
}

func TestAnswersNothingBeforeAnythingIsWritten(t *testing.T) {
	got, err := fresh(t).All()
	if err != nil || len(got) != 0 {
		t.Errorf("got %+v, %v", got, err)
	}
}

// One unreadable line should not cost somebody the rest of their history.
func TestSkipsALineItCannotRead(t *testing.T) {
	file := filepath.Join(t.TempDir(), "plays.jsonl")

	err := os.WriteFile(file, []byte(
		`{"service":"youtube","id":"a","at":"2026-09-03T12:00:00Z"}`+"\n"+
			"половина строки\n"+
			`{"service":"rutube","id":"b","at":"2026-09-03T13:00:00Z"}`+"\n"), 0o600)
	if err != nil {
		t.Fatal(err)
	}

	store, _ := Open(file)

	got, err := store.All()
	if err != nil {
		t.Fatal(err)
	}

	if len(got) != 2 {
		t.Errorf("read %d, wanted the two that were whole", len(got))
	}
}

// A play ends when the next one starts, and that is the only end most of them
// get: a browser tab knows when it opened and rarely when it stopped.
func TestClosesAPlayWhenTheNextOneStarts(t *testing.T) {
	kept := fresh(t)
	now := time.Now().UTC()

	_ = kept.Add(one("a", now))
	_ = kept.Add(one("b", now.Add(3*time.Minute)))

	got, _ := kept.All()

	var first play.Play
	for _, p := range got {
		if p.ID == "a" {
			first = p
		}
	}

	if first.Seconds != 180 {
		t.Errorf("counted %d seconds, wanted 180", first.Seconds)
	}
}

// Somebody who left a tab open overnight did not watch for nine hours, and a
// list that says they did is worse than one that says nothing.
func TestLeavesAnOvernightTabUncounted(t *testing.T) {
	kept := fresh(t)
	now := time.Now().UTC()

	_ = kept.Add(one("a", now))
	_ = kept.Add(one("b", now.Add(9*time.Hour)))

	got, _ := kept.All()

	for _, p := range got {
		if p.ID == "a" && p.Seconds != 0 {
			t.Errorf("counted %d seconds for a tab left overnight", p.Seconds)
		}
	}
}

func TestLeavesTheNewestOneOpen(t *testing.T) {
	kept := fresh(t)
	now := time.Now().UTC()

	_ = kept.Add(one("a", now))
	_ = kept.Add(one("b", now.Add(time.Minute)))

	got, _ := kept.All()
	if got[0].ID != "b" || got[0].Seconds != 0 {
		t.Errorf("got %+v, wanted the newest still open", got[0])
	}
}

// The whole file is rewritten when a play is closed, and a program stopped
// halfway through that should cost nothing.
func TestKeepsEverythingWhenItRewrites(t *testing.T) {
	kept := fresh(t)
	now := time.Now().UTC()

	for _, id := range []string{"a", "b", "c", "d"} {
		_ = kept.Add(one(id, now))
		now = now.Add(time.Minute)
	}

	got, _ := kept.All()
	if len(got) != 4 {
		t.Errorf("kept %d of four", len(got))
	}
}

// Somebody who is not sure whether an import worked will run it again.
func TestMergeCostsNothingTwice(t *testing.T) {
	kept := fresh(t)
	now := time.Now().UTC()

	incoming := []play.Play{one("a", now), one("b", now.Add(time.Hour))}

	added, err := kept.Merge(incoming)
	if err != nil || added != 2 {
		t.Fatalf("added %d, %v", added, err)
	}

	again, err := kept.Merge(incoming)
	if err != nil || again != 0 {
		t.Errorf("added %d the second time, wanted none", again)
	}

	got, _ := kept.All()
	if len(got) != 2 {
		t.Errorf("kept %d", len(got))
	}
}

// The same video watched twice is two plays; the same row imported twice is
// one. The moment is what tells them apart.
func TestMergeKeepsTheSameThingWatchedTwice(t *testing.T) {
	kept := fresh(t)
	now := time.Now().UTC()

	added, _ := kept.Merge([]play.Play{one("a", now), one("a", now.Add(time.Hour))})
	if added != 2 {
		t.Errorf("added %d, wanted two", added)
	}
}

func TestMergeSitsAlongsideWhatWasWatchedLive(t *testing.T) {
	kept := fresh(t)
	now := time.Now().UTC()

	_ = kept.Add(one("live", now))
	_, _ = kept.Merge([]play.Play{one("old", now.Add(-24*time.Hour))})

	got, _ := kept.All()
	if len(got) != 2 || got[0].ID != "live" {
		t.Errorf("got %+v", got)
	}
}
