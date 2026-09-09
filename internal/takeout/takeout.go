// Package takeout reads the history a service hands over: what was watched
// before the extension was installed.
package takeout

import (
	"encoding/json"
	"io"
	"strings"
	"time"

	"netwatch/internal/play"
)

type record struct {
	Header   string `json:"header"`
	Title    string `json:"title"`
	TitleURL string `json:"titleUrl"`
	Time     string `json:"time"`
}

// Read turns a watch history into plays. The count of skipped rows comes
// back too: a file that yields nothing should say so.
func Read(from io.Reader) (plays []play.Play, skipped int, err error) {
	var records []record

	if err := json.NewDecoder(from).Decode(&records); err != nil {
		return nil, 0, err
	}

	for _, one := range records {
		got, ok := asPlay(one)
		if !ok {
			skipped++

			continue
		}

		plays = append(plays, got)
	}

	return plays, skipped, nil
}

func asPlay(one record) (play.Play, bool) {
	if one.TitleURL == "" {
		return play.Play{}, false
	}

	at, err := time.Parse(time.RFC3339, one.Time)
	if err != nil {
		return play.Play{}, false
	}

	got, ok := play.Recognise(one.TitleURL, title(one.Title), at)
	if !ok {
		return play.Play{}, false
	}

	return got, true
}

// Google writes the title as "Watched " plus the name, in whatever language
// the account was set to. An unknown prefix is left alone rather than guessed at.
func title(said string) string {
	for _, prefix := range []string{"Watched ", "Смотрели ", "Вы смотрели "} {
		if strings.HasPrefix(said, prefix) {
			return strings.TrimPrefix(said, prefix)
		}
	}

	return said
}
