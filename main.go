// netwatch keeps a list of what was watched and listened to on this machine.
//
// Nothing here reads the network: what is watched lives inside TLS. The
// browser knows the name, so a small extension reports it.
package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"netwatch/internal/discord"
	"netwatch/internal/mine"
	"netwatch/internal/now"
	"netwatch/internal/play"
	"netwatch/internal/quiet"
	"netwatch/internal/store"
	"netwatch/internal/takeout"
	"netwatch/internal/web"
)

// The application everybody who runs this uses, unless they put in their own.
// Rich Presence does not care who owns the number: it decides the name Discord
// writes above the card and where the pictures come from, and nothing else.
const Application = "1546852021934751804"

func main() {
	port := flag.Int("port", 7373, "where to listen, on this machine only")
	file := flag.String("file", "", "where to keep the list")
	from := flag.String("import", "", "a watch-history.json out of a Takeout archive")
	alone := flag.Bool("window", true, "open a window of its own on start")
	given := flag.String("discord", "",
		"an application id, to put what plays on a Discord card. off forgets it")
	flag.Parse()

	where, err := chosen(*file)
	if err != nil {
		log.Fatal(err)
	}

	kept, err := store.Open(where)
	if err != nil {
		log.Fatal(err)
	}

	if *from != "" {
		if err := bring(kept, *from); err != nil {
			log.Fatal(err)
		}

		return
	}

	id, err := remembered(*given, filepath.Dir(where))
	if err != nil {
		log.Fatal(err)
	}

	if id == "" {
		id = Application
	}

	hushed, err := quiet.Open(filepath.Join(filepath.Dir(where), "quiet"))
	if err != nil {
		log.Fatal(err)
	}

	written := writing(filepath.Join(filepath.Dir(where), "log"))
	if written != nil {
		defer written.Close()
	}

	own, err := mine.Open(filepath.Join(filepath.Dir(where), "services"))
	if err != nil {
		log.Fatal(err)
	}

	for host, shown := range own.All() {
		play.Add(host, shown)
	}

	watching := now.New()
	said := &latest{}

	showing := &card{
		watching: watching,
		hidden:   hushed.Hidden,
		told: func(text string) {
			said.keep(text)
			note(written, text)
		},
	}

	server := &web.Server{
		Store:    kept,
		Watching: watching,
		Quiet:    hushed,
		Mine:     own,
		Says:     said.last,
		Quitting: func() { os.Exit(0) },
		Id:       id,
		Joining: func(asked string) error {
			kept, err := remembered(asked, filepath.Dir(where))
			if err != nil {
				return err
			}

			showing.to(kept)

			return nil
		},
	}
	address := fmt.Sprintf("127.0.0.1:%d", *port)

	// The port is taken before anything is said about it. Saying it first and
	// failing after leaves two cheerful lines above the reason nothing works.
	ear, err := net.Listen("tcp", address)
	if err != nil {
		// Already running: somebody who closed the window wants it back, not two
		// of these.
		if awake(address) {
			note(written, "netwatch is already running.")

			if *alone {
				if err := window("http://" + address); err != nil {
					note(written, err.Error())
				}
			}

			return
		}

		log.Fatal(err)
	}

	note(written, fmt.Sprintf("netwatch is running. Open http://%s in a browser.", address))
	note(written, fmt.Sprintf("The list is kept in %s and goes nowhere else.", where))

	// The card is the one thing here that leaves the machine, so it runs only
	// for somebody who went and got an application id for it.
	if id != "" {
		showing.to(id)
	}

	if *alone {
		if err := window("http://" + address); err != nil {
			note(written, err.Error())
		}
	}

	log.Fatal(http.Serve(ear, web.Near(server.Routes())))
}

func bring(kept *store.Store, from string) error {
	file, err := os.Open(from)
	if err != nil {
		return err
	}
	defer file.Close()

	plays, skipped, err := takeout.Read(file)
	if err != nil {
		return err
	}

	added, err := kept.Merge(plays)
	if err != nil {
		return err
	}

	fmt.Printf("Read %d, kept %d new.\n", len(plays), added)

	if skipped > 0 {
		fmt.Printf("Passed over %d: taken down, or a search rather than a watch.\n",
			skipped)
	}

	if added == 0 && len(plays) > 0 {
		fmt.Println("All of it was already here, which is what a second run should do.")
	}

	return nil
}

func awake(address string) bool {
	client := http.Client{Timeout: 2 * time.Second}

	answer, err := client.Get("http://" + address + "/api/now")
	if err != nil {
		return false
	}
	defer answer.Body.Close()

	return answer.StatusCode == http.StatusOK
}

// Windows can start a program with no console, and then anything printed
// goes nowhere. log is pointed at the same file.
func writing(file string) *os.File {
	written, err := os.OpenFile(file, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		return nil
	}

	log.SetOutput(io.MultiWriter(os.Stderr, written))

	return written
}

func note(written io.Writer, text string) {
	fmt.Println(text)

	if written != nil {
		fmt.Fprintf(written, "%s %s\n", time.Now().Format(time.RFC3339), text)
	}
}

// A card that can be turned on and off while running. Until this, the id came
// from a flag, and a program started by double clicking is handed no flags at
// all: the exe somebody downloads could not be joined to Discord by any means.
type card struct {
	mu   sync.Mutex
	stop context.CancelFunc

	watching *now.Watch
	hidden   func(string) bool
	told     func(string)
}

func (c *card) to(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stop != nil {
		c.stop()
		c.stop = nil
	}

	if id == "" {
		c.told("The card is off.")

		return
	}

	ctx, stop := context.WithCancel(context.Background())
	c.stop = stop

	go discord.Follow(ctx, id, c.watching, c.hidden, c.told)
}

type latest struct {
	mu   sync.Mutex
	text string
}

func (l *latest) keep(text string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	l.text = text
}

func (l *latest) last() string {
	l.mu.Lock()
	defer l.mu.Unlock()

	return l.text
}

func remembered(asked, dir string) (string, error) {
	file := filepath.Join(dir, "discord")

	// An id is a number, so the word cannot be one.
	if asked == "off" || asked == "-" {
		if err := os.Remove(file); err != nil && !os.IsNotExist(err) {
			return "", err
		}

		return "", nil
	}

	if asked != "" {
		return asked, os.WriteFile(file, []byte(asked+"\n"), 0o600)
	}

	kept, err := os.ReadFile(file)
	if os.IsNotExist(err) {
		return "", nil
	}

	return strings.TrimSpace(string(kept)), err
}

func chosen(asked string) (string, error) {
	if asked != "" {
		return asked, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return filepath.Join(home, ".netwatch", "plays.jsonl"), nil
}
