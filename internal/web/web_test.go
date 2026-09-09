package web

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"netwatch/internal/mine"
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

func own(t *testing.T) *mine.List {
	t.Helper()

	list, err := mine.Open(filepath.Join(t.TempDir(), "services"))
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

	return (&Server{Store: kept, Watching: now.New(), Quiet: hushed(t), Mine: own(t)}).Routes()
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

	handler := (&Server{Store: kept, Watching: now.New(), Quiet: hushed(t), Mine: own(t)}).Routes()
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

// Three spans in a ring, and the page names the next one.
func TestShowsAWeekAMonthOrTheLot(t *testing.T) {
	handler := serving(t)

	for _, step := range []struct {
		asked  string
		titled string
		offers string
	}{
		{"/", "This week", "span=month"},
		{"/?span=month", "This month", "span=all"},
		{"/?span=all", "All of it", "span=week"},
	} {
		page := httptest.NewRecorder()
		handler.ServeHTTP(page, httptest.NewRequest("GET", step.asked, nil))

		if !strings.Contains(page.Body.String(), step.titled) {
			t.Errorf("%s is not titled %q", step.asked, step.titled)
		}

		if !strings.Contains(page.Body.String(), step.offers) {
			t.Errorf("%s does not offer %s", step.asked, step.offers)
		}
	}
}

// A search somebody just typed should survive a click on something else.
func TestKeepsTheSearchWhenTheSpanChanges(t *testing.T) {
	page := httptest.NewRecorder()
	serving(t).ServeHTTP(page, httptest.NewRequest("GET", "/?find=bracket", nil))

	if !strings.Contains(page.Body.String(), "find=bracket") {
		t.Errorf("the link away drops the search: %s", page.Body)
	}
}

// A year of watching is thousands of lines, and the way back to one of them is
// its name.
func TestFindsAPlayByName(t *testing.T) {
	handler := serving(t)

	sent(t, handler, "/api/now", `{"url":"https://youtu.be/abc","title":"Bottom bracket"}`)
	sent(t, handler, "/api/gone", `{}`)
	sent(t, handler, "/api/now", `{"url":"https://rutube.ru/video/xyz/","title":"Something else"}`)

	// Nothing playing: the line at the top is not the list and does not answer
	// to a search, so it would turn up whatever was asked for.
	sent(t, handler, "/api/gone", `{}`)

	found := httptest.NewRecorder()
	handler.ServeHTTP(found, httptest.NewRequest("GET", "/?find=bracket", nil))

	if !strings.Contains(found.Body.String(), "Bottom bracket") {
		t.Error("did not find it by name")
	}

	if strings.Contains(found.Body.String(), "Something else") {
		t.Error("kept what was not asked for")
	}

	// Somebody looking for "rutube" means the service.
	service := httptest.NewRecorder()
	handler.ServeHTTP(service, httptest.NewRequest("GET", "/?find=RUTUBE", nil))

	if !strings.Contains(service.Body.String(), "Something else") {
		t.Error("did not find it by service, or minded the capitals")
	}
}

// A line of JSON per play is honest and unreadable by anything somebody
// already has on their machine.
func TestHandsTheListOverAsASpreadsheet(t *testing.T) {
	handler := serving(t)

	sent(t, handler, "/api/now", `{"url":"https://youtu.be/abc","title":"Нечто, с запятой"}`)

	sheet := httptest.NewRecorder()
	handler.ServeHTTP(sheet, httptest.NewRequest("GET", "/plays.csv", nil))

	if sheet.Code != http.StatusOK {
		t.Fatalf("got %d", sheet.Code)
	}

	if kind := sheet.Header().Get("content-type"); !strings.HasPrefix(kind, "text/csv") {
		t.Errorf("came back as %q", kind)
	}

	body := sheet.Body.String()

	if !strings.Contains(body, "at,service,id,seconds,title,url") {
		t.Errorf("no heading: %s", body)
	}

	// A comma in a title is what breaks a spreadsheet written by hand.
	if !strings.Contains(body, `"Нечто, с запятой"`) {
		t.Errorf("the comma went through unquoted: %s", body)
	}
}

// Crossing a line out is a write, so it comes off this page or not at all.
func TestCrossesOutOnePlay(t *testing.T) {
	handler := serving(t)

	sent(t, handler, "/api/now", `{"url":"https://youtu.be/abc","title":"Нечто"}`)

	plays := httptest.NewRecorder()
	handler.ServeHTTP(plays, httptest.NewRequest("GET", "/api/plays", nil))

	var kept []play.Play
	if err := json.Unmarshal(plays.Body.Bytes(), &kept); err != nil {
		t.Fatal(err)
	}

	body := "service=" + kept[0].Service + "&id=" + kept[0].ID +
		"&at=" + url.QueryEscape(kept[0].At.Format(time.RFC3339Nano))

	request := httptest.NewRequest("POST", "/api/forget", strings.NewReader(body))
	request.Host = "127.0.0.1:7373"
	request.RemoteAddr = "127.0.0.1:5000"
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("origin", "http://127.0.0.1:7373")

	// Where the button was pressed, so that a search is not thrown away.
	request.Header.Set("referer", "http://127.0.0.1:7373/?find=bracket")

	answer := httptest.NewRecorder()
	Near(handler).ServeHTTP(answer, request)

	if answer.Code != http.StatusSeeOther {
		t.Fatalf("got %d %s", answer.Code, answer.Body)
	}

	if back := answer.Header().Get("location"); back != "http://127.0.0.1:7373/?find=bracket" {
		t.Errorf("came back to %q", back)
	}

	left := httptest.NewRecorder()
	handler.ServeHTTP(left, httptest.NewRequest("GET", "/api/plays", nil))

	if strings.Contains(left.Body.String(), "Нечто") {
		t.Errorf("still there: %s", left.Body)
	}
}

// A browser that stopped reporting looks exactly like an evening nobody
// watched anything. Only the date of the last play tells them apart.
func TestSaysWhetherItIsWorking(t *testing.T) {
	handler := serving(t)

	first := shown(t, handler)

	if !strings.Contains(first, "Nothing reported yet") {
		t.Errorf("said nothing about a fresh machine")
	}

	if !strings.Contains(first, "Two things to do") {
		t.Errorf("did not say what to do first")
	}

	sent(t, handler, "/api/seen", `{"url":"https://youtu.be/abc","title":"Нечто"}`)

	after := shown(t, handler)

	if !strings.Contains(after, "Reporting") || !strings.Contains(after, "1 play kept.") {
		t.Errorf("got %q", after[:200])
	}

	// The two things are done, so the block that said to do them is gone.
	if strings.Contains(after, "Two things to do") {
		t.Error("still telling somebody to start")
	}
}

// A site somebody adds brings no rule for reading its addresses, and needs
// none: the page names what it plays.
func TestTakesASiteSomebodyAdds(t *testing.T) {
	handler := serving(t)

	body := strings.NewReader("host=https://plvideo.ru/some/path&shown=Plvideo")

	request := httptest.NewRequest("POST", "/api/services", body)
	request.Host = "127.0.0.1:7373"
	request.RemoteAddr = "127.0.0.1:5000"
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("origin", "http://127.0.0.1:7373")

	answer := httptest.NewRecorder()
	Near(handler).ServeHTTP(answer, request)

	if answer.Code != http.StatusSeeOther {
		t.Fatalf("got %d %s", answer.Code, answer.Body)
	}

	// The address had a scheme and a path on it, and only the host is a host.
	one, ok := play.Reported("https://plvideo.ru/watch/abc", "Нечто", time.Now())
	if !ok || one.Service != "plvideo.ru" || one.Title != "Нечто" {
		t.Errorf("got %+v %v", one, ok)
	}
}

// The extension asks for the sites somebody added, because the browser has
// never been told about those and the built in ones are in the manifest.
func TestTellsTheExtensionAboutAddedSites(t *testing.T) {
	handler := serving(t)

	empty := httptest.NewRecorder()
	handler.ServeHTTP(empty, httptest.NewRequest("GET", "/api/mine", nil))

	if strings.TrimSpace(empty.Body.String()) != "{}" {
		t.Errorf("a fresh machine offers %s", empty.Body)
	}

	body := strings.NewReader("host=some.example&shown=Some")

	request := httptest.NewRequest("POST", "/api/services", body)
	request.Host = "127.0.0.1:7373"
	request.RemoteAddr = "127.0.0.1:5000"
	request.Header.Set("content-type", "application/x-www-form-urlencoded")
	request.Header.Set("origin", "http://127.0.0.1:7373")

	Near(handler).ServeHTTP(httptest.NewRecorder(), request)

	listed := httptest.NewRecorder()
	handler.ServeHTTP(listed, httptest.NewRequest("GET", "/api/mine", nil))

	if !strings.Contains(listed.Body.String(), `"some.example":"Some"`) {
		t.Errorf("got %s", listed.Body)
	}
}
