package sum

import (
	"testing"
	"time"

	"netwatch/internal/play"
)

var now = time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)

func at(service string, ago time.Duration, seconds int) play.Play {
	return play.Play{Service: service, ID: service, At: now.Add(-ago), Seconds: seconds}
}

func TestAddsUpWhatWasWatched(t *testing.T) {
	total := Week([]play.Play{
		at("youtube", time.Hour, 600),
		at("youtube", 2*time.Hour, 300),
		at("rutube", 3*time.Hour, 120),
	}, now)

	if total.Plays != 3 || total.Seconds != 1020 {
		t.Errorf("got %d plays and %d seconds", total.Plays, total.Seconds)
	}
}

func TestPutsTheHeaviestServiceFirst(t *testing.T) {
	total := Week([]play.Play{
		at("rutube", time.Hour, 100),
		at("youtube", time.Hour, 900),
	}, now)

	if total.Services[0].Name != "youtube" {
		t.Errorf("put %s first", total.Services[0].Name)
	}
}

// A week of short things nobody stayed for still says something.
func TestBreaksATieAtNothingByCount(t *testing.T) {
	total := Week([]play.Play{
		at("rutube", time.Hour, 0),
		at("youtube", time.Hour, 0),
		at("youtube", 2*time.Hour, 0),
	}, now)

	if total.Services[0].Name != "youtube" {
		t.Errorf("put %s first", total.Services[0].Name)
	}
}

func TestLeavesOutWhatIsOlderThanTheSpan(t *testing.T) {
	total := Week([]play.Play{
		at("youtube", time.Hour, 60),
		at("youtube", 30*24*time.Hour, 6000),
	}, now)

	if total.Plays != 1 || total.Seconds != 60 {
		t.Errorf("got %d plays and %d seconds", total.Plays, total.Seconds)
	}
}

// Nobody should mistake a small total for a quiet week: a play still open
// counts for nothing until the next thing is watched.
func TestSaysHowManyWereNeverClosed(t *testing.T) {
	total := Week([]play.Play{
		at("youtube", time.Hour, 0),
		at("youtube", 2*time.Hour, 300),
	}, now)

	if total.Open != 1 {
		t.Errorf("said %d open, wanted one", total.Open)
	}
}

func TestAnswersNothingForNothing(t *testing.T) {
	total := Week(nil, now)

	if total.Plays != 0 || len(total.Services) != 0 {
		t.Errorf("got %+v", total)
	}
}
