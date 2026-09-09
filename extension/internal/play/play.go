// Package play tells one service from another by the address alone.
package play

import (
	"net/url"
	"regexp"
	"strings"
	"sync"
	"time"
)

type Play struct {
	Service string    `json:"service"`
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	URL     string    `json:"url"`
	At      time.Time `json:"at"`

	Seconds int `json:"seconds,omitempty"`
}

type Service struct {
	Name  string
	Hosts []string

	Shown string

	Heard bool

	Kind string

	Unnamed bool

	Watching func(*url.URL) string
}

var services = []Service{
	{
		Name:  "youtube",
		Kind:  "Video",
		Shown: "YouTube",
		Hosts: []string{"youtube.com", "www.youtube.com", "m.youtube.com", "youtu.be"},
		Watching: func(u *url.URL) string {
			if u.Host == "youtu.be" {
				return strings.TrimPrefix(u.Path, "/")
			}

			if id := piece(u.Path, "/shorts/", "/live/", "/embed/"); id != "" {
				return id
			}

			return u.Query().Get("v")
		},
	},
	{
		Name:  "rutube",
		Kind:  "Video",
		Shown: "RuTube",
		Hosts: []string{"rutube.ru", "www.rutube.ru"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/video/", "/shorts/", "/live/")
		},
	},
	{
		Name:    "yandex-music",
		Kind:    "Music",
		Shown:   "Yandex Music",
		Heard:   true,
		Unnamed: true,
		Hosts:   []string{"music.yandex.ru", "music.yandex.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/track/")
		},
	},
	{
		Name:  "vk-video",
		Kind:  "Video",
		Shown: "VK Video",
		Hosts: []string{"vk.com", "vkvideo.ru", "m.vk.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/video", "/clip")
		},
	},
	{
		Name:  "twitch",
		Kind:  "Streams",
		Shown: "Twitch",
		Hosts: []string{"twitch.tv", "www.twitch.tv", "clips.twitch.tv"},
		Watching: func(u *url.URL) string {
			if id := after(u.Path, "/videos/"); id != "" {
				return id
			}

			return strings.Trim(u.Path, "/")
		},
	},
	{
		Name:  "dzen",
		Kind:  "Video",
		Shown: "Dzen",
		Hosts: []string{"dzen.ru", "www.dzen.ru"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/video/watch/", "/shorts/")
		},
	},
	{
		Name:  "ok-video",
		Kind:  "Video",
		Shown: "OK Video",
		Hosts: []string{"ok.ru", "www.ok.ru", "m.ok.ru"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/video/", "/live/")
		},
	},
	{
		Name:  "vimeo",
		Kind:  "Video",
		Shown: "Vimeo",
		Hosts: []string{"vimeo.com", "www.vimeo.com", "player.vimeo.com"},
		Watching: func(u *url.URL) string {
			return numbered(u.Path)
		},
	},
	{
		Name:  "dailymotion",
		Kind:  "Video",
		Shown: "Dailymotion",
		Hosts: []string{"dailymotion.com", "www.dailymotion.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "coub",
		Kind:  "Video",
		Shown: "Coub",
		Hosts: []string{"coub.com", "www.coub.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/view/")
		},
	},
	{
		Name:  "netflix",
		Kind:  "Films",
		Shown: "Netflix",
		Hosts: []string{"netflix.com", "www.netflix.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/watch/")
		},
	},
	{
		Name:  "kinopoisk",
		Kind:  "Films",
		Shown: "Kinopoisk",
		Hosts: []string{"hd.kinopoisk.ru", "kinopoisk.ru", "www.kinopoisk.ru"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/watch/", "/film/", "/series/")
		},
	},
	{
		Name:  "okko",
		Kind:  "Films",
		Shown: "Okko",
		Hosts: []string{"okko.tv", "www.okko.tv"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/movie/", "/serial/", "/video/")
		},
	},
	{
		Name:  "ivi",
		Kind:  "Films",
		Shown: "ivi",
		Hosts: []string{"ivi.ru", "www.ivi.ru"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/watch/")
		},
	},
	{
		Name:  "wink",
		Kind:  "Films",
		Shown: "Wink",
		Hosts: []string{"wink.ru", "www.wink.ru"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/movies/", "/series/", "/channels/")
		},
	},
	{
		Name:  "premier",
		Kind:  "Films",
		Shown: "Premier",
		Hosts: []string{"premier.one", "www.premier.one"},
		Watching: func(u *url.URL) string {
			return under(u.Path, "/show/", "/movie/")
		},
	},
	{
		Name:  "kick",
		Kind:  "Streams",
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
		Kind:  "Streams",
		Shown: "VK Play",
		Hosts: []string{"live.vkplay.ru", "vkplay.live"},
		Watching: func(u *url.URL) string {
			return strings.Trim(u.Path, "/")
		},
	},
	{
		Name:  "youtube-music",
		Kind:  "Music",
		Shown: "YouTube Music",
		Heard: true,
		Hosts: []string{"music.youtube.com"},
		Watching: func(u *url.URL) string {
			return u.Query().Get("v")
		},
	},
	{
		Name:  "spotify",
		Kind:  "Music",
		Shown: "Spotify",
		Heard: true,
		Hosts: []string{"open.spotify.com", "play.spotify.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/track/", "/album/", "/playlist/", "/episode/", "/show/")
		},
	},
	{
		Name:  "soundcloud",
		Kind:  "Music",
		Shown: "SoundCloud",
		Heard: true,
		Hosts: []string{"soundcloud.com", "m.soundcloud.com", "on.soundcloud.com"},
		Watching: func(u *url.URL) string {
			return deep(u.Path, 2)
		},
	},
	{
		Name:  "apple-music",
		Kind:  "Music",
		Shown: "Apple Music",
		Heard: true,
		Hosts: []string{"music.apple.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/album/", "/playlist/", "/song/")
		},
	},
	{
		Name:  "deezer",
		Kind:  "Music",
		Shown: "Deezer",
		Heard: true,
		Hosts: []string{"deezer.com", "www.deezer.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/track/", "/album/", "/playlist/", "/episode/")
		},
	},
	{
		Name:  "zvuk",
		Kind:  "Music",
		Shown: "Zvuk",
		Heard: true,
		Hosts: []string{"zvuk.com", "www.zvuk.com", "sber-zvuk.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/track/", "/release/", "/playlist/")
		},
	},
	{
		Name:  "mixcloud",
		Kind:  "Music",
		Shown: "Mixcloud",
		Heard: true,
		Hosts: []string{"mixcloud.com", "www.mixcloud.com"},
		Watching: func(u *url.URL) string {
			return deep(u.Path, 2)
		},
	},
	{
		Name:  "tiktok",
		Kind:  "Video",
		Shown: "TikTok",
		Hosts: []string{"tiktok.com", "www.tiktok.com", "vm.tiktok.com"},
		Watching: func(u *url.URL) string {
			if u.Host == "vm.tiktok.com" {
				return strings.Trim(u.Path, "/")
			}

			return piece(u.Path, "/video/", "/t/")
		},
	},
	{
		Name:  "bilibili",
		Kind:  "Video",
		Shown: "Bilibili",
		Hosts: []string{"bilibili.com", "www.bilibili.com", "m.bilibili.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "crunchyroll",
		Kind:  "Films",
		Shown: "Crunchyroll",
		Hosts: []string{"crunchyroll.com", "www.crunchyroll.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/watch/")
		},
	},
	{
		Name:  "disney-plus",
		Kind:  "Films",
		Shown: "Disney+",
		Hosts: []string{"disneyplus.com", "www.disneyplus.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "max",
		Kind:  "Films",
		Shown: "Max",
		Hosts: []string{"max.com", "www.max.com", "play.max.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/video/", "/movie/", "/show/")
		},
	},
	{
		Name:  "prime-video",
		Kind:  "Films",
		Shown: "Prime Video",
		Hosts: []string{"primevideo.com", "www.primevideo.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/detail/", "/watch/")
		},
	},
	{
		Name:  "apple-tv",
		Kind:  "Films",
		Shown: "Apple TV+",
		Hosts: []string{"tv.apple.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/episode/", "/movie/", "/show/")
		},
	},
	{
		Name:  "hulu",
		Kind:  "Films",
		Shown: "Hulu",
		Hosts: []string{"hulu.com", "www.hulu.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/watch/")
		},
	},
	{
		Name:  "nebula",
		Kind:  "Video",
		Shown: "Nebula",
		Hosts: []string{"nebula.tv", "www.nebula.tv"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/videos/")
		},
	},
	{
		Name:  "odysee",
		Kind:  "Video",
		Shown: "Odysee",
		Hosts: []string{"odysee.com", "www.odysee.com"},
		Watching: func(u *url.URL) string {
			return deep(u.Path, 2)
		},
	},
	{
		Name:  "nicovideo",
		Kind:  "Video",
		Shown: "Niconico",
		Hosts: []string{"nicovideo.jp", "www.nicovideo.jp", "sp.nicovideo.jp"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/watch/")
		},
	},
	{
		Name:  "trovo",
		Kind:  "Streams",
		Shown: "Trovo",
		Hosts: []string{"trovo.live", "www.trovo.live"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/s/")
		},
	},
}

func Shown(name string) string {
	if service, ok := find(name); ok {
		return service.Shown
	}

	return name
}

func Kinds() []string {
	kinds := []string{"Video", "Streams", "Films", "Music"}

	if len(mine()) > 0 {
		kinds = append(kinds, Yours)
	}

	return kinds
}

const Yours = "Yours"

var (
	minding sync.Mutex
	added   []Service
)

func Add(host, shown string) {
	host = strings.TrimSpace(strings.ToLower(host))
	shown = strings.TrimSpace(shown)

	if host == "" {
		return
	}

	if shown == "" {
		shown = host
	}

	minding.Lock()
	defer minding.Unlock()

	for _, one := range added {
		if one.Name == host {
			return
		}
	}

	added = append(added, Service{
		Name:     host,
		Shown:    shown,
		Kind:     Yours,
		Hosts:    []string{host, "www." + host},
		Unnamed:  true,
		Watching: func(*url.URL) string { return "" },
	})
}

func mine() []Service {
	minding.Lock()
	defer minding.Unlock()

	return append([]Service(nil), added...)
}

func all() []Service {
	return append(append([]Service(nil), services...), mine()...)
}

func Kind(name string) string {
	if service, ok := find(name); ok {
		return service.Kind
	}

	return ""
}

func Heard(name string) bool {
	service, ok := find(name)

	return ok && service.Heard
}

func find(name string) (Service, bool) {
	for _, service := range all() {
		if service.Name == name {
			return service, true
		}
	}

	return Service{}, false
}

func Services() []string {
	names := make([]string, 0, len(services))

	for _, service := range all() {
		names = append(names, service.Name)
	}

	return names
}

func Recognise(address, title string, at time.Time) (Play, bool) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" {
		return Play{}, false
	}

	for _, service := range all() {
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
			Title:   tidy(title, service.Name),
			URL:     address,
			At:      at.UTC(),
		}, true
	}

	return Play{}, false
}

// A tab is called "Нечто — YouTube", and a browser puts the number of unread
// notifications in front of that. Neither is the name of anything watched.
func tidy(title, service string) string {
	title = strings.TrimSpace(counted.ReplaceAllString(title, ""))

	name := strings.ToLower(Shown(service))
	low := strings.ToLower(title)

	for _, apart := range []string{" - ", " — ", " – ", " | "} {
		if end := strings.LastIndex(low, apart+name); end > 0 {
			return strings.TrimSpace(title[:end])
		}
	}

	return title
}

var counted = regexp.MustCompile(`^\(\d+\)\s*`)

// Reported is a play the page named because the address would not. Some
// players keep the track out of the address: it sits in a bar at the foot of
func Reported(address, title string, at time.Time) (Play, bool) {
	parsed, err := url.Parse(address)
	if err != nil || parsed.Host == "" {
		return Play{}, false
	}

	for _, service := range all() {
		if !holds(service.Hosts, parsed.Host) || !service.Unnamed {
			continue
		}

		name := tidy(title, service.Name)
		if name == "" {
			return Play{}, false
		}

		return Play{
			Service: service.Name,
			ID:      name,
			Title:   name,
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

func under(path string, parts ...string) string {
	for _, part := range parts {
		if strings.HasPrefix(path, part) {
			return strings.Trim(strings.TrimPrefix(path, part), "/")
		}
	}

	return ""
}

func numbered(path string) string {
	last := path[strings.LastIndex(path, "/")+1:]

	if last == "" || strings.TrimLeft(last, "0123456789") != "" {
		return ""
	}

	return last
}

func piece(path string, markers ...string) string {
	for _, marker := range markers {
		if id := after(path, marker); id != "" {
			return id
		}
	}

	return ""
}

func deep(path string, parts int) string {
	trimmed := strings.Trim(path, "/")

	if trimmed == "" || strings.Count(trimmed, "/") < parts-1 {
		return ""
	}

	return trimmed
}
