package discord

import (
	"strings"

	"netwatch/internal/now"
	"netwatch/internal/play"
)

// Discord cuts a longer line itself, mid-word and without saying it did.
const longest = 128

const paused = "Paused"

func Card(live now.Live) *Activity {
	kind := watching
	if play.Heard(live.Play.Service) {
		kind = listening
	}

	card := &Activity{
		Type:    kind,
		Details: line(named(live.Play)),
		State:   line(whose(live)),
		Assets: &Assets{
			Large:     live.Play.Service,
			LargeText: play.Shown(live.Play.Service),
		},
	}

	// A bar that stands still reads as a bar that is stuck.
	if live.Paused {
		card.State = paused
	} else {
		card.Timestamps = when(live)
	}

	if button, ok := open(live.Play.URL); ok {
		card.Buttons = []Button{button}
	}

	return card
}

func named(one play.Play) string {
	if one.Title != "" {
		return one.Title
	}

	return one.ID
}

func whose(live now.Live) string {
	if live.By != "" {
		return live.By
	}

	return play.Shown(live.Play.Service)
}

// With an end Discord counts down, without one it counts up — as a stream should.
func when(live now.Live) *Timestamps {
	if live.Started.IsZero() {
		return nil
	}

	stamps := &Timestamps{Start: live.Started.UnixMilli()}

	if !live.Ends.IsZero() {
		stamps.End = live.Ends.UnixMilli()
	}

	return stamps
}

// A button that points off the web takes the whole card down with it.
func open(address string) (Button, bool) {
	if !strings.HasPrefix(address, "https://") && !strings.HasPrefix(address, "http://") {
		return Button{}, false
	}

	return Button{Label: "Open", URL: address}, true
}

// Runes: a Russian title is two bytes a letter. One character is refused by
// Discord and takes the card with it, so a line that short is left out.
func line(text string) string {
	runes := []rune(strings.TrimSpace(text))

	if len(runes) < 2 {
		return ""
	}

	if len(runes) <= longest {
		return string(runes)
	}

	return strings.TrimSpace(string(runes[:longest-1])) + "…"
}
