package discord

import (
	"bufio"
	"strings"
	"testing"
	"time"

	"netwatch/internal/now"
	"netwatch/internal/play"
)

const took = `{"cmd":"SET_ACTIVITY","data":{}}`

func following(t *testing.T, answers ...answer) (*follower, <-chan said) {
	t.Helper()

	pipe, heard := fake(append([]answer{{frame, ready}}, answers...)...)

	to, err := greet(pipe, "424242")
	if err != nil {
		t.Fatal(err)
	}

	<-heard

	return &follower{watching: now.New(), to: to}, heard
}

// The page says how far in it is, and that grows with the clock. A position
// standing still while time moves is a seek, not a video playing.
func watched(f *follower, id string, at time.Time, position time.Duration) {
	f.watching.Says(now.Said{
		Play:     play.Play{Service: "youtube", ID: id, Title: "Нечто " + id},
		Position: position,
		Length:   10 * time.Minute,
	}, at)
}

func TestSendsWhatIsPlaying(t *testing.T) {
	f, heard := following(t, answer{frame, took})
	at := time.Now()

	watched(f, "a", at, 0)
	f.update(at)

	if sent := string((<-heard).body); !strings.Contains(sent, "Нечто a") {
		t.Errorf("sent %s", sent)
	}
}

// The same video reported again every ten seconds is still the same card.
func TestDoesNotSendTheSameCardTwice(t *testing.T) {
	f, heard := following(t, answer{frame, took}, answer{frame, took})
	at := time.Now()

	watched(f, "a", at, 0)
	f.update(at)
	<-heard

	later := at.Add(Apart + time.Second)
	watched(f, "a", later, Apart+time.Second)
	f.update(later)

	select {
	case again := <-heard:
		t.Errorf("sent it again: %s", again.body)
	case <-time.After(100 * time.Millisecond):
	}
}

func TestWaitsBeforeSendingAnother(t *testing.T) {
	f, heard := following(t, answer{frame, took}, answer{frame, took})
	at := time.Now()

	watched(f, "a", at, 0)
	f.update(at)
	<-heard

	watched(f, "b", at.Add(time.Second), 0)
	f.update(at.Add(time.Second))

	select {
	case early := <-heard:
		t.Errorf("sent one a second later: %s", early.body)
	case <-time.After(100 * time.Millisecond):
	}
}

// Nothing playing takes the card down rather than leaving the last one up.
func TestClearsTheCardWhenNothingPlays(t *testing.T) {
	f, heard := following(t, answer{frame, took})

	f.update(time.Now())

	if sent := string((<-heard).body); !strings.Contains(sent, `"activity":null`) {
		t.Errorf("sent %s", sent)
	}
}

func TestSaysAThingOnlyOnce(t *testing.T) {
	var heard []string

	f := &follower{say: func(text string) { heard = append(heard, text) }}

	f.mention("одно")
	f.mention("одно")
	f.mention("другое")

	if len(heard) != 2 {
		t.Errorf("said %v", heard)
	}
}

// The point of a clock on the read: the loop lets go and tries again later,
// instead of holding a client that stopped answering until the program quits.
func TestLetsGoOfAClientThatStoppedAnswering(t *testing.T) {
	f := &follower{
		watching: now.New(),
		to:       sleeping(),
	}

	at := time.Now()
	watched(f, "a", at, 0)

	f.update(at)

	if f.to != nil {
		t.Error("still holding a client that says nothing")
	}

	if !f.again.After(at) {
		t.Error("did not put the next try off")
	}
}

// A service kept off the card looks, from Discord's side, like nothing playing.
func TestSendsNothingForAServiceKeptQuiet(t *testing.T) {
	f, heard := following(t, answer{frame, took})
	f.hidden = func(service string) bool { return service == "youtube" }

	at := time.Now()
	watched(f, "a", at, 0)

	f.update(at)

	if sent := string((<-heard).body); !strings.Contains(sent, `"activity":null`) {
		t.Errorf("sent %s for a service kept quiet", sent)
	}
}

func sleeping() *Presence {
	pipe := deaf()

	return &Presence{
		pipe:     pipe,
		patience: 50 * time.Millisecond,
		heard:    bufio.NewReader(pipe),
	}
}

// A Discord that is not there is ordinary rather than broken, and the loop has
// to wait rather than spend the evening dialling a socket nobody put there.
func TestWaitsBeforeLookingForAClientAgain(t *testing.T) {
	var said []string

	f := &follower{
		id:       "424242",
		watching: now.New(),
		say:      func(text string) { said = append(said, text) },
	}

	at := time.Now()

	f.turn(at)

	if f.to != nil {
		t.Fatal("found a client on a machine with none")
	}

	if !f.again.After(at) {
		t.Error("did not put the next try off")
	}

	// Turning again inside the wait must not go near the socket, and must not
	// say the same thing twice.
	f.turn(at.Add(time.Second))

	if len(said) != 1 {
		t.Errorf("said %v", said)
	}
}
