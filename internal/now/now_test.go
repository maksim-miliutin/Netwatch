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

// A page can say anything, and what it says goes on a card other people read.
func TestDoesNotBelieveNonsenseAboutTime(t *testing.T) {
	at := time.Now()

	for _, said := range []struct {
		what             string
		position, length time.Duration
		started          time.Duration
		ends             time.Duration
		hasEnd           bool
	}{
		// The end is now rather than six minutes ago.
		{"past the end", 500 * time.Second, 100 * time.Second, 100 * time.Second, 0, true},
		{"before the start", -50 * time.Second, 100 * time.Second, 0, 100 * time.Second, true},
		{"a century long", 1 << 62, 1 << 62, 0, 0, false},
		{"a length of nothing", 30 * time.Second, 0, 30 * time.Second, 0, false},
	} {
		watch := New()
		watch.Says(Said{
			Play:     play.Play{Service: "youtube", ID: "a"},
			Position: said.position,
			Length:   said.length,
		}, at)

		live, _ := watch.Playing(at)

		if got := at.Sub(live.Started); got != said.started {
			t.Errorf("%s: started %v ago, wanted %v", said.what, got, said.started)
		}

		if live.Ends.IsZero() == said.hasEnd {
			t.Errorf("%s: end is %v", said.what, live.Ends)

			continue
		}

		if said.hasEnd {
			if got := live.Ends.Sub(at); got != said.ends {
				t.Errorf("%s: ends in %v, wanted %v", said.what, got, said.ends)
			}
		}
	}
}

// Whatever else, the end never comes before the moment it is read.
func TestNeverEndsInThePast(t *testing.T) {
	at := time.Now()

	for _, position := range []time.Duration{0, 50, 99, 100, 101, 5000} {
		watch := New()
		watch.Says(Said{
			Play:     play.Play{Service: "youtube", ID: "a"},
			Position: position * time.Second,
			Length:   100 * time.Second,
		}, at)

		if live, _ := watch.Playing(at); !live.Ends.IsZero() && live.Ends.Before(at) {
			t.Errorf("at %v the end is %v ago", position, at.Sub(live.Ends))
		}
	}
}
