package web

import (
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"netwatch/internal/now"
	"netwatch/internal/store"
)

func serving(t *testing.T) http.Handler {
	t.Helper()

	kept, err := store.Open(filepath.Join(t.TempDir(), "plays.jsonl"))
	if err != nil {
		t.Fatal(err)
	}

	return (&Server{Store: kept, Watching: now.New()}).Routes()
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

	if page := shown(t, handler); !strings.Contains(page, "Now: <b>Нечто</b>") {
		t.Errorf("the page says nothing about it: %s", page)
	}
}

func TestSaysNothingIsOnUntilSomethingIs(t *testing.T) {
	if page := shown(t, serving(t)); strings.Contains(page, "Now:") {
		t.Errorf("something plays on a machine nobody touched: %s", page)
	}
}

func TestStopsWhenTheTabIsGone(t *testing.T) {
	handler := serving(t)

	sent(t, handler, "/api/now", `{"url":"https://youtu.be/abc","title":"Нечто"}`)
	sent(t, handler, "/api/gone", `{}`)

	if page := shown(t, handler); strings.Contains(page, "Now:") {
		t.Error("still playing after the tab went")
	}
}

// Somebody who went from a video to a search is watching nothing.
func TestAPageThatIsNotAPlayEndsWhatWasPlaying(t *testing.T) {
	handler := serving(t)

	sent(t, handler, "/api/now", `{"url":"https://youtu.be/abc","title":"Нечто"}`)
	sent(t, handler, "/api/now", `{"url":"https://www.youtube.com/results?q=нечто","title":"Поиск"}`)

	if page := shown(t, handler); strings.Contains(page, "Now:") {
		t.Error("still playing after the tab went to a search")
	}
}
