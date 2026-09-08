// Package store keeps the plays in a file, one to a line.
//
// A line at a time rather than a database: what is written is only added
// to, it all fits in memory, and a database would cost a dependency.
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
	// Opened for every write rather than held: this program spends its life
	// idle, and a handle held for hours outlives the disk it points at.
	file string

	// What was read last, and what the file looked like then. Nothing else
	// writes here, and noticing a hand edit costs one stat.
	plays []play.Play
	when  time.Time
	size  int64

	mu sync.Mutex
}

func Open(file string) (*Store, error) {
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		return nil, err
	}

	return &Store{file: file}, nil
}

// Add writes one play down unless the same one is already last, and closes
// the one before it: a play ends when the next begins.
func (s *Store) Add(one play.Play) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	already, err := s.read()
	if err != nil {
		return err
	}

	if len(already) > 0 {
		last := already[len(already)-1]
		// Still running counts as the same watch. One already closed does not:
		// coming back to a ten minute video an hour later is a second watch.
		if last.Service == one.Service && last.ID == one.ID && last.Seconds == 0 {
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

	if _, err := file.Write(append(line, '\n')); err != nil {
		return err
	}

	// What is kept was just brought up to date by close, if it ran at all.
	s.kept(append(s.plays, one))

	return nil
}

// Merge writes down plays from somewhere other than a browser tab, leaving
// out the ones already here: an import run twice should cost nothing.
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

// What makes two plays the same one: the thing watched and the moment.
func key(one play.Play) string {
	return one.Service + "\x00" + one.ID + "\x00" + one.At.UTC().Format(time.RFC3339)
}

func (s *Store) All() ([]play.Play, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	plays, err := s.read()
	if err != nil {
		return nil, err
	}

	// Sorted on a copy. In place it would turn what is kept back to front, and
	// the next Add would take the oldest play for the newest.
	newest := make([]play.Play, len(plays))
	copy(newest, plays)

	sort.Slice(newest, func(a, b int) bool {
		return newest[a].At.After(newest[b].At)
	})

	return newest, nil
}

// A list somebody cannot cross a line out of is a list they stop keeping.
func (s *Store) Forget(service, id string, at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plays, err := s.read()
	if err != nil {
		return err
	}

	left := make([]play.Play, 0, len(plays))

	for _, one := range plays {
		if one.Service == service && one.ID == id && one.At.Equal(at) {
			continue
		}

		left = append(left, one)
	}

	if len(left) == len(plays) {
		return nil
	}

	return s.rewrite(left)
}

// Stop closes what is open because something said it ended rather than because
// the next play arrived. A page that is left knows the moment it was left.
func (s *Store) Stop(at time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	plays, err := s.read()
	if err != nil || len(plays) == 0 {
		return err
	}

	return s.close(plays[len(plays)-1], at)
}

// Ends is how long a play may be counted for. A tab left open overnight was
// not nine hours of watching, and a list that says so is worse than silence.
const Ends = 3 * time.Hour

// close writes down how long the play before this one lasted. The file is
// rewritten whole: a few thousand lines, and seeking into one is not simpler.
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

	// On a copy: a rewrite that fails would otherwise leave what is kept
	// saying closed while the file still says open.
	closed := make([]play.Play, len(plays))
	copy(closed, plays)

	for i := range closed {
		if closed[i].At.Equal(last.At) && closed[i].ID == last.ID {
			closed[i].Seconds = int(lasted.Seconds())
		}
	}

	return s.rewrite(closed)
}

func (s *Store) rewrite(plays []play.Play) error {
	// Written beside and moved into place: half a file costs everything.
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

	if err := os.Rename(temporary, s.file); err != nil {
		return err
	}

	s.kept(plays)

	return nil
}

func (s *Store) read() ([]play.Play, error) {
	stat, err := os.Stat(s.file)
	if os.IsNotExist(err) {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	if s.plays != nil && stat.ModTime().Equal(s.when) && stat.Size() == s.size {
		return s.plays, nil
	}

	file, err := os.Open(s.file)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var plays []play.Play

	lines := bufio.NewScanner(file)
	for lines.Scan() {
		var one play.Play

		// One bad line should not cost somebody the rest of their history.
		if err := json.Unmarshal(lines.Bytes(), &one); err != nil {
			continue
		}

		plays = append(plays, one)
	}

	if err := lines.Err(); err != nil {
		return nil, err
	}

	s.plays, s.when, s.size = plays, stat.ModTime(), stat.Size()

	return plays, nil
}

// kept remembers what was just written, so that the next read does not go back
// to the disk for a file this only just finished with.
func (s *Store) kept(plays []play.Play) {
	stat, err := os.Stat(s.file)
	if err != nil {
		s.plays = nil

		return
	}

	s.plays, s.when, s.size = plays, stat.ModTime(), stat.Size()
}
