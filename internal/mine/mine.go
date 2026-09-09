// Package mine keeps the services somebody added themselves.
//
// A line to a service: the address of the site, then the name to show. What is
// added here brings no rule for reading its addresses and needs none — the
// page says what is playing, and that is enough to write it down.
package mine

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

type List struct {
	file string

	mu    sync.Mutex
	hosts map[string]string
}

func Open(file string) (*List, error) {
	list := &List{file: file, hosts: map[string]string{}}

	body, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return list, nil
	}

	if err != nil {
		return nil, err
	}

	for _, line := range strings.Split(string(body), "\n") {
		host, shown, _ := strings.Cut(strings.TrimSpace(line), " ")

		if host = strings.TrimSpace(host); host != "" {
			list.hosts[host] = strings.TrimSpace(shown)
		}
	}

	return list, nil
}

func (l *List) Add(host, shown string) error {
	host = strings.TrimSpace(strings.ToLower(host))
	if host == "" {
		return nil
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	l.hosts[host] = strings.TrimSpace(shown)

	if err := os.MkdirAll(filepath.Dir(l.file), 0o700); err != nil {
		return err
	}

	lines := make([]string, 0, len(l.hosts))
	for host, shown := range l.hosts {
		lines = append(lines, strings.TrimSpace(host+" "+shown))
	}

	sort.Strings(lines)

	return os.WriteFile(l.file, []byte(strings.Join(lines, "\n")+"\n"), 0o600)
}

func (l *List) All() map[string]string {
	l.mu.Lock()
	defer l.mu.Unlock()

	kept := map[string]string{}
	for host, shown := range l.hosts {
		kept[host] = shown
	}

	return kept
}
