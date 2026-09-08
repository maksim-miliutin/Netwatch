package store

import (
	"encoding/json"
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

// Waiting for the next video to close this one leaves the last watch of every
// day uncounted.
func TestClosesWhatIsOpenWhenToldItEnded(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))

	if err := store.Stop(now.Add(3 * time.Minute)); err != nil {
		t.Fatal(err)
	}

	got, _ := store.All()
	if got[0].Seconds != 180 {
		t.Errorf("counted %d seconds, wanted 180", got[0].Seconds)
	}
}

func TestClosingNothingIsNotAnError(t *testing.T) {
	if err := fresh(t).Stop(time.Now()); err != nil {
		t.Errorf("got %v on an empty list", err)
	}
}

func TestWritesSomethingAgainOnceItWasClosed(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))
	_ = store.Stop(now.Add(time.Minute))
	_ = store.Add(one("a", now.Add(time.Hour)))

	got, _ := store.All()
	if len(got) != 2 {
		t.Errorf("wrote %d, wanted two", len(got))
	}
}

// The same tab reports itself every ten seconds, and the page redraws itself
// just as often. Neither should send the whole file through a parser again.
func TestDoesNotReadTheFileTwiceForNothing(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	if err := store.Add(one("a", now)); err != nil {
		t.Fatal(err)
	}

	if _, err := store.All(); err != nil {
		t.Fatal(err)
	}

	// The file is filled with rubbish of exactly the same length and put back
	// to the same moment. Nothing a stat can see has changed, so a read that
	// went to the disk would come back with nothing.
	was, err := os.Stat(store.file)
	if err != nil {
		t.Fatal(err)
	}

	rubbish := make([]byte, was.Size())
	for i := range rubbish {
		rubbish[i] = 'x'
	}

	if err := os.WriteFile(store.file, rubbish, 0o600); err != nil {
		t.Fatal(err)
	}

	if err := os.Chtimes(store.file, was.ModTime(), was.ModTime()); err != nil {
		t.Fatal(err)
	}

	got, err := store.All()
	if err != nil || len(got) != 1 {
		t.Fatalf("got %d %v, wanted what was kept", len(got), err)
	}
}

// All hands out a copy. Sorted in place, what is kept would end up back to
// front, and the next Add would take the oldest play for the newest.
func TestSortingTheListLeavesTheOrderAlone(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))
	_ = store.Add(one("b", now.Add(time.Minute)))

	if _, err := store.All(); err != nil {
		t.Fatal(err)
	}

	// b is the newest, so a play of b again is the same watch and writes
	// nothing. If the order had turned over, a would be taken for the newest.
	_ = store.Add(one("b", now.Add(2*time.Minute)))

	got, _ := store.All()
	if len(got) != 2 {
		t.Errorf("wrote %d, wanted two", len(got))
	}
}

// A file edited by hand is noticed: the size and the moment both change.
func TestNoticesTheFileChangingUnderIt(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))
	_, _ = store.All()

	line, _ := json.Marshal(play.Play{Service: "rutube", ID: "b", At: now.Add(time.Minute)})

	file, err := os.OpenFile(store.file, os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}

	_, _ = file.Write(append(line, '\n'))
	file.Close()

	got, _ := store.All()
	if len(got) != 2 {
		t.Errorf("saw %d, wanted the hand written one too", len(got))
	}
}

// A list somebody cannot cross a line out of is a list they stop keeping.
func TestForgetsOnePlay(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))
	_ = store.Add(one("b", now.Add(time.Minute)))

	got, _ := store.All()
	if err := store.Forget("youtube", "a", got[1].At); err != nil {
		t.Fatal(err)
	}

	left, _ := store.All()
	if len(left) != 1 || left[0].ID != "b" {
		t.Errorf("left %+v", left)
	}
}

// The same video watched twice is two lines, and crossing out one leaves the
// other: the moment is what tells them apart.
func TestForgetsOnlyTheOneAsked(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))
	_ = store.Stop(now.Add(time.Minute))
	_ = store.Add(one("a", now.Add(time.Hour)))

	_ = store.Forget("youtube", "a", now)

	left, _ := store.All()
	if len(left) != 1 {
		t.Errorf("left %d, wanted the second watch", len(left))
	}
}

func TestForgettingWhatIsNotThereChangesNothing(t *testing.T) {
	store := fresh(t)
	now := time.Now().UTC()

	_ = store.Add(one("a", now))

	if err := store.Forget("youtube", "nothing", now); err != nil {
		t.Fatal(err)
	}

	if left, _ := store.All(); len(left) != 1 {
		t.Errorf("left %d", len(left))
	}
}
