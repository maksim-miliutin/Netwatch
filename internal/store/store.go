// Package store keeps the plays in a file, one to a line.
//
// A line at a time rather than a database: what is written is only ever added
// to, the whole of it fits in memory, and anybody who wants to read it can do
// so with any tool that reads text. A database would buy nothing here and
// would cost a dependency on the day this has to build on a machine that has
// none.
package store

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"netwatch/internal/play"
)

type Store struct {
	// The file is opened for every write rather than held open: this program
	// spends its life idle, and a handle held for hours is a handle that
	// outlives the disk it points at.
	file string

	mu sync.Mutex
}

func Open(file string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return nil, err
	}

	return &Store{file: file}, nil
}

// Add writes one play down, unless the same one is already the last thing
// written. A tab left open reports itself again on every check, and the same
// video twice in a row is one watch rather than two.
//
// It also closes the one before it: a play ends when the next one starts, and
// that is the only end most of them get. A tab closed without another opening
// stays open in the list until something else is watched, which is the honest
// answer rather than a guess.
func (s *Store) Add(one play.Play) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	already, err := s.read()
	if err != nil {
		return err
	}

	if len(already) > 0 {
		last := already[len(already)-1]
		if last.Service == one.Service && last.ID == one.ID {
			return nil
		}

		if err := s.close(last, one.At); err != nil {
			return err
		}
	}

	line, err := json.Marshal(one)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(s.file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	defer file.Close()

	_, err = file.Write(append(line, '\n'))

	return err
}

// Merge writes down plays that came from somewhere other than a browser tab,
// leaving out the ones already here. An import run twice should cost nothing:
// somebody who is not sure whether it worked will run it again.
func (s *Store) Merge(incoming []play.Play) (added int, err error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	already, err := s.read()
	if err != nil {
		return 0, err
	}

	seen := map[string]bool{}
	for _, one := range already {
		seen[key(one)] = true
	}

	for _, one := range incoming {
		if seen[key(one)] {
			continue
		}

		seen[key(one)] = true
		already = append(already, one)
		added++
	}

	if added == 0 {
		return 0, nil
	}

	sort.Slice(already, func(a, b int) bool {
		return already[a].At.Before(already[b].At)
	})

	return added, s.rewrite(already)
}

// What makes two plays the same one: the thing watched and the moment. The
// same video watched twice is two plays, and the same row imported twice is
// one.
func key(one play.Play) string {
	return one.Service + "\x00" + one.ID + "\x00" + one.At.UTC().Format(time.RFC3339)
}

// All hands back everything written, newest first.
func (s *Store) All() ([]play.Play, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	plays, err := s.read()
	if err != nil {
		return nil, err
	}

	sort.Slice(plays, func(a, b int) bool {
		return plays[a].At.After(plays[b].At)
	})

	return plays, nil
}

// Ends is how long a play may be counted for. Somebody who leaves a tab open
// overnight did not watch for nine hours, and a list that says they did is
// worse than one that says nothing.
const Ends = 3 * time.Hour

// close writes down how long the play before this one lasted. The whole file
// is rewritten: it is a few thousand lines at most, and rewriting it whole is
// simpler than seeking into a line and hoping the new one is the same length.
func (s *Store) close(last play.Play, at time.Time) error {
	if last.Seconds != 0 {
		return nil
	}

	lasted := at.Sub(last.At)
	if lasted <= 0 || lasted > Ends {
		return nil
	}

	plays, err := s.read()
	if err != nil {
		return err
	}

	for i := range plays {
		if plays[i].At.Equal(last.At) && plays[i].ID == last.ID {
			plays[i].Seconds = int(lasted.Seconds())
		}
	}

	return s.rewrite(plays)
}

func (s *Store) rewrite(plays []play.Play) error {
	// Written beside and moved into place: a program stopped halfway through
	// this should cost nothing, and half a file costs everything.
	temporary := s.file + ".new"

	file, err := os.OpenFile(temporary, os.O_TRUNC|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}

	writer := bufio.NewWriter(file)

	for _, one := range plays {
		line, err := json.Marshal(one)
		if err != nil {
			file.Close()

			return err
		}

		if _, err := writer.Write(append(line, '\n')); err != nil {
			file.Close()

			return err
		}
	}

	if err := writer.Flush(); err != nil {
		file.Close()

		return err
	}

	if err := file.Close(); err != nil {
		return err
	}

	return os.Rename(temporary, s.file)
}

func (s *Store) read() ([]play.Play, error) {
	file, err := os.Open(s.file)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}
	defer file.Close()

	var plays []play.Play

	lines := bufio.NewScanner(file)
	for lines.Scan() {
		var one play.Play

		// A line that will not read is skipped rather than fatal: one bad
		// line should not cost somebody the rest of their history.
		if err := json.Unmarshal(lines.Bytes(), &one); err != nil {
			continue
		}

		plays = append(plays, one)
	}

	return plays, lines.Err()
}
