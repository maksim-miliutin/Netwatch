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

			// A watch page names the video in a parameter, and everything
			// else names it in the path. Shorts are watched more than
			// anything else on there and went unwritten for months.
			if id := piece(u.Path, "/shorts/", "/live/", "/embed/"); id != "" {
				return id
			}

			return u.Query().Get("v")
		},
	},
	{
		Name:  "rutube",
		Shown: "RuTube",
		Hosts: []string{"rutube.ru", "www.rutube.ru"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/video/", "/shorts/", "/live/")
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
			// A clip is named the same way a video is, and is watched more.
			return piece(u.Path, "/video", "/clip")
		},
	},
	{
		Name:  "twitch",
		Shown: "Twitch",
		Hosts: []string{"twitch.tv", "www.twitch.tv", "clips.twitch.tv"},
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
			return piece(u.Path, "/video/watch/", "/shorts/")
		},
	},
	{
		Name:  "ok-video",
		Shown: "OK Video",
		Hosts: []string{"ok.ru", "www.ok.ru", "m.ok.ru"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/video/", "/live/")
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
	{
		Name:  "youtube-music",
		Shown: "YouTube Music",
		Heard: true,
		Hosts: []string{"music.youtube.com"},
		Watching: func(u *url.URL) string {
			return u.Query().Get("v")
		},
	},
	{
		Name:  "spotify",
		Shown: "Spotify",
		Heard: true,
		Hosts: []string{"open.spotify.com", "play.spotify.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/track/", "/album/", "/playlist/", "/episode/", "/show/")
		},
	},
	{
		Name:  "soundcloud",
		Shown: "SoundCloud",
		Heard: true,
		Hosts: []string{"soundcloud.com", "m.soundcloud.com", "on.soundcloud.com"},
		Watching: func(u *url.URL) string {
			// A name on its own is a person; a name with something under it is
			// a track by that person.
			return deep(u.Path, 2)
		},
	},
	{
		Name:  "apple-music",
		Shown: "Apple Music",
		Heard: true,
		Hosts: []string{"music.apple.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/album/", "/playlist/", "/song/")
		},
	},
	{
		Name:  "deezer",
		Shown: "Deezer",
		Heard: true,
		Hosts: []string{"deezer.com", "www.deezer.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/track/", "/album/", "/playlist/", "/episode/")
		},
	},
	{
		Name:  "zvuk",
		Shown: "Zvuk",
		Heard: true,
		Hosts: []string{"zvuk.com", "www.zvuk.com", "sber-zvuk.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/track/", "/release/", "/playlist/")
		},
	},
	{
		Name:  "mixcloud",
		Shown: "Mixcloud",
		Heard: true,
		Hosts: []string{"mixcloud.com", "www.mixcloud.com"},
		Watching: func(u *url.URL) string {
			return deep(u.Path, 2)
		},
	},
	{
		Name:  "tiktok",
		Shown: "TikTok",
		Hosts: []string{"tiktok.com", "www.tiktok.com", "vm.tiktok.com"},
		Watching: func(u *url.URL) string {
			// Nobody shares the long address. A short one is the whole of the
			// path, and it stands in for an id: two of them are two videos
			// even when neither says which.
			if u.Host == "vm.tiktok.com" {
				return strings.Trim(u.Path, "/")
			}

			return piece(u.Path, "/video/", "/t/")
		},
	},
	{
		Name:  "bilibili",
		Shown: "Bilibili",
		Hosts: []string{"bilibili.com", "www.bilibili.com", "m.bilibili.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "crunchyroll",
		Shown: "Crunchyroll",
		Hosts: []string{"crunchyroll.com", "www.crunchyroll.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/watch/")
		},
	},
	{
		Name:  "disney-plus",
		Shown: "Disney+",
		Hosts: []string{"disneyplus.com", "www.disneyplus.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/video/")
		},
	},
	{
		Name:  "max",
		Shown: "Max",
		Hosts: []string{"max.com", "www.max.com", "play.max.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/video/", "/movie/", "/show/")
		},
	},
	{
		Name:  "prime-video",
		Shown: "Prime Video",
		Hosts: []string{"primevideo.com", "www.primevideo.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/detail/", "/watch/")
		},
	},
	{
		Name:  "apple-tv",
		Shown: "Apple TV+",
		Hosts: []string{"tv.apple.com"},
		Watching: func(u *url.URL) string {
			return piece(u.Path, "/episode/", "/movie/", "/show/")
		},
	},
	{
		Name:  "hulu",
		Shown: "Hulu",
		Hosts: []string{"hulu.com", "www.hulu.com"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/watch/")
		},
	},
	{
		Name:  "nebula",
		Shown: "Nebula",
		Hosts: []string{"nebula.tv", "www.nebula.tv"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/videos/")
		},
	},
	{
		Name:  "odysee",
		Shown: "Odysee",
		Hosts: []string{"odysee.com", "www.odysee.com"},
		Watching: func(u *url.URL) string {
			return deep(u.Path, 2)
		},
	},
	{
		Name:  "nicovideo",
		Shown: "Niconico",
		Hosts: []string{"nicovideo.jp", "www.nicovideo.jp", "sp.nicovideo.jp"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/watch/")
		},
	},
	{
		Name:  "trovo",
		Shown: "Trovo",
		Hosts: []string{"trovo.live", "www.trovo.live"},
		Watching: func(u *url.URL) string {
			return after(u.Path, "/s/")
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

// piece is what follows whichever of these markers the address uses. A site
// that names a track, an album and a playlist the same way needs one rule
// rather than three services.
func piece(path string, markers ...string) string {
	for _, marker := range markers {
		if id := after(path, marker); id != "" {
			return id
		}
	}

	return ""
}

// deep is the path when it goes at least that many parts down. It is how a
// site that names a track after its author tells one from a profile page.
func deep(path string, parts int) string {
	trimmed := strings.Trim(path, "/")

	if trimmed == "" || strings.Count(trimmed, "/") < parts-1 {
		return ""
	}

	return trimmed
}
