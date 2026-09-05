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

	// Shown is the name people read. Name stays lowercase and plain: it is a
	// key in a file and the name of a picture Discord looks up by it.
	Shown string

	// Heard is what somebody listens to rather than watches, which Discord
	// says out loud above the card.
	Heard bool

	// Watching answers what is being watched here, and nothing when the page
	// is not a watch at all: a search on YouTube is still YouTube.
	Watching func(*url.URL) string
}

var services = []Service{
	{
		Name:  "youtube",
		Shown: "YouTube",
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
		Shown: "RuTube",
		Hosts: []string{"rutube.ru", "www.rutube.ru"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "yandex-music",
		Shown: "Yandex Music",
		Heard: true,
		Hosts: []string{"music.yandex.ru", "music.yandex.com"},
		Watching: func(u *url.URL) string {
			// The album on its own is a page somebody browsed rather than heard.
			return after(u.Path, "/track/")
		},
	},
	{
		Name:  "vk-video",
		Shown: "VK Video",
		Hosts: []string{"vk.com", "vkvideo.ru", "m.vk.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video")
		},
	},
	{
		Name:  "twitch",
		Shown: "Twitch",
		Hosts: []string{"twitch.tv", "www.twitch.tv"},
		Watching: func(u *url.URL) string {
			if id := after(u.Path, "/videos/"); id != "" {
				return id
			}

			// A bare channel address is a live stream, named after the channel.
			return strings.Trim(u.Path, "/")
		},
	},
	{
		Name:  "dzen",
		Shown: "Dzen",
		Hosts: []string{"dzen.ru", "www.dzen.ru"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/watch/")
		},
	},
	{
		Name:  "ok-video",
		Shown: "OK Video",
		Hosts: []string{"ok.ru", "www.ok.ru", "m.ok.ru"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "vimeo",
		Shown: "Vimeo",
		Hosts: []string{"vimeo.com", "www.vimeo.com", "player.vimeo.com"},
		Watching: func(u *url.URL) string {
			return numbered(u.Path)
		},
	},
	{
		Name:  "dailymotion",
		Shown: "Dailymotion",
		Hosts: []string{"dailymotion.com", "www.dailymotion.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "coub",
		Shown: "Coub",
		Hosts: []string{"coub.com", "www.coub.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/view/")
		},
	},
	{
		Name:  "netflix",
		Shown: "Netflix",
		Hosts: []string{"netflix.com", "www.netflix.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/watch/")
		},
	},
	{
		Name:  "kinopoisk",
		Shown: "Kinopoisk",
		Hosts: []string{"hd.kinopoisk.ru", "kinopoisk.ru", "www.kinopoisk.ru"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/watch/", "/film/", "/series/")
		},
	},
	{
		Name:  "okko",
		Shown: "Okko",
		Hosts: []string{"okko.tv", "www.okko.tv"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/movie/", "/serial/", "/video/")
		},
	},
	{
		Name:  "ivi",
		Shown: "ivi",
		Hosts: []string{"ivi.ru", "www.ivi.ru"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/watch/")
		},
	},
	{
		Name:  "wink",
		Shown: "Wink",
		Hosts: []string{"wink.ru", "www.wink.ru"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/movies/", "/series/", "/channels/")
		},
	},
	{
		Name:  "premier",
		Shown: "Premier",
		Hosts: []string{"premier.one", "www.premier.one"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/show/", "/movie/")
		},
	},
	{
		Name:  "kick",
		Shown: "Kick",
		Hosts: []string{"kick.com", "www.kick.com"},
		Watching: func(u *url.URL) string {
			if id := after(u.Path, "/video/"); id != "" {
				return id
			}

			return strings.Trim(u.Path, "/")
		},
	},
	{
		Name:  "vkplay",
		Shown: "VK Play",
		Hosts: []string{"live.vkplay.ru", "vkplay.live"},
		Watching: func(u *url.URL) string {
			return strings.Trim(u.Path, "/")
		},
	},
}

// Shown hands back the name a service is read by, and the key itself for one
// this no longer knows: a play imported years ago still has to be drawn.
func Shown(name string) string {
	if service, ok := find(name); ok {
		return service.Shown
	}

	return name
}

func Heard(name string) bool {
	service, ok := find(name)

	return ok && service.Heard
}

func find(name string) (Service, bool) {
	for _, service := range services {
		if service.Name == name {
			return service, true
		}
	}

	return Service{}, false
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

// under is what follows any of these path parts, and nothing when the path is
// under none of them. A cinema names a film in the path rather than in a
// parameter, and the rest of the path does for an id: what is needed is
// telling one film from the next, not matching whatever shape the site uses
// this year.
func under(path string, parts ...string) string {
	for _, part := range parts {
		if strings.HasPrefix(path, part) {
			return strings.Trim(strings.TrimPrefix(path, part), "/")
		}
	}

	return ""
}

// numbered is the last part of a path when it is a number, which is the whole
// of how Vimeo names a film: vimeo.com/347119375.
func numbered(path string) string {
	last := path[strings.LastIndex(path, "/")+1:]

	if last == "" || strings.TrimLeft(last, "0123456789") != "" {
		return ""
	}

	return last
}
