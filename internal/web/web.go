// Package web answers the browser: one page to look at, one address for a
// browser extension to report to.
package web

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"strings"
	"time"

	"netwatch/internal/now"
	"netwatch/internal/play"
	"netwatch/internal/store"
	"netwatch/internal/sum"
)

//go:embed page.html
var pageSource string

// Minutes and hours rather than seconds: nobody counts a week in seconds.
var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"hours": hours,
	"times": times,
	"clock": clock,
}).Parse(pageSource))

func hours(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d sec", seconds)
	}

	if seconds < 3600 {
		return fmt.Sprintf("%d min", seconds/60)
	}

	return fmt.Sprintf("%d h %d min", seconds/3600, (seconds%3600)/60)
}

func times(count int) string {
	if count == 1 {
		return "once"
	}

	return fmt.Sprintf("%d times", count)
}

type Server struct {
	Store    *store.Store
	Watching *now.Watch
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.show)
	mux.HandleFunc("POST /api/seen", s.seen)
	mux.HandleFunc("POST /api/now", s.playing)
	mux.HandleFunc("POST /api/gone", s.gone)
	mux.HandleFunc("GET /api/now", s.showing)
	mux.HandleFunc("GET /api/plays", s.plays)
	mux.HandleFunc("GET /api/services", s.services)
	mux.HandleFunc("GET /api/week", s.week)

	return mux
}

// Nothing here is reachable from anywhere but this machine. Checked per
// request rather than only in the address it listens on: a program that binds
// loopback and trusts everything it gets is one setting away from an open door.
//
// The name it was called by is checked too. A name somebody else owns, pointed
// at 127.0.0.1, makes a browser treat their page as this one — and the address
// dialled is loopback either way, so the first check sees nothing wrong.
//
// And that whoever wrote here meant to. A page can post to any address it likes
// without asking the browser first, so long as what it sends looks like a form
// — enough to write the list and put anything on the card. It cannot send this
// content type without asking, and nothing here ever answers that question.
func Near(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !here(r.RemoteAddr) {
			http.Error(w, "this answers only this machine", http.StatusForbidden)

			return
		}

		if !ours(r.Host) {
			http.Error(w, "this answers only to its own name", http.StatusForbidden)

			return
		}

		if r.Method == http.MethodPost && !readable(r.Header.Get("content-type")) {
			http.Error(w, "this reads json", http.StatusUnsupportedMediaType)

			return
		}

		next.ServeHTTP(w, r)
	})
}

func here(address string) bool {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		host = address
	}

	ip := net.ParseIP(host)

	return ip != nil && ip.IsLoopback()
}

func readable(said string) bool {
	kind, _, _ := strings.Cut(said, ";")

	return strings.TrimSpace(strings.ToLower(kind)) == "application/json"
}

// By name as well as by number: localhost is what somebody types.
func ours(host string) bool {
	name, _, err := net.SplitHostPort(host)
	if err != nil {
		name = host
	}

	if name == "localhost" {
		return true
	}

	ip := net.ParseIP(strings.Trim(name, "[]"))

	return ip != nil && ip.IsLoopback()
}

type seen struct {
	URL   string `json:"url"`
	Title string `json:"title"`
}

func (s *Server) seen(w http.ResponseWriter, r *http.Request) {
	var said seen

	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&said); err != nil {
		http.Error(w, "unreadable", http.StatusBadRequest)

		return
	}

	one, ok := play.Recognise(said.URL, said.Title, time.Now())
	if !ok {
		// Not a refusal: the extension reports every tab, and most tabs are
		// not a thing being watched.
		answer(w, map[string]bool{"kept": false})

		return
	}

	if err := s.Store.Add(one); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	answer(w, map[string]any{"kept": true, "service": one.Service})
}

// What a page says about itself while it runs. Seconds with a fraction
// because that is what a media element counts in.
type living struct {
	URL      string  `json:"url"`
	Title    string  `json:"title"`
	By       string  `json:"by"`
	Paused   bool    `json:"paused"`
	Position float64 `json:"position"`
	Length   float64 `json:"length"`
}

// A page knows the name of what it plays; a tab knows the name of the tab.
// "Нечто — YouTube" is a worse line in a list than "Нечто", so this writes the
// play down as well.
func (s *Server) playing(w http.ResponseWriter, r *http.Request) {
	var said living

	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&said); err != nil {
		http.Error(w, "unreadable", http.StatusBadRequest)

		return
	}

	at := time.Now()

	one, ok := play.Recognise(said.URL, said.Title, at)
	if !ok {
		s.Watching.Nothing()
		answer(w, map[string]bool{"kept": false})

		return
	}

	s.Watching.Says(now.Said{
		Play:     one,
		By:       strings.TrimSpace(said.By),
		Paused:   said.Paused,
		Position: seconds(said.Position),
		Length:   seconds(said.Length),
	}, at)

	if err := s.Store.Add(one); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	answer(w, map[string]any{"kept": true, "service": one.Service})
}

func (s *Server) gone(w http.ResponseWriter, r *http.Request) {
	s.Watching.Nothing()

	if err := s.Store.Stop(time.Now()); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	answer(w, map[string]bool{"stopped": true})
}

// A negative position is a page still loading, and a length of a year is a
// stream pretending to have an end.
func seconds(said float64) time.Duration {
	if said <= 0 || said > (30*24*time.Hour).Seconds() {
		return 0
	}

	return time.Duration(said * float64(time.Second))
}

// A Now is the line at the top of the page, and the whole of what the popup in
// the browser has to go on.
type Now struct {
	Playing bool   `json:"playing"`
	Service string `json:"service,omitempty"`
	ID      string `json:"id,omitempty"`
	Title   string `json:"title,omitempty"`
	By      string `json:"by,omitempty"`
	Paused  bool   `json:"paused,omitempty"`

	// Seconds in and seconds long, so the line can say where it stands. Whole
	// is zero for a stream, which has nowhere to stand in.
	Gone  int `json:"gone,omitempty"`
	Whole int `json:"whole,omitempty"`
}

func (s *Server) showing(w http.ResponseWriter, r *http.Request) {
	answer(w, s.onNow())
}

// Shows is how much of the list the page draws. A Takeout import is tens of
// thousands of rows, and the page redraws itself every ten seconds: all of it
// is a page nobody scrolls and a file read for nothing.
const Shows = 200

// A Day breaks the list up so that a date is said once instead of on every row.
type Day struct {
	Date    string
	Seconds int
	Plays   []play.Play
}

func byDay(plays []play.Play) []Day {
	var days []Day

	for _, one := range plays {
		date := one.At.Format("Monday, 2 January")

		if len(days) == 0 || days[len(days)-1].Date != date {
			days = append(days, Day{Date: date})
		}

		days[len(days)-1].Plays = append(days[len(days)-1].Plays, one)
		days[len(days)-1].Seconds += one.Seconds
	}

	return days
}

// Minutes and seconds, and hours only when there are any: 4:07 rather than
// 0:04:07.
func clock(seconds int) string {
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, (seconds%3600)/60, seconds%60)
	}

	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func (s *Server) onNow() Now {
	live, ok := s.Watching.Playing(time.Now())
	if !ok {
		return Now{}
	}

	title := live.Play.Title
	if title == "" {
		title = live.Play.ID
	}

	on := Now{
		Playing: true,
		Service: live.Play.Service,
		ID:      live.Play.ID,
		Title:   title,
		By:      live.By,
		Paused:  live.Paused,
	}

	// A paused thing stands where the page left it. A running one is wherever
	// the clock has carried it since it started.
	if live.Paused {
		on.Gone = int(live.Position.Seconds())
	} else {
		on.Gone = int(time.Since(live.Started).Seconds())
	}

	if !live.Ends.IsZero() {
		on.Whole = int(live.Ends.Sub(live.Started).Seconds())
	}

	return on
}

func (s *Server) plays(w http.ResponseWriter, r *http.Request) {
	plays, err := s.Store.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	answer(w, plays)
}

func (s *Server) week(w http.ResponseWriter, r *http.Request) {
	plays, err := s.Store.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	answer(w, sum.Week(plays, time.Now()))
}

func (s *Server) services(w http.ResponseWriter, r *http.Request) {
	answer(w, play.Services())
}

func (s *Server) show(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)

		return
	}

	plays, err := s.Store.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "text/html; charset=utf-8")

	week := sum.Week(plays, time.Now())
	more := 0

	if len(plays) > Shows {
		more = len(plays) - Shows
		plays = plays[:Shows]
	}

	_ = page.Execute(w, struct {
		Now  Now
		Days []Day
		Week sum.Total
		More int
	}{s.onNow(), byDay(plays), week, more})
}

func answer(w http.ResponseWriter, body any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
