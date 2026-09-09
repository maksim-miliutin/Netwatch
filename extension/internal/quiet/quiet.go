// Package quiet holds the services somebody would rather friends did not see.
// What is hidden rather than what is shown, so a service added later shows.
package quiet

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type List struct {
	file string

	mu     sync.Mutex
	hidden map[string]bool
}

func Open(file string) (*List, error) {
	list := &List{file: file, hidden: map[string]bool{}}

	body, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return list, nil
	}

	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(body), "\n") {
		if name := strings.TrimSpace(line); name != "" {
			list.hidden[name] = true
		}
	}

	return list, nil
}

func (l *List) Hidden(service string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.hidden[service]
}

// The whole set at once: that is what a page of checkboxes knows.
func (l *List) Hide(services []string) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.hidden = map[string]bool{}

	for _, name := range services {
		if name = strings.TrimSpace(name); name != "" {
			l.hidden[name] = true
		}
	}

	if err := os.MkdirAll(filepath.Dir(l.file), 0o700); err != nil {
		return err
	}

	return os.WriteFile(l.file, []byte(strings.Join(l.Names(), "\n")+"\n"), 0o600)
}

func (l *List) Names() []string {
	names := make([]string, 0, len(l.hidden))

	for name := range l.hidden {
		names = append(names, name)
	}

	sort.Strings(names)

	return names
}
