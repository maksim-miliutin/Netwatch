// Package web answers the browser: one page to look at, one address for a
// browser extension to report to.
package web

import (
	_ "embed"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"html/template"
	"net"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"netwatch/internal/mine"
	"netwatch/internal/now"
	"netwatch/internal/play"
	"netwatch/internal/quiet"
	"netwatch/internal/store"
	"netwatch/internal/sum"
)

//go:embed page.html
var pageSource string

//go:embed gone.html
var farewell []byte

//go:embed icon.png
var mark []byte

var page = template.Must(template.New("page").Funcs(template.FuncMap{
	"hours": hours,
	"shown": play.Shown,
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
	Quiet    *quiet.List
	Mine     *mine.List

	// The last thing the card had to say, for the popup. Nil when nobody is
	// telling: the card is off, or this is a test.
	Says func() string

	// What to do when the page asks to stop. Nil in a test, which should not
	// be able to end the run that is testing it.
	Quitting func()

	// The application id the card is on, and what to do when the page hands
	// over another. A dash turns it off.
	Id      string
	Joining func(id string) error
}

func (s *Server) Routes() *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /", s.show)
	mux.HandleFunc("POST /api/seen", s.seen)
	mux.HandleFunc("POST /api/now", s.playing)
	mux.HandleFunc("POST /api/gone", s.gone)
	mux.HandleFunc("GET /api/now", s.showing)
	mux.HandleFunc("POST /api/quiet", s.hiding)
	mux.HandleFunc("POST /api/services", s.adding)
	mux.HandleFunc("GET /api/mine", s.own)
	mux.HandleFunc("POST /api/quit", s.quit)
	mux.HandleFunc("POST /api/discord", s.joining)
	mux.HandleFunc("GET /icon.png", s.icon)
	mux.HandleFunc("POST /api/forget", s.forget)
	mux.HandleFunc("GET /plays.csv", s.sheet)
	mux.HandleFunc("GET /api/plays", s.plays)
	mux.HandleFunc("GET /api/services", s.services)
	mux.HandleFunc("GET /api/week", s.week)

	return mux
}

// Nothing here is reachable from anywhere but this machine, and three things
// are asked rather than one. Where the connection came from; the name it was
// called by, since a name pointed at 127.0.0.1 dials loopback all the same;
// and whether whoever wrote here meant to.
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

		if r.Method == http.MethodPost && !meant(r) {
			http.Error(w, "this reads json, or a form off its own page",
				http.StatusUnsupportedMediaType)

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

// Two ways in: a type a page cannot send across sites without asking, or a
// form the browser says came from here.
func meant(r *http.Request) bool {
	if readable(r.Header.Get("content-type")) {
		return true
	}

	came := r.Header.Get("origin")

	return came == "http://"+r.Host || came == "https://"+r.Host
}

func readable(said string) bool {
	kind, _, _ := strings.Cut(said, ";")

	return strings.TrimSpace(strings.ToLower(kind)) == "application/json"
}

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

type living struct {
	URL      string  `json:"url"`
	Title    string  `json:"title"`
	By       string  `json:"by"`
	Paused   bool    `json:"paused"`
	Position float64 `json:"position"`
	Length   float64 `json:"length"`
}

func (s *Server) playing(w http.ResponseWriter, r *http.Request) {
	var said living

	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 8<<10)).Decode(&said); err != nil {
		http.Error(w, "unreadable", http.StatusBadRequest)

		return
	}

	at := time.Now()

	one, ok := play.Recognise(said.URL, said.Title, at)

	// A player that keeps the track out of the address leaves the page as the
	// only one who knows what is on.
	if !ok {
		one, ok = play.Reported(said.URL, said.Title, at)
	}

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

	Discord string `json:"discord,omitempty"`
}

func (s *Server) icon(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("content-type", "image/png")
	w.Header().Set("cache-control", "max-age=86400")
	_, _ = w.Write(mark)
}

func (s *Server) showing(w http.ResponseWriter, r *http.Request) {
	answer(w, s.onNow())
}

func (s *Server) told() string {
	if s.Says == nil {
		return ""
	}

	return s.Says()
}

func (s *Server) sheet(w http.ResponseWriter, r *http.Request) {
	plays, err := s.Store.All()
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	w.Header().Set("content-type", "text/csv; charset=utf-8")
	w.Header().Set("content-disposition", `attachment; filename="plays.csv"`)

	sheet := csv.NewWriter(w)
	defer sheet.Flush()

	_ = sheet.Write([]string{"at", "service", "id", "seconds", "title", "url"})

	for _, one := range plays {
		_ = sheet.Write([]string{
			one.At.Format(time.RFC3339),
			one.Service,
			one.ID,
			strconv.Itoa(one.Seconds),
			one.Title,
			one.URL,
		})
	}
}

func (s *Server) forget(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "unreadable", http.StatusBadRequest)

		return
	}

	at, err := time.Parse(time.RFC3339Nano, r.FormValue("at"))
	if err != nil {
		http.Error(w, "unreadable", http.StatusBadRequest)

		return
	}

	if err := s.Store.Forget(r.FormValue("service"), r.FormValue("id"), at); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	back(w, r)
}

func (s *Server) joining(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || s.Joining == nil {
		http.Error(w, "unreadable", http.StatusBadRequest)

		return
	}

	asked := strings.TrimSpace(r.FormValue("id"))

	if err := s.Joining(asked); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	s.Id = asked
	if asked == "-" {
		s.Id = ""
	}

	back(w, r)
}

// Quitting is a write, so it goes through the same door as the rest: off
// this page or not at all.
func (s *Server) quit(w http.ResponseWriter, r *http.Request) {
	// A page rather than an answer: the button is pressed by a person, and a
	// person handed raw json reads it as something having gone wrong.
	w.Header().Set("content-type", "text/html; charset=utf-8")
	_, _ = w.Write(farewell)

	if s.Quitting == nil {
		return
	}

	go func() {
		time.Sleep(200 * time.Millisecond)
		s.Quitting()
	}()
}

type Choice struct {
	Name   string
	Shown  string
	Ticked bool
}

type Sort struct {
	Kind    string
	Choices []Choice
}

func (s *Server) choices() []Sort {
	var sorts []Sort

	for _, kind := range play.Kinds() {
		one := Sort{Kind: kind}

		for _, name := range play.Services() {
			if play.Kind(name) != kind {
				continue
			}

			one.Choices = append(one.Choices, Choice{
				Name:   name,
				Shown:  play.Shown(name),
				Ticked: !s.Quiet.Hidden(name),
			})
		}

		if len(one.Choices) > 0 {
			sorts = append(sorts, one)
		}
	}

	return sorts
}

func (s *Server) own(w http.ResponseWriter, r *http.Request) {
	if s.Mine == nil {
		answer(w, map[string]string{})

		return
	}

	answer(w, s.Mine.All())
}

// A site somebody adds brings no rule for reading its addresses, and needs
// none: what it plays is named by the page, the way Yandex Music is.
func (s *Server) adding(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil || s.Mine == nil {
		http.Error(w, "unreadable", http.StatusBadRequest)

		return
	}

	host := strings.TrimSpace(strings.ToLower(r.FormValue("host")))
	host = strings.TrimPrefix(strings.TrimPrefix(host, "https://"), "http://")
	host, _, _ = strings.Cut(host, "/")

	if host == "" {
		back(w, r)

		return
	}

	if err := s.Mine.Add(host, r.FormValue("shown")); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	play.Add(host, r.FormValue("shown"))
	back(w, r)
}

func (s *Server) hiding(w http.ResponseWriter, r *http.Request) {
	if err := r.ParseForm(); err != nil {
		http.Error(w, "unreadable", http.StatusBadRequest)

		return
	}

	shown := map[string]bool{}
	for _, name := range r.Form["show"] {
		shown[name] = true
	}

	var hidden []string

	for _, name := range r.Form["service"] {
		if !shown[name] {
			hidden = append(hidden, name)
		}
	}

	if err := s.Quiet.Hide(hidden); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	back(w, r)
}

// Shows is how much of the list the page draws. All of a Takeout import is
// a page nobody scrolls, redrawn every ten seconds.
const Shows = 200

func elsewhere(span, find string) string {
	asked := url.Values{}
	asked.Set("span", span)

	if find != "" {
		asked.Set("find", find)
	}

	return "/?" + asked.Encode()
}

// Back to where the button was pressed, for the same reason: crossing a line
// out should not throw away the search it was found by.
func back(w http.ResponseWriter, r *http.Request) {
	came := r.Referer()

	if !strings.HasPrefix(came, "http://"+r.Host+"/") &&
		!strings.HasPrefix(came, "https://"+r.Host+"/") {
		came = "/"
	}

	http.Redirect(w, r, came, http.StatusSeeOther)
}

func matching(plays []play.Play, find string) []play.Play {
	find = strings.ToLower(find)

	var found []play.Play

	for _, one := range plays {
		if strings.Contains(strings.ToLower(one.Title), find) ||
			strings.Contains(strings.ToLower(one.Service), find) {
			found = append(found, one)
		}
	}

	return found
}

// The line under the name answers what people open this page to ask: is it
// working. A browser that stopped reporting looks exactly like an evening
// nobody watched anything, and only the date of the last play tells them apart.
func (s *Server) working(plays []play.Play) string {
	said := "Nothing reported yet"

	if len(plays) > 0 {
		said = "Reporting"

		if time.Since(plays[0].At) > 24*time.Hour {
			said = "Nothing reported since " + plays[0].At.Format("2 January")
		}
	}

	if s.Id != "" {
		said += ", card on"
	} else {
		said += ", card off"
	}

	if len(plays) == 1 {
		return said + ", 1 play kept."
	}

	return said + ", " + strconv.Itoa(len(plays)) + " plays kept."
}

type Day struct {
	Date    string
	Seconds int
	Plays   []play.Play
}

func byDay(plays []play.Play, at time.Time) []Day {
	var days []Day

	for _, one := range plays {
		date := dated(one.At, at)

		if len(days) == 0 || days[len(days)-1].Date != date {
			days = append(days, Day{Date: date})
		}

		days[len(days)-1].Plays = append(days[len(days)-1].Plays, one)
		days[len(days)-1].Seconds += one.Seconds
	}

	return days
}

func dated(when, at time.Time) string {
	when, at = when.Local(), at.Local()

	if sameDay(when, at) {
		return "Today"
	}

	if sameDay(when, at.AddDate(0, 0, -1)) {
		return "Yesterday"
	}

	return when.Format("Monday, 2 January")
}

func sameDay(one, other time.Time) bool {
	year, month, day := one.Date()
	otherYear, otherMonth, otherDay := other.Date()

	return year == otherYear && month == otherMonth && day == otherDay
}

func clock(seconds int) string {
	if seconds >= 3600 {
		return fmt.Sprintf("%d:%02d:%02d", seconds/3600, (seconds%3600)/60, seconds%60)
	}

	return fmt.Sprintf("%d:%02d", seconds/60, seconds%60)
}

func (s *Server) onNow() Now {
	live, ok := s.Watching.Playing(time.Now())
	if !ok {
		return Now{Discord: s.told()}
	}

	title := live.Play.Title
	if title == "" {
		title = live.Play.ID
	}

	on := Now{
		Playing: true,
		Discord: s.told(),
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

	working := s.working(plays)
	fresh := len(plays) == 0

	choices := s.choices()

	// A month of watching is a few hundred lines, and a year is thousands. The
	// only way back to a particular one is its name.
	find := strings.TrimSpace(r.URL.Query().Get("find"))
	if find != "" {
		plays = matching(plays, find)
	}

	// Three spans in a ring, and the page names the next one so that finding
	// it takes a click rather than a guess.
	title, next, named := "This week", "month", "the month"
	total := sum.Week(plays, time.Now())

	switch r.URL.Query().Get("span") {
	case "month":
		title, next, named = "This month", "all", "all of it"
		total = sum.Month(plays, time.Now())

	case "all":
		title, next, named = "All of it", "week", "the week"
		total = sum.Over(plays, time.Time{})
	}

	more := 0

	if len(plays) > Shows {
		more = len(plays) - Shows
		plays = plays[:Shows]
	}

	_ = page.Execute(w, struct {
		Now     Now
		Days    []Day
		Total   sum.Total
		Title   string
		Next    string
		Named   string
		Span    string
		More    int
		Find    string
		Working string
		Fresh   bool
		Card    string
		Said    string
		Choices []Sort
	}{s.onNow(), byDay(plays, time.Now()), total, title, elsewhere(next, find),
		named, r.URL.Query().Get("span"), more, find, working, fresh, s.Id,
		s.told(), choices})
}

func answer(w http.ResponseWriter, body any) {
	w.Header().Set("content-type", "application/json")
	_ = json.NewEncoder(w).Encode(body)
}
