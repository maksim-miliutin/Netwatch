// Package sum answers what a list of plays adds up to: how much, and where
// it went.
package sum

import (
	"sort"
	"time"

	"netwatch/internal/play"
)

type Service struct {
	Name    string `json:"name"`
	Plays   int    `json:"plays"`
	Seconds int    `json:"seconds"`
}

type Total struct {
	Since    time.Time `json:"since"`
	Plays    int       `json:"plays"`
	Seconds  int       `json:"seconds"`
	Services []Service `json:"services"`

	// How many were never closed, so a small total is not read as a quiet week.
	Open int `json:"open"`
}

func Over(plays []play.Play, since time.Time) Total {
	total := Total{Since: since}
	byName := map[string]*Service{}

	for _, one := range plays {
		if one.At.Before(since) {
			continue
		}

		total.Plays++
		total.Seconds += one.Seconds

		if one.Seconds == 0 {
			total.Open++
		}

		found, ok := byName[one.Service]
		if !ok {
			found = &Service{Name: one.Service}
			byName[one.Service] = found
		}

		found.Plays++
		found.Seconds += one.Seconds
	}

	for _, service := range byName {
		total.Services = append(total.Services, *service)
	}

	// By time first, by count when two services tie at nothing.
	sort.Slice(total.Services, func(a, b int) bool {
		if total.Services[a].Seconds != total.Services[b].Seconds {
			return total.Services[a].Seconds > total.Services[b].Seconds
		}

		return total.Services[a].Plays > total.Services[b].Plays
	})

	return total
}

// Week is the span most people mean when they ask where the time went.
func Week(plays []play.Play, now time.Time) Total {
	return Over(plays, now.AddDate(0, 0, -7))
}

// Month is the other one they mean, and a calendar month rather than thirty
// days: a month is a thing on a wall, not a number of turns of the earth.
func Month(plays []play.Play, now time.Time) Total {
	return Over(plays, now.AddDate(0, -1, 0))
}
