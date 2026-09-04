// Package play knows what counts as watching something, and how to tell one
// service from another by the address alone.
package play

import (
	"net/url"
	"strings"
	"time"
)

// A Play is one thing watched or listened to, once.
type Play struct {
	Service string    `json:"service"`
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	URL     string    `json:"url"`
	At      time.Time `json:"at"`

	// How long it was open. Zero when nobody counted, which is most of the
	// time: a browser tab knows when it opened and rarely when it stopped.
	Seconds int `json:"seconds,omitempty"`
}

// A Service is one place things get watched. Adding another is one entry in
// the list below and nothing else: no switch to extend, no interface to
// implement, no registration to remember.
type Service struct {
	Name  string
	Hosts []string

	// Watching answers what is being watched at this address, or an empty
	// string when the page is not a thing being watched at all. A search
	// page on YouTube is still YouTube and still not a video.
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
			// The track sits at the end of an album path, and the album on
			// its own is a page somebody browsed rather than heard.
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

			// A bare channel address is a live stream, and the channel name
			// is the only name it has while it is running.
			return strings.Trim(u.Path, "/")
		},
	},
}

// Services lists what is recognised, for whoever asks what this knows about.
func Services() []string {
	names := make([]string, 0, len(services))

	for _, service := range services {
		names = append(names, service.Name)
	}

	return names
}

// Recognise reads a play out of an address, or reports that the address is
// not one of the things this watches.
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

// after returns what follows a marker in a path, up to the next slash.
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
