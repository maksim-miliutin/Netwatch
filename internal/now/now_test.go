package now

import (
	"testing"
	"time"

	"netwatch/internal/play"
)

func said(id string, position time.Duration) Said {
	return Said{
		Play:     play.Play{Service: "youtube", ID: id, Title: id},
		Position: position,
		Length:   10 * time.Minute,
	}
}

func TestHoldsWhatThePageSaid(t *testing.T) {
	watch := New()
	at := time.Now()

	watch.Says(said("a", time.Minute), at)

	live, ok := watch.Playing(at)
	if !ok || live.Play.ID != "a" {
		t.Fatalf("got %+v %v", live, ok)
	}

	if want := at.Add(-time.Minute); !live.Started.Equal(want) {
		t.Errorf("started %s, wanted %s", live.Started, want)
	}
}

func TestHasNothingBeforeAnybodySaysAnything(t *testing.T) {
	if _, ok := New().Playing(time.Now()); ok {
		t.Error("something is playing on a machine nobody touched")
	}
}

// A browser that was closed says nothing at all, which is what a very long
// film also says.
func TestForgetsAPageThatWentQuiet(t *testing.T) {
	watch := New()
	at := time.Now()

	watch.Says(said("a", 0), at)

	if _, ok := watch.Playing(at.Add(Quiet + time.Second)); ok {
		t.Error("still playing after nobody said so for a while")
	}
}

func TestStopsWhenTheTabSaysSo(t *testing.T) {
	watch := New()
	at := time.Now()

	watch.Says(said("a", 0), at)
	watch.Nothing()

	if _, ok := watch.Playing(at); ok {
		t.Error("still playing after the page said it was gone")
	}
}

// Two reports a moment apart give two starts a moment apart. Taking the
// second one would shift the bar every ten seconds.
func TestKeepsTheStartStillWhileItPlays(t *testing.T) {
	watch := New()
	at := time.Now()

	watch.Says(said("a", time.Minute), at)
	first, _ := watch.Playing(at)

	later := at.Add(10 * time.Second)
	watch.Says(said("a", 70*time.Second+time.Second), later)

	live, _ := watch.Playing(later)
	if !live.Started.Equal(first.Started) {
		t.Errorf("start moved to %s from %s", live.Started, first.Started)
	}
}

// Dragging the bar is not drift, and the card should follow it.
func TestFollowsAJumpToAnotherPlace(t *testing.T) {
	watch := New()
	at := time.Now()

	watch.Says(said("a", time.Minute), at)
	first, _ := watch.Playing(at)

	watch.Says(said("a", 8*time.Minute), at)

	live, _ := watch.Playing(at)
	if live.Started.Equal(first.Started) {
		t.Error("start stayed where it was after a seek")
	}
}

func TestLeavesTheEndOpenWhenNobodySaidHowLong(t *testing.T) {
	watch := New()
	at := time.Now()

	watch.Says(Said{Play: play.Play{Service: "twitch", ID: "someone"}}, at)

	live, _ := watch.Playing(at)
	if !live.Ends.IsZero() {
		t.Errorf("a stream ends at %s", live.Ends)
	}
}
