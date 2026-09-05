// Package now holds what is playing at this moment, which a list of things
// that already happened cannot answer.
package now

import (
	"sync"
	"time"

	"netwatch/internal/play"
)

// A closed browser says nothing at all, which is what a very long film says.
const Quiet = 35 * time.Second

// Less than this is a report arriving late; more is somebody dragging the bar.
const Drift = 3 * time.Second

type Said struct {
	Play     play.Play
	By       string
	Paused   bool
	Position time.Duration

	// Zero when the page did not say, which is what a live stream looks like.
	Length time.Duration
}

// A Live is told the way a bar wants to hear it: when this began, not how far in.
type Live struct {
	Play   play.Play
	By     string
	Paused bool

	Started time.Time
	Ends    time.Time

	// The start of a paused video walks forward a second every second.
	Position time.Duration
}

type Watch struct {
	mu   sync.Mutex
	live Live
	told time.Time
	on   bool
}

func New() *Watch {
	return &Watch{}
}

func (w *Watch) Says(said Said, at time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	live := Live{
		Play:     said.Play,
		By:       said.By,
		Paused:   said.Paused,
		Started:  at.Add(-said.Position),
		Position: said.Position,
	}

	if said.Length > 0 {
		live.Ends = live.Started.Add(said.Length)
	}

	if w.on && steady(w.live, live) {
		live.Started = w.live.Started
		live.Ends = w.live.Ends
	}

	w.live = live
	w.told = at
	w.on = true
}

// Nothing is a fact rather than a timeout, and does not wait out the quiet.
func (w *Watch) Nothing() {
	w.mu.Lock()
	defer w.mu.Unlock()

	w.live = Live{}
	w.on = false
}

func (w *Watch) Playing(at time.Time) (Live, bool) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.on || at.Sub(w.told) > Quiet {
		return Live{}, false
	}

	return w.live, true
}

func steady(was, is Live) bool {
	if was.Play.Service != is.Play.Service || was.Play.ID != is.Play.ID {
		return false
	}

	moved := was.Started.Sub(is.Started)
	if moved < 0 {
		moved = -moved
	}

	return moved <= Drift
}
