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
	"time"

	"netwatch/internal/play"
	"netwatch/internal/store"
	"netwatch/internal/sum"
)

//go:embed page.html
var pageSource string

// Minutes and hours rather than seconds: nobody counts a week in seconds.
var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"hours": hours,
}).Parse(pageSource))

func hours(seconds int) string {
	if seconds < 60 {
		return fmt.Sprintf("%d сек", seconds)
	}

	if seconds < 3600 {
		return fmt.Sprintf("%d мин", seconds/60)
	}

	return fmt.Sprintf("%d ч %d мин", seconds/3600, (seconds%3600)/60)
}

type Server struct {
	Store *store.Store
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.show)
	mux.HandleFunc("POST /api/seen", s.seen)
	mux.HandleFunc("GET /api/plays", s.plays)
	mux.HandleFunc("GET /api/services", s.services)
	mux.HandleFunc("GET /api/week", s.week)

	return mux
}

// Nothing here is reachable from anywhere but this machine, and the check is
// per request rather than only in the address it listens on: a program that
// binds loopback and trusts everything it gets is one setting away from being
// an open door.
func Near(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err != nil {
			host = r.RemoteAddr
		}

		if ip := net.ParseIP(host); ip == nil || !ip.IsLoopback() {
			http.Error(w, "this answers only this machine", http.StatusForbidden)

			return
		}

		next.ServeHTTP(w, r)
	})
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

	_ = page.Execute(w, struct {
		Plays []play.Play
		Week  sum.Total
	}{plays, sum.Week(plays, time.Now())})
}

func answer(w http.ResponseWriter, body any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
