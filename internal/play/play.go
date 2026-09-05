// Package play tells one service from another by the address alone.
package play

import (
	"net/url"
	"strings"
	"time"
)

type Play struct {
	Service string    `json:"service"`
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	URL     string    `json:"url"`
	At      time.Time `json:"at"`

	// Zero when nobody counted: a tab knows when it opened, rarely when it
	// stopped.
	Seconds int `json:"seconds,omitempty"`
}

// A Service is one place things get watched. Adding another is one entry in
// the list below: no switch to extend, no interface, no registration.
type Service struct {
	Name  string
	Hosts []string

	// Watching answers what is being watched here, and nothing when the page
	// is not a watch at all: a search on YouTube is still YouTube.
	Watching func(*url.URL) string
}

var services = []Service{
	{
		Name:  "youtube",
		Hosts: []string{"youtube.com", "www.youtube.com", "m.youtube.com", "youtu.be"},
		Watching: func(u *url.URL) string {
			if u.Host == "youtu.be" {
				return strings.TrimPrefix(u.Path, "/")
			}

			return u.Query().Get("v")
		},
	},
	{
		Name:  "rutube",
		Hosts: []string{"rutube.ru", "www.rutube.ru"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "yandex-music",
		Hosts: []string{"music.yandex.ru", "music.yandex.com"},
		Watching: func(u *url.URL) string {
			// The album on its own is a page somebody browsed rather than heard.
			return after(u.Path, "/track/")
		},
	},
	{
		Name:  "vk-video",
		Hosts: []string{"vk.com", "vkvideo.ru", "m.vk.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video")
		},
	},
	{
		Name:  "twitch",
		Hosts: []string{"twitch.tv", "www.twitch.tv"},
		Watching: func(u *url.URL) string {
			if id := after(u.Path, "/videos/"); id != "" {
				return id
			}

			// A bare channel address is a live stream, named after the channel.
			return strings.Trim(u.Path, "/")
		},
	},
}

func Services() []string {
	names := make([]string, 0, len(services))

	for _, service := range services {
		names = append(names, service.Name)
	}

	return names
}

func Recognise(address, title string, at time.Time) (Play, bool) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" {
		return Play{}, false
	}

	for _, service := range services {
		if !holds(service.Hosts, parsed.Host) {
			continue
		}

		id := service.Watching(parsed)
		if id == "" {
			return Play{}, false
		}

		return Play{
			Service: service.Name,
			ID:      id,
			Title:   strings.TrimSpace(title),
			URL:     address,
			At:      at.UTC(),
		}, true
	}

	return Play{}, false
}

func holds(hosts []string, host string) bool {
	for _, one := range hosts {
		if one == host {
			return true
		}
	}

	return false
}

func after(path, marker string) string {
	at := strings.Index(path, marker)
	if at == -1 {
		return ""
	}

	rest := path[at+len(marker):]
	if slash := strings.Index(rest, "/"); slash != -1 {
		rest = rest[:slash]
	}

	return rest
}
