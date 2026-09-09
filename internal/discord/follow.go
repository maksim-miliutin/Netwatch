package discord

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"time"

	"netwatch/internal/now"
)

const (
	Beat  = time.Second
	Apart = 5 * time.Second
)

const Again = 10 * time.Second

func Follow(ctx context.Context, id string, watching *now.Watch,
	hidden func(string) bool, say func(string)) {
	beat := time.NewTicker(Beat)
	defer beat.Stop()

	f := &follower{id: id, watching: watching, hidden: hidden, say: say}

	defer f.drop()

	for {
		select {
		case <-ctx.Done():
			return

		case at := <-beat.C:
			f.turn(at)
		}
	}
}

type follower struct {
	id       string
	watching *now.Watch
	hidden   func(string) bool
	say      func(string)

	to    *Presence
	shown []byte
	sent  time.Time
	again time.Time
	said  string
}

func (f *follower) turn(at time.Time) {
	if f.to == nil {
		if at.Before(f.again) {
			return
		}

		f.connect(at)

		return
	}

	f.update(at)
}

func (f *follower) connect(at time.Time) {
	to, err := Open(f.id)
	if err != nil {
		f.again = at.Add(Again)

		if errors.Is(err, ErrNoClient) {
			f.mention("No Discord here yet. The card goes up when one opens.")

			return
		}

		f.mention(err.Error() + ". Application ID from discord.com/developers/applications")

		return
	}

	f.to = to
	f.shown = nil

	f.mention("Discord is listening. What plays goes on the card.")
}

func (f *follower) update(at time.Time) {
	var card *Activity

	if live, ok := f.watching.Playing(at); ok && !f.quiet(live.Play.Service) {
		card = Card(live)
	}

	next, err := json.Marshal(card)
	if err != nil || bytes.Equal(next, f.shown) {
		return
	}

	if at.Sub(f.sent) < Apart {
		return
	}

	if err := f.to.Show(card); err != nil {
		f.mention(err.Error())
		f.drop()
		f.again = at.Add(Again)

		return
	}

	f.shown = next
	f.sent = at
}

func (f *follower) quiet(service string) bool {
	return f.hidden != nil && f.hidden(service)
}

func (f *follower) drop() {
	if f.to == nil {
		return
	}

	f.to.Close()
	f.to = nil
	f.shown = nil
}

func (f *follower) mention(text string) {
	if text == f.said || f.say == nil {
		return
	}

	f.said = text
	f.say(text)
}
