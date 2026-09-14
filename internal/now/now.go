// Package now holds what is playing at this moment, which a list of things
// that already happened cannot answer.
package now

import (
	"sync"
	"time"

	"netwatch/internal/play"
)

const Quiet = 35 * time.Second

const Drift = 3 * time.Second

type Said struct {
	Play     play.Play
	By       string
	Paused   bool
	Position time.Duration

	Length time.Duration
}

type Live struct {
	Play   play.Play
	By     string
	Paused bool

	Started time.Time
	Ends    time.Time

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

// Longest a play can be before the page saying it is not to be believed.
const Most = 24 * time.Hour

func (w *Watch) Says(said Said, at time.Time) {
	w.mu.Lock()
	defer w.mu.Unlock()

	position, length := sane(said.Position, said.Length)

	live := Live{
		Play:     said.Play,
		By:       said.By,
		Paused:   said.Paused,
		Started:  at.Add(-position),
		Position: position,
	}

	if length > 0 {
		live.Ends = live.Started.Add(length)
	}

	if w.on && steady(w.live, live) {
		live.Started = w.live.Started
		live.Ends = w.live.Ends
	}

	w.live = live
	w.told = at
	w.on = true
}

// A page can say anything: a position past the end, a position before the
// start, a length of a century. Those go on the card as a bar that has already
// run out or has not begun.
func sane(position, length time.Duration) (time.Duration, time.Duration) {
	if length < 0 || length > Most {
		length = 0
	}

	if position < 0 || position > Most {
		position = 0
	}

	if length > 0 && position > length {
		position = length
	}

	return position, length
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
