package play

import (
	"testing"
	"time"
)

var when = time.Date(2026, 9, 3, 12, 0, 0, 0, time.UTC)

func TestRecognisesWhatIsBeingWatched(t *testing.T) {
	cases := []struct {
		address string
		service string
		id      string
	}{
		{"https://www.youtube.com/watch?v=dQw4w9WgXcQ", "youtube", "dQw4w9WgXcQ"},
		{"https://youtu.be/dQw4w9WgXcQ", "youtube", "dQw4w9WgXcQ"},
		{"https://rutube.ru/video/abc123/", "rutube", "abc123"},
		{"https://music.yandex.ru/album/42/track/777", "yandex-music", "777"},
		{"https://www.twitch.tv/videos/1234", "twitch", "1234"},
	}

	for _, one := range cases {
		got, ok := Recognise(one.address, "нечто", when)
		if !ok {
			t.Fatalf("%s: not recognised at all", one.address)
		}

		if got.Service != one.service || got.ID != one.id {
			t.Errorf("%s: got %s/%s, wanted %s/%s",
				one.address, got.Service, got.ID, one.service, one.id)
		}
	}
}

// A page on a service is not the same as a thing being watched on it. Counting
// a search as a play would fill the list with everything somebody typed.
func TestPassesOverPagesThatAreNotAThingBeingWatched(t *testing.T) {
	for _, address := range []string{
		"https://www.youtube.com/results?search_query=cats",
		"https://www.youtube.com/feed/subscriptions",
		"https://music.yandex.ru/album/42",
		"https://rutube.ru/",
		"https://example.com/watch?v=whatever",
	} {
		if _, ok := Recognise(address, "", when); ok {
			t.Errorf("%s: counted as a play, and it is not one", address)
		}
	}
}

func TestSaysNoToAnAddressThatIsNotOne(t *testing.T) {
	for _, address := range []string{"", "not an address", "/just/a/path"} {
		if _, ok := Recognise(address, "", when); ok {
			t.Errorf("%q: read as an address", address)
		}
	}
}

// A live channel has no video id, and its name is the only name it has while
// it is running.
func TestReadsALiveChannelByItsName(t *testing.T) {
	got, ok := Recognise("https://www.twitch.tv/somebody", "", when)
	if !ok || got.ID != "somebody" {
		t.Errorf("got %+v, wanted the channel name", got)
	}
}

func TestKeepsTheTimeInOneZone(t *testing.T) {
	local := time.Date(2026, 9, 3, 15, 0, 0, 0, time.FixedZone("MSK", 3*60*60))

	got, _ := Recognise("https://youtu.be/x", "", local)
	if got.At.Location() != time.UTC {
		t.Errorf("kept %v, and a list sorted across zones is not sorted", got.At.Location())
	}
}

func TestRecognisesTheOtherServices(t *testing.T) {
	for address, wanted := range map[string]string{
		"https://dzen.ru/video/watch/64f0ab":       "dzen",
		"https://ok.ru/video/8123456":              "ok-video",
		"https://vimeo.com/347119375":              "vimeo",
		"https://www.dailymotion.com/video/x8abcd": "dailymotion",
		"https://coub.com/view/2abcde":             "coub",
		"https://www.netflix.com/watch/81234567":   "netflix",
		"https://hd.kinopoisk.ru/watch/4d1eba8b":   "kinopoisk",
		"https://okko.tv/movie/nechto":             "okko",
		"https://www.ivi.ru/watch/199617":          "ivi",
		"https://wink.ru/movies/nechto":            "wink",
		"https://premier.one/show/nechto":          "premier",
		"https://kick.com/somebody":                "kick",
		"https://live.vkplay.ru/somebody":          "vkplay",
	} {
		one, ok := Recognise(address, "Нечто", time.Now())

		if !ok || one.Service != wanted {
			t.Errorf("%s: got %q %v", address, one.Service, ok)
		}
	}
}

// A catalogue is a page somebody browsed, not a film they watched.
func TestIgnoresTheirCatalogues(t *testing.T) {
	for _, address := range []string{
		"https://www.ivi.ru/new",
		"https://okko.tv/collections/hits",
		"https://vimeo.com/upgrade",
		"https://dzen.ru/news",
		"https://premier.one/",
	} {
		if _, ok := Recognise(address, "Нечто", time.Now()); ok {
			t.Errorf("%s counted as a watch", address)
		}
	}
}

func TestNamesServicesTheWayPeopleReadThem(t *testing.T) {
	if got := Shown("yandex-music"); got != "Yandex Music" {
		t.Errorf("got %q", got)
	}

	// A play imported before the list changed still has to be drawn.
	if got := Shown("myvi"); got != "myvi" {
		t.Errorf("got %q", got)
	}

	if !Heard("yandex-music") || Heard("youtube") {
		t.Error("music is watched or video is listened to")
	}
}

func TestRecognisesMusicAndForeignServices(t *testing.T) {
	for address, wanted := range map[string]string{
		"https://music.youtube.com/watch?v=abc":          "youtube-music",
		"https://open.spotify.com/track/4cOdK2wGLETKBW3": "spotify",
		"https://open.spotify.com/intl-ru/album/1DFixLW": "spotify",
		"https://soundcloud.com/somebody/a-song":         "soundcloud",
		"https://music.apple.com/us/album/nechto/1234":   "apple-music",
		"https://www.deezer.com/en/track/3135556":        "deezer",
		"https://zvuk.com/track/12345":                   "zvuk",
		"https://www.mixcloud.com/somebody/a-show/":      "mixcloud",
		"https://www.tiktok.com/@somebody/video/7123":    "tiktok",
		"https://www.bilibili.com/video/BV1xx411":        "bilibili",
		"https://www.crunchyroll.com/watch/GRDQ/nechto":  "crunchyroll",
		"https://www.disneyplus.com/en-gb/video/abc-123": "disney-plus",
		"https://play.max.com/video/watch/abc/def":       "max",
		"https://www.primevideo.com/detail/0ABCDE":       "prime-video",
		"https://tv.apple.com/us/episode/nechto/umc.1":   "apple-tv",
		"https://www.hulu.com/watch/abc-123":             "hulu",
		"https://nebula.tv/videos/a-slug":                "nebula",
		"https://odysee.com/@somebody/a-video":           "odysee",
		"https://www.nicovideo.jp/watch/sm9":             "nicovideo",
		"https://trovo.live/s/somebody":                  "trovo",
	} {
		one, ok := Recognise(address, "Something", time.Now())

		if !ok || one.Service != wanted {
			t.Errorf("%s: got %q %v", address, one.Service, ok)
		}
	}
}

// A person is not a track, and a front page is not a film.
func TestIgnoresProfilesAndFrontPages(t *testing.T) {
	for _, address := range []string{
		"https://soundcloud.com/somebody",
		"https://odysee.com/@somebody",
		"https://open.spotify.com/search/nechto",
		"https://www.tiktok.com/@somebody",
		"https://www.hulu.com/hub/movies",
	} {
		if _, ok := Recognise(address, "Something", time.Now()); ok {
			t.Errorf("%s counted as a watch", address)
		}
	}
}

// A watch page names the video in a parameter and everything else names it in
// the path. Shorts are watched more than anything, and went unwritten.
func TestRecognisesShortsAndStreams(t *testing.T) {
	for address, wanted := range map[string]string{
		"https://www.youtube.com/shorts/abc123": "abc123",
		"https://www.youtube.com/live/def456":   "def456",
		"https://www.youtube.com/embed/ghi789":  "ghi789",
		"https://m.youtube.com/watch?v=jkl":     "jkl",
		"https://youtu.be/mno":                  "mno",
		"https://rutube.ru/shorts/pqr/":         "pqr",
		"https://rutube.ru/video/stu/":          "stu",
	} {
		one, ok := Recognise(address, "Something", time.Now())

		if !ok || one.ID != wanted {
			t.Errorf("%s: got %q %v", address, one.ID, ok)
		}
	}
}

// Clips are named the way videos are, and go by faster and in greater numbers.
func TestRecognisesClips(t *testing.T) {
	for address, wanted := range map[string]string{
		"https://vk.com/clip-2000123_456789":     "-2000123_456789",
		"https://vkvideo.ru/clip-2000123_456789": "-2000123_456789",
		"https://vk.com/video-123_456":           "-123_456",
		"https://clips.twitch.tv/SomeClipName":   "SomeClipName",
	} {
		one, ok := Recognise(address, "Something", time.Now())

		if !ok || one.ID != wanted {
			t.Errorf("%s: got %q %v", address, one.ID, ok)
		}
	}
}
