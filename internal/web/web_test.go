package web

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"netwatch/internal/now"
	"netwatch/internal/play"
	"netwatch/internal/quiet"
	"netwatch/internal/store"
)

func hushed(t *testing.T) *quiet.List {
	t.Helper()

	list, err := quiet.Open(filepath.Join(t.TempDir(), "quiet"))
	if err != nil {
		t.Fatal(err)
	}

	return list
}

func serving(t *testing.T) http.Handler {
	t.Helper()

	kept, err := store.Open(filepath.Join(t.TempDir(), "plays.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	return (&Server{Store: kept, Watching: now.New(), Quiet: hushed(t)}).Routes()
}

func post(t *testing.T, handler http.Handler, body string) *httptest.ResponseRecorder {
	return sent(t, handler, "/api/seen", body)
}

func sent(t *testing.T, handler http.Handler, where, body string) *httptest.ResponseRecorder {
	t.Helper()

	request := httptest.NewRequest("POST", where, strings.NewReader(body))
	answer := httptest.NewRecorder()

	handler.ServeHTTP(answer, request)

	return answer
}

func TestKeepsAThingBeingWatched(t *testing.T) {
	handler := serving(t)

	answer := post(t, handler,
		`{"url":"https://youtu.be/abc","title":"Нечто"}`)

	if answer.Code != http.StatusOK || !strings.Contains(answer.Body.String(), `"kept":true`) {
		t.Fatalf("got %d %s", answer.Code, answer.Body)
	}

	list := httptest.NewRecorder()
	handler.ServeHTTP(list, httptest.NewRequest("GET", "/api/plays", nil))

	if !strings.Contains(list.Body.String(), "abc") {
		t.Errorf("not in the list: %s", list.Body)
	}
}

// The extension reports every tab. Most tabs are not a thing being watched,
// and refusing them would be an error where there is none.
func TestTakesAPageThatIsNotAPlayWithoutComplaining(t *testing.T) {
	answer := post(t, serving(t), `{"url":"https://example.com/","title":"nothing"}`)

	if answer.Code != http.StatusOK {
		t.Errorf("got %d, wanted a plain no", answer.Code)
	}

	if !strings.Contains(answer.Body.String(), `"kept":false`) {
		t.Errorf("got %s", answer.Body)
	}
}

func TestRefusesABodyItCannotRead(t *testing.T) {
	if answer := post(t, serving(t), "не json"); answer.Code != http.StatusBadRequest {
		t.Errorf("got %d", answer.Code)
	}
}

func TestShowsThePage(t *testing.T) {
	answer := httptest.NewRecorder()
	serving(t).ServeHTTP(answer, httptest.NewRequest("GET", "/", nil))

	if answer.Code != http.StatusOK || !strings.Contains(answer.Body.String(), "netwatch") {
		t.Errorf("got %d", answer.Code)
	}
}

// A program that binds loopback and trusts everything it gets is one setting
// away from being an open door.
func TestAnswersOnlyThisMachine(t *testing.T) {
	guarded := Near(serving(t))

	for address, wanted := range map[string]int{
		"127.0.0.1:5000":   http.StatusOK,
		"[::1]:5000":       http.StatusOK,
		"192.168.0.5:5000": http.StatusForbidden,
		"8.8.8.8:5000":     http.StatusForbidden,
	} {
		request := httptest.NewRequest("GET", "/api/services", nil)
		request.Host = "127.0.0.1:7373"
		request.RemoteAddr = address

		answer := httptest.NewRecorder()
		guarded.ServeHTTP(answer, request)

		if answer.Code != wanted {
			t.Errorf("%s: got %d, wanted %d", address, answer.Code, wanted)
		}
	}
}

func TestAddsUpTheWeek(t *testing.T) {
	handler := serving(t)

	for _, address := range []string{
		"https://youtu.be/a", "https://youtu.be/b", "https://rutube.ru/video/c/",
	} {
		post(t, handler, `{"url":"`+address+`","title":"нечто"}`)
	}

	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer, httptest.NewRequest("GET", "/api/week", nil))

	if answer.Code != http.StatusOK {
		t.Fatalf("got %d", answer.Code)
	}

	if !strings.Contains(answer.Body.String(), `"plays":3`) {
		t.Errorf("got %s", answer.Body)
	}
}

// Nobody counts a week in seconds.
func TestSaysHowLongInWordsPeopleUse(t *testing.T) {
	for seconds, wanted := range map[int]string{
		45: "45 sec", 300: "5 min", 3900: "1 h 5 min",
	} {
		if got := hours(seconds); got != wanted {
			t.Errorf("%d: got %q, wanted %q", seconds, got, wanted)
		}
	}
}

func shown(t *testing.T, handler http.Handler) string {
	t.Helper()

	answer := httptest.NewRecorder()
	handler.ServeHTTP(answer, httptest.NewRequest("GET", "/", nil))

	return answer.Body.String()
}

// A page knows the name of what it plays; a tab knows the name of the tab.
func TestKeepsWhatAPlayingPageSaysAboutItself(t *testing.T) {
	handler := serving(t)

	answer := sent(t, handler, "/api/now",
		`{"url":"https://youtu.be/abc","title":"Нечто","position":12.5,"length":600}`)

	if answer.Code != http.StatusOK {
		t.Fatalf("got %d %s", answer.Code, answer.Body)
	}

	if page := shown(t, handler); !strings.Contains(page, `class="what">Нечто`) {
		t.Errorf("the page says nothing about it: %s", page)
	}
}

func TestSaysNothingIsOnUntilSomethingIs(t *testing.T) {
	if page := shown(t, serving(t)); strings.Contains(page, `class="now"`) {
		t.Errorf("something plays on a machine nobody touched: %s", page)
	}
}

func TestStopsWhenTheTabIsGone(t *testing.T) {
	handler := serving(t)

	sent(t, handler, "/api/now", `{"url":"https://youtu.be/abc","title":"Нечто"}`)
	sent(t, handler, "/api/gone", `{}`)

	if page := shown(t, handler); strings.Contains(page, `class="now"`) {
		t.Error("still playing after the tab went")
	}
}

// Somebody who went from a video to a search is watching nothing.
func TestAPageThatIsNotAPlayEndsWhatWasPlaying(t *testing.T) {
	handler := serving(t)

	sent(t, handler, "/api/now", `{"url":"https://youtu.be/abc","title":"Нечто"}`)
	sent(t, handler, "/api/now", `{"url":"https://www.youtube.com/results?q=нечто","title":"Поиск"}`)

	if page := shown(t, handler); strings.Contains(page, `class="now"`) {
		t.Error("still playing after the tab went to a search")
	}
}

// A name somebody else owns, pointed at this machine, makes a browser treat
// their page as this one. The address dialled is loopback either way.
func TestRefusesANameThatIsNotThisMachine(t *testing.T) {
	for name, wanted := range map[string]int{
		"127.0.0.1:7373": http.StatusOK,
		"localhost:7373": http.StatusOK,
		"[::1]:7373":     http.StatusOK,
		"evil.example":   http.StatusForbidden,
	} {
		request := httptest.NewRequest("GET", "/api/services", nil)
		request.Host = name
		request.RemoteAddr = "127.0.0.1:5000"

		answer := httptest.NewRecorder()
		Near(serving(t)).ServeHTTP(answer, request)

		if answer.Code != wanted {
			t.Errorf("%s: got %d, wanted %d", name, answer.Code, wanted)
		}
	}
}

// A page can post to any address it likes without asking the browser first, so
// long as what it sends looks like a form.
func TestRefusesAPostThatDidNotMeanToComeHere(t *testing.T) {
	for kind, wanted := range map[string]int{
		"application/json":                  http.StatusOK,
		"application/json; charset=utf-8":   http.StatusOK,
		"text/plain":                        http.StatusUnsupportedMediaType,
		"application/x-www-form-urlencoded": http.StatusUnsupportedMediaType,
		"":                                  http.StatusUnsupportedMediaType,
	} {
		request := httptest.NewRequest("POST", "/api/now",
			strings.NewReader(`{"url":"https://youtu.be/abc","title":"Нечто"}`))
		request.Host = "127.0.0.1:7373"
		request.RemoteAddr = "127.0.0.1:5000"
		request.Header.Set("content-type", kind)

		answer := httptest.NewRecorder()
		Near(serving(t)).ServeHTTP(answer, request)

		if answer.Code != wanted {
			t.Errorf("%q: got %d, wanted %d", kind, answer.Code, wanted)
		}
	}
}

// A Takeout import is tens of thousands of rows. Drawing all of them is a page
// nobody scrolls, redrawn every ten seconds.
func TestDrawsOnlySoMuchOfTheList(t *testing.T) {
	kept, err := store.Open(filepath.Join(t.TempDir(), "plays.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	at := time.Now().UTC()

	for i := 0; i < Shows+5; i++ {
		one := play.Play{
			Service: "youtube",
			ID:      fmt.Sprintf("v%d", i),
			Title:   fmt.Sprintf("Number %d", i),
			At:      at.Add(-time.Duration(i) * time.Minute),
		}

		if err := kept.Add(one); err != nil {
			t.Fatal(err)
		}
	}

	handler := (&Server{Store: kept, Watching: now.New(), Quiet: hushed(t)}).Routes()
	page := shown(t, handler)

	if strings.Count(page, "<tr>") != Shows {
		t.Errorf("drew %d rows, wanted %d", strings.Count(page, "<tr>"), Shows)
	}

	if !strings.Contains(page, "5 older in the file") {
		t.Error("said nothing about the rest")
	}
}

// The popup in the browser has nothing else to go on: it cannot read the page,
// and it has to tell "not running" from "running, nothing playing".
func TestTellsThePopupWhatIsPlaying(t *testing.T) {
	handler := serving(t)

	empty := httptest.NewRecorder()
	handler.ServeHTTP(empty, httptest.NewRequest("GET", "/api/now", nil))

	if !strings.Contains(empty.Body.String(), `"playing":false`) {
		t.Errorf("got %s on an untouched machine", empty.Body)
	}

	sent(t, handler, "/api/now",
		`{"url":"https://youtu.be/abc","title":"Нечто","by":"Кто-то","position":30,"length":600}`)

	on := httptest.NewRecorder()
	handler.ServeHTTP(on, httptest.NewRequest("GET", "/api/now", nil))

	for _, want := range []string{`"playing":true`, `"title":"Нечто"`, `"by":"Кто-то"`, `"whole":600`} {
		if !strings.Contains(on.Body.String(), want) {
			t.Errorf("got %s, without %s", on.Body, want)
		}
	}
}

// A date is something to work out. The two days people read most should not
// need working out.
func TestSaysTodayAndYesterdayInWords(t *testing.T) {
	at := time.Date(2026, time.September, 5, 18, 0, 0, 0, time.Local)

	for _, when := range []struct {
		played time.Time
		wanted string
	}{
		{at.Add(-time.Hour), "Today"},
		{at.AddDate(0, 0, -1), "Yesterday"},
		{at.AddDate(0, 0, -3), "Wednesday, 2 September"},
	} {
		one := play.Play{Service: "youtube", ID: "a", At: when.played}

		if got := byDay([]play.Play{one}, at)[0].Date; got != when.wanted {
			t.Errorf("got %q, wanted %q", got, when.wanted)
		}
	}
}

// A play just before midnight and one just after are two days, not one, even
// though barely a minute went by.
func TestMidnightStartsANewDay(t *testing.T) {
	at := time.Date(2026, time.September, 5, 12, 0, 0, 0, time.Local)

	late := play.Play{Service: "youtube", ID: "a", At: at.AddDate(0, 0, -1).Add(11*time.Hour + 59*time.Minute)}
	early := play.Play{Service: "youtube", ID: "b", At: at.Add(-11*time.Hour - 59*time.Minute)}

	if days := byDay([]play.Play{early, late}, at); len(days) != 2 {
		t.Errorf("got %d days, wanted two", len(days))
	}
}

// A form off the page is the other way in: a browser says where a form came
// from, and this one has to have come from here.
func TestTakesTheFormThatChoosesServices(t *testing.T) {
	handler := serving(t)

	body := strings.NewReader("service=youtube&service=twitch&show=youtube")

	request := httptest.NewRequest("POST", "/api/quiet", body)
	request.Host = "127.0.0.1:7373"
	request.RemoteAddr = "127.0.0.1:5000"
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("origin", "http://127.0.0.1:7373")

	answer := httptest.NewRecorder()
	Near(handler).ServeHTTP(answer, request)

	if answer.Code != http.StatusSeeOther {
		t.Fatalf("got %d %s", answer.Code, answer.Body)
	}
}

// The same form from somebody else's page is refused: a card is read by
// everybody in a server, and who chooses what goes on it matters.
func TestRefusesTheSameFormFromSomewhereElse(t *testing.T) {
	request := httptest.NewRequest("POST", "/api/quiet",
		strings.NewReader("service=youtube"))
	request.Host = "127.0.0.1:7373"
	request.RemoteAddr = "127.0.0.1:5000"
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("origin", "https://evil.example")

	answer := httptest.NewRecorder()
	Near(serving(t)).ServeHTTP(answer, request)

	if answer.Code != http.StatusUnsupportedMediaType {
		t.Errorf("got %d, wanted it refused", answer.Code)
	}
}
